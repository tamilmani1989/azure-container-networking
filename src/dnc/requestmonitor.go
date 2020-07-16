package main

import (
	"bytes"
	"dnc/aiwrapper"
	"dnc/allocationhelper"
	"dnc/api"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/nmagentcontracts"
	"dnc/utils"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-container-networking/aitelemetry"
	"github.com/Azure/azure-container-networking/cns"
	"github.com/Azure/azure-container-networking/cns/restserver"
)

var (
	initiateMigration = true
	containerTypes    sync.Map
)

const (
	// NMAgent using 169.254.0.0 to 169.254.3.255. So allocating a range not conflicting with that.
	localIPSubnetAddressString            = "169.254.128.0/17"
	imdsIPSubnetString                    = "169.254.169.254/32"
	localIPSubnetPrefixLength             = 17
	unauthorizedErrRetryDurationSec       = 60
	defaultMaxConcurrentRequestsToProcess = 1000
	defaultContentType                    = "application/json; charset=UTF-8"
)

func processRequest(requestID string, requestContext *datamodel.RequestContext) {
	var (
		removeWorkItem bool
		err            *api.JsonErr
	)

	// Update retry/request interval (by doubling it) unless it's greater than max back off interval
	if requestContext.RetryInterval <= maxRequestBackoffInterval {
		requestContext.RetryInterval *= 2
	}

	switch requestContext.Type {
	case datamodel.VirtualNetworkRequest:
		// Acquire the NW lock while the current request completes the pending queue processing
		sessionInfo := map[string]string{datamodel.NetworkRepositoryName: requestContext.NetworkID}
		sessionManager.BeginWriteSession(sessionInfo)
		defer sessionManager.EndWriteSession(sessionInfo)

		removeWorkItem, err = processVirtualNetworkRequest(requestContext)

		break

	case datamodel.ReservationRequest:
		// Keep the NW and reservation lock while the current request completes the pending queue processing
		sessionInfo := map[string]string{
			datamodel.ReservationRepositoryName: datamodel.GetReservationIDFromKey(requestID),
			datamodel.NetworkRepositoryName:     requestContext.NetworkID,
		}
		sessionManager.BeginWriteSession(sessionInfo)
		defer sessionManager.EndWriteSession(sessionInfo)

		removeWorkItem, err = ipAllocator.ProcessIPReservationAllocationWithKey(requestID)
		break

	case datamodel.NetworkContainerRequest:
		// Acquire the NC lock while the current request completes the pending queue processing
		sessionInfo := map[string]string{
			datamodel.NCRepositoryName: datamodel.GetNCIdFromKey(requestID),
		}
		sessionManager.BeginWriteSession(sessionInfo)
		defer sessionManager.EndWriteSession(sessionInfo)

		removeWorkItem, err = processNetworkContainerRequest(requestContext, requestID, sessionInfo)
		if err != nil && err.HTTPStatus == http.StatusProcessing {
			requestContext.RetryInterval = requestContextRetryInterval
		}
		break

	}

	if err == nil && removeWorkItem {
		datamodel.RemovePendingRequest(requestID)
	} else {
		if err != nil {
			requestContext.LastRetriedDateTime = time.Now().UTC()
		} else {
			requestContext.LastRetriedDateTime = time.Time{}
			requestContext.RetryInterval = requestContextRetryInterval
			requestContext.StateTransitionDateTime = time.Now().UTC()
		}

		logger.Errorf("Adding request: %+v back in pending queue due to error: %v", requestContext, err)
		datamodel.AddRequestToPendingQueue(requestID, requestContext)
	}
}

// Decode the response received for join network request.
// Returns IsRetirbale, optional error if any, isThrottled
func decodeNetworkJoinResponse(
	networkID string,
	joinResponse *http.Response,
	joinErr error) (bool, *api.JsonErr, bool) {
	if joinErr != nil {
		msg := fmt.Sprintf(
			"Join for network [%s] failed with reponse error [%s]. Marking this error as retriable",
			networkID,
			joinErr.Error())
		return true, &api.JsonErr{
			Code:       api.JoinDenied,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, false
	}

	defer joinResponse.Body.Close()

	if joinResponse.StatusCode == http.StatusProcessing {
		msg := fmt.Sprintf(
			"Join for network [%s] is pending. Marking this error as retriable",
			networkID)
		return true, &api.JsonErr{
			Code:       api.JoinPending,
			Text:       msg,
			HTTPStatus: http.StatusProcessing}, false
	}

	if joinResponse.StatusCode == http.StatusUnauthorized {
		msg := fmt.Sprintf("Join for network [%s] is unauthorized. Marking this error as non retriable.", networkID)
		return false, &api.JsonErr{
			Code:       api.NetworkJoinFailed,
			Text:       msg,
			HTTPStatus: http.StatusUnauthorized}, false
	}

	if joinResponse.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Join for network [%s] failed with status [%d]. Marking this error as retriable.",
			networkID,
			joinResponse.StatusCode)
		return true, &api.JsonErr{
			Code:       api.NetworkJoinFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, joinResponse.StatusCode == http.StatusForbidden
	}

	return false, nil, false
}

// Post request to host nmagent to express the intent for joining a network.
// Returns IsRetriable, optional error if any, isThrottled
func postRequestToAgentForNetworkJoin(networkID string) (bool, *api.JsonErr, bool) {
	joinURL := fmt.Sprintf(
		config.HostAgentSettings.ConnectionStrings.JoinNetworkURLFmt,
		config.HostAgentSettings.WireServerIP,
		networkID)

	loggershim.LS.Debugf("[NMAgent] joining network [%s]", networkID)
	// Empty body is required as wireserver cannot handle a post without the body.
	var body bytes.Buffer
	json.NewEncoder(&body).Encode("")
	joinResponse, joinRespErr := httpClient.Post(joinURL, "application/json", &body)

	return decodeNetworkJoinResponse(networkID, joinResponse, joinRespErr)
}

// Retrieve network configuration from host agent.
func getNetworkConfigurationFromAgent(networkID string) (bool, *api.JsonErr, *nmagentcontracts.VirtualNetworkResponse, bool) {
	getURL := fmt.Sprintf(
		config.HostAgentSettings.ConnectionStrings.GetNetworkConfigurationURLFmt,
		config.HostAgentSettings.WireServerIP,
		networkID)

	getResponse, getRespErr := httpClient.Get(getURL)
	if getRespErr != nil {
		msg := fmt.Sprintf(
			"Retrieval of network configuration for network [%s] failed with error [%s]. Marking this error as retriable.",
			networkID,
			getRespErr.Error())
		return true, &api.JsonErr{
			Code:       api.NetworkConfigurationRetrievalFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil, false
	}

	defer getResponse.Body.Close()

	if getResponse.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Get network configuration call for network %s failed with status[%d].",
			networkID,
			getResponse.StatusCode)
		return true, &api.JsonErr{
			Code:       api.NetworkJoinFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil, getResponse.StatusCode == http.StatusForbidden
	}

	var networkConfigurationResponse nmagentcontracts.VirtualNetworkResponse
	decodeRespErr := utils.DecodeResponse(getResponse, &networkConfigurationResponse)
	if decodeRespErr != nil {
		msg := fmt.Sprintf("Failed to decode get virtual network configuration response for network %s due to error %s. "+
			"Marking this error as retriable.",
			networkID,
			decodeRespErr.Error())
		return true, &api.JsonErr{
			Code:       api.DecodeResponseFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil, false
	}

	logger.Debugf("[NMAgent][Response] for network %s configuration:%s", networkID, networkConfigurationResponse.String())
	getNetworkResponseCode, parseErr := strconv.ParseInt(networkConfigurationResponse.ResponseCode, 10, 64)
	if parseErr != nil {
		msg := fmt.Sprintf(
			"Failed to convert response code %s  to integer when parsing network configuration response for network %s. "+
				"Marking this error as retriable.",
			networkConfigurationResponse.ResponseCode,
			networkID)
		return true, &api.JsonErr{
			Code:       api.ParseResponseCodeFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil, false
	}

	if getNetworkResponseCode == http.StatusProcessing {
		msg := fmt.Sprintf(
			"GetNetworkConfiguration for network %s is pending", networkID)
		return true, &api.JsonErr{
			Code:       api.ProcessingPending,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil, false
	}

	if getNetworkResponseCode == http.StatusUnauthorized {
		msg := fmt.Sprintf("Join for network %s is unauthorized", networkID)
		return false, &api.JsonErr{
			Code:       api.NetworkJoinFailed,
			Text:       msg,
			HTTPStatus: http.StatusUnauthorized}, nil, false
	}

	if getNetworkResponseCode != http.StatusOK {
		msg := fmt.Sprintf("Failed to retrieve network due to error %d", getNetworkResponseCode)
		return true, &api.JsonErr{
			Code:       api.NetworkJoinFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil, false
	}

	return false, nil, &networkConfigurationResponse, false
}

func validateAndRetrieveNetworkConfigurationUpdate(
	repoNetworkConfig *datamodel.NetworkConfiguration,
	networkConfigResponse *nmagentcontracts.VirtualNetworkResponse) (
	bool,
	*api.JsonErr,
	map[string]nmagentcontracts.Subnet,
	map[string]bool, map[string]*datamodel.SubnetConfiguration) {
	currentSubnets := make(map[string]*datamodel.SubnetConfiguration)

	var currentDNSServers []string
	var updatedDNSServers []string
	if len(networkConfigResponse.DNSServers) != 0 {
		updatedDNSServers = strings.Split(networkConfigResponse.DNSServers, ",")
	} else {
		updatedDNSServers = make([]string, 0)
	}

	if len(repoNetworkConfig.DNSServers) != 0 {
		currentDNSServers = repoNetworkConfig.DNSServers
	} else {
		currentDNSServers = make([]string, 0)
	}

	hasDNSServersChanged := !utils.AreSetsEqual(currentDNSServers, updatedDNSServers)

	var currentCnetSpaces []string
	var updatedCnetSpaces []string
	if len(networkConfigResponse.CNetSpace) != 0 {
		updatedCnetSpaces = strings.Split(networkConfigResponse.CNetSpace, ",")
	} else {
		updatedCnetSpaces = make([]string, 0)
	}

	if len(repoNetworkConfig.CNetSpace) != 0 {
		currentCnetSpaces = strings.Split(repoNetworkConfig.CNetSpace, ",")
	} else {
		currentCnetSpaces = make([]string, 0)
	}

	hasCnetSpacesChanged := !utils.AreSetsEqual(currentCnetSpaces, updatedCnetSpaces)

	// Retrieve the current subnets from the repository.
	for subnetName := range repoNetworkConfig.Subnets {
		subnetConfiguration, subnetCfgErr := datamodel.TryQuerySubnetConfiguration(repoNetworkConfig.NetworkID, subnetName)
		if subnetCfgErr != nil {
			return (subnetCfgErr.Code == api.StatusInternalServerError), subnetCfgErr, nil, nil, nil
		}

		if subnetConfiguration != nil {
			currentSubnets[subnetName] = subnetConfiguration
		}
	}

	subnetsToRemove := make(map[string]bool)
	subnetsToAdd := make(map[string]nmagentcontracts.Subnet)
	subnetsToUpdate := make(map[string]*datamodel.SubnetConfiguration)
	subnetNamesInResponse := make(map[string]bool)

	// Iterate through all the subnets in the response.
	for _, subnet := range networkConfigResponse.Subnets {
		subnetNamesInResponse[subnet.SubnetName] = true
		repoSubnetConfig, exist := currentSubnets[subnet.SubnetName]

		if exist && repoSubnetConfig != nil {
			// If the subnet exist in the repository and there are no allocations
			// (i.e) no reservations exist, then mark the subnet for update.
			if len(repoSubnetConfig.Reservations) == 0 {
				subnetsToUpdate[subnet.SubnetName] = repoSubnetConfig
				continue
			}

			// If reservations exists, then changing the IPSubnet is not allowed.
			// This should never happen.
			updatedSubnet, err := utils.ParseIPNet(subnet.AddressPrefix)
			if err != nil {
				msg := fmt.Sprintf("Failed to parse IPSubnet %s from received config %s",
					subnet.AddressPrefix,
					networkConfigResponse.String())
				return false, &api.JsonErr{
					Code:       api.NetworkJoinFailed,
					Text:       msg,
					HTTPStatus: http.StatusInternalServerError}, nil, nil, nil
			}

			currentSubnet, err := utils.ParseIPNet(repoSubnetConfig.IPSubnet)
			if err != nil {
				msg := fmt.Sprintf("Failed to parse IPSubnet %s from repo config %s.",
					subnet.AddressPrefix,
					repoSubnetConfig.String())
				return false, &api.JsonErr{
					Code:       api.NetworkJoinFailed,
					Text:       msg,
					HTTPStatus: http.StatusInternalServerError}, nil, nil, nil
			}

			if !utils.AreIPSubnetsEqual(currentSubnet, updatedSubnet) {
				msg := fmt.Sprintf(
					"Update of subnet %s is not allowed as there are allocations. Current subnet:%s, Updated subnet:%s.",
					repoSubnetConfig.SubnetName,
					currentSubnet.String(),
					updatedSubnet.String())
				return false, &api.JsonErr{
					Code:       api.NetworkJoinFailed,
					Text:       msg,
					HTTPStatus: http.StatusInternalServerError}, nil, nil, nil
			}

			needsUpdate := hasDNSServersChanged || hasCnetSpacesChanged
			if !needsUpdate {
				var currentSTTags []string
				var updatedSTTags []string
				if len(subnet.Tags) != 0 {
					updatedSTTags = make([]string, len(subnet.Tags))
					for index, tag := range subnet.Tags {
						updatedSTTags[index] = tag.Name
					}
				} else {
					updatedSTTags = make([]string, 0)
				}

				if len(repoSubnetConfig.Tags) != 0 {
					currentSTTags = make([]string, len(repoSubnetConfig.Tags))
					for index, tag := range repoSubnetConfig.Tags {
						currentSTTags[index] = tag.Name
					}
				} else {
					currentSTTags = make([]string, 0)
				}

				needsUpdate = !utils.AreSetsEqual(currentSTTags, updatedSTTags)
			}

			if needsUpdate {
				subnetsToUpdate[subnet.SubnetName] = repoSubnetConfig
			}
		} else {
			// This is a new subnet and mark it to be added to the repository.
			subnetsToAdd[subnet.SubnetName] = subnet
		}
	}

	// Cleanup all the subnets that are no longer present in the received configuration.
	for subnetName, repoSubnetConfig := range currentSubnets {
		_, exists := subnetNamesInResponse[subnetName]
		// If a subnet has been deleted outside of DNC,
		// delete it only if there are no resources in them
		if !exists {
			if len(repoSubnetConfig.Reservations) > 0 {
				logger.Errorf("Error: Leaked NC's detected: Subnet %s in network %s is removed but there are allocations.",
					repoSubnetConfig.SubnetName,
					repoSubnetConfig.NetworkID)
			}

			subnetsToRemove[subnetName] = true
		}
	}

	return false, nil, subnetsToAdd, subnetsToRemove, subnetsToUpdate
}

// Routine that processes virtual network request.
func processVirtualNetworkRequest(requestContext *datamodel.RequestContext) (bool, *api.JsonErr) {
	repoNetworkConfiguration, err := datamodel.TryQueryNetworkConfiguration(requestContext.NetworkID)
	if err != nil {
		return false, err
	}

	if repoNetworkConfiguration == nil {
		logger.Debugf("Failed to find configuration in repository for network %s\n", requestContext.NetworkID)
		return true, nil
	}

	switch repoNetworkConfiguration.Status {
	case datamodel.Pending:
		return processVirtualNetworksPendingJoin(requestContext.NetworkID)

	case datamodel.Refreshing:
		_, success := refreshNetworkConfiguration(datamodel.GetNetworkKey(requestContext.NetworkID), true)
		if !success {
			return false, &api.JsonErr{
				Code:       api.NetworkJoinFailed,
				Text:       fmt.Sprintf("VirtualNetworkRequest for network [%s] refresh failed. Marking as Retriable error", requestContext.NetworkID),
				HTTPStatus: http.StatusInternalServerError}
		}
	}

	return true, nil
}

// Routine that processes join network request.
func processVirtualNetworksPendingJoin(networkID string) (bool, *api.JsonErr) {
	var failureDetail *api.JsonErr
	var isRetriableError bool
	var networkResponse *nmagentcontracts.VirtualNetworkResponse
	var subnetsToAdd map[string]nmagentcontracts.Subnet
	var subnetsToRemove map[string]bool
	var subnetsToUpdate map[string]*datamodel.SubnetConfiguration
	var subnetsInUpdatedResponse map[string]nmagentcontracts.Subnet
	var ipSubnetMap map[string]string
	var repoNetworkConfiguration *datamodel.NetworkConfiguration
	var vnetVersion uint64
	var dnsServers []string

	// Network configuration is expected to be present as we will always
	// add a dummy configuration and mark it as pending processing once
	// a request is received.
	repoNetworkConfiguration, err := datamodel.TryQueryNetworkConfiguration(networkID)
	if err != nil {
		return false, err
	}

	if repoNetworkConfiguration == nil {
		logger.Debugf("Failed to find configuration in repository for network %s", networkID)
		return true, nil
	}

	isRetriableError, failureDetail, _ = postRequestToAgentForNetworkJoin(networkID)
	if failureDetail != nil {
		if isRetriableError {
			return false, failureDetail
		}
		goto Cleanup
	}

	isRetriableError, failureDetail, networkResponse, _ = getNetworkConfigurationFromAgent(networkID)
	if failureDetail != nil {
		if isRetriableError {
			return false, failureDetail
		}
		goto Cleanup
	}

	isRetriableError, failureDetail, subnetsToAdd, subnetsToRemove, subnetsToUpdate =
		validateAndRetrieveNetworkConfigurationUpdate(
			repoNetworkConfiguration,
			networkResponse)

	if failureDetail != nil {
		if isRetriableError {
			return false, failureDetail
		}
		goto Cleanup
	}

	for subnetName := range subnetsToRemove {
		failureDetail = datamodel.DeleteSubnetConfiguration(networkID, subnetName)
		if failureDetail != nil {
			goto Cleanup
		}
	}

	ipSubnetMap = make(map[string]string)
	subnetsInUpdatedResponse = make(map[string]nmagentcontracts.Subnet)
	for _, subnet := range networkResponse.Subnets {
		subnetsInUpdatedResponse[subnet.SubnetName] = subnet
		ipSubnetMap[subnet.SubnetName] = datamodel.GetSubnetKey(networkID, subnet.SubnetName)
	}

	// Commiting virtual network config in two phases.
	// First phase, update only the configuration and the subnet
	// list and the state is still pending join.
	// This helps in driving cleanup as we have handle to all the subnets
	// and we can handle any failures that happens when persisting subnet
	// configurations.
	if len(networkResponse.DNSServers) == 0 {
		dnsServers = make([]string, 0)
	} else {
		dnsServers = strings.Split(networkResponse.DNSServers, ",")
	}

	if len(networkResponse.DNSServers) == 0 {
		dnsServers = make([]string, 0)
	} else {
		dnsServers = strings.Split(networkResponse.DNSServers, ",")
	}

	vnetVersion, _ = strconv.ParseUint(networkResponse.VnetVersion, 10, 64)
	repoNetworkConfiguration = &datamodel.NetworkConfiguration{
		NetworkID:  networkID,
		Version:    vnetVersion,
		CNetSpace:  networkResponse.CNetSpace,
		VNetSpace:  networkResponse.VNetSpace,
		Subnets:    ipSubnetMap,
		DNSServers: dnsServers}

	failureDetail = datamodel.PersistNetworkConfiguration(repoNetworkConfiguration)
	if failureDetail != nil {
		goto Cleanup
	}

	for subnetName := range subnetsToAdd {
		updatedSubnetCfg := subnetsInUpdatedResponse[subnetName]
		var tags []nmagentcontracts.Tag

		if updatedSubnetCfg.Tags == nil || len(updatedSubnetCfg.Tags) == 0 {
			tags = make([]nmagentcontracts.Tag, 0)
		} else {
			tags = updatedSubnetCfg.Tags
		}

		subnetConfig := datamodel.SubnetConfiguration{
			NetworkID:         networkID,
			SubnetName:        subnetName,
			IPSubnet:          updatedSubnetCfg.AddressPrefix,
			IsDelegatedSubnet: true,
			Gateway:           updatedSubnetCfg.DefaultGateway,
			Tags:              tags,
			UpdateStatus:      datamodel.SubnetUpdateCompleted}
		failureDetail = datamodel.PutSubnetConfiguration(&subnetConfig)
		if failureDetail != nil {
			goto Cleanup
		}
	}

	for subnetName, currentRepoSubnetCfg := range subnetsToUpdate {
		ipSubnetMap[subnetName] = datamodel.GetSubnetKey(networkID, subnetName)
		updatedSubnetCfg := subnetsInUpdatedResponse[subnetName]
		currentRepoSubnetCfg.Tags = updatedSubnetCfg.Tags
		failureDetail = datamodel.PutSubnetConfiguration(currentRepoSubnetCfg)
		if failureDetail != nil {
			goto Cleanup
		}

		// Register tags (add them to cache) if subnet is delegated
		if currentRepoSubnetCfg.IsDelegatedSubnet {
			for _, tag := range currentRepoSubnetCfg.Tags {
				if _, err := RegisterAndGetTagDefinition(tag); err != nil {
					logger.Errorf("Failed to register and get tag definitions with error: %+v", err)
				}
			}
		}
	}

	// Mark the join operation as completed.
	repoNetworkConfiguration.NetworkState = datamodel.Join
	repoNetworkConfiguration.Status = datamodel.Completed
	failureDetail = datamodel.PersistNetworkConfiguration(repoNetworkConfiguration)

Cleanup:
	// If it is an unknown failure then we return with marking the
	// operation has failed so we can retry later.
	if failureDetail != nil {
		if failureDetail.Code == api.StatusInternalServerError {
			return false, err
		}

		// Non retriable errors, mark the operation as failed.
		repoNetworkConfiguration.Status = datamodel.Failed
		repoNetworkConfiguration.FailureDetail = *failureDetail
		if err = datamodel.PersistNetworkConfiguration(repoNetworkConfiguration); err != nil {
			return false, err
		}
	}

	return true, nil
}

func getAllocatedLocalAddressesOnNode(nodeID string) (map[string]*net.IPNet, *api.JsonErr) {
	nodeConfiguration, err := datamodel.QueryNodeConfiguration(nodeID)
	if err != nil {
		return nil, err
	}

	allocatedLocalAddresses := make(map[string]*net.IPNet)

	if nodeConfiguration.DeployedNetworkContainerMap != nil {
		for networkID, ncMap := range nodeConfiguration.DeployedNetworkContainerMap {
			for networkContainerID := range ncMap {
				repoNetworkContainerConfiguration, err := datamodel.TryQueryNetworkContainer(
					networkID,
					networkContainerID)
				if err != nil {
					return nil, err
				}

				if repoNetworkContainerConfiguration == nil {
					logger.Debugf("Configuration for network container %s not found.",
						networkContainerID)
					continue
				}

				if repoNetworkContainerConfiguration.LocalIP != "" {
					localIPSubnetString := repoNetworkContainerConfiguration.LocalIP + utils.PrefixDelimiter + utils.IPV4DefaultPrefix
					localIPSubnet, _ := utils.ParseIPNet(localIPSubnetString)
					allocatedLocalAddresses[localIPSubnetString] = localIPSubnet
				}
			}
		}
	}

	return allocatedLocalAddresses, nil
}

// This function allocate a local ip from 169.254.128.0/17 address space
func processAllocateLocalIP(
	repoNCConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	var (
		markProcessingFailed bool
		allocatedIP          *net.IPNet
		failureDetail        *api.JsonErr
		allocationErr        error
	)

	nodeID := repoNCConfig.GetAssociationInfo().NodeID

	sessionInfo := map[string]string{datamodel.NodeRepositoryName: nodeID}
	sessionManager.BeginWriteSession(sessionInfo)
	defer sessionManager.EndWriteSession(sessionInfo)

	// get all allocated local addresses for a node
	allocatedLocalIPAddresses, err := getAllocatedLocalAddressesOnNode(nodeID)
	if err != nil {
		return false, err
	}

	// ignore 169.254.169.254 ip
	_, imdsSubnet, _ := net.ParseCIDR(imdsIPSubnetString)
	allocatedLocalIPAddresses[imdsIPSubnetString] = imdsSubnet

	_, localIPSubnet, _ := net.ParseCIDR(localIPSubnetAddressString)

	//ignore first few addresses on subnet
	reservedIPSubnet, addrErr := allocationhelper.GetReservedAddressesInSubnet(localIPSubnet)
	if addrErr != nil {
		markProcessingFailed = true
		failureDetail = &api.JsonErr{
			Code:       api.StatusInternalServerError,
			HTTPStatus: http.StatusInternalServerError,
			Text:       fmt.Sprintf(addrErr.Error()),
		}
		// todo: this function needs to be cleaned up further to not rely on gotos
		goto Cleanup
	}

	// join reservedIP map and allocatedLocalIP map
	for subnetString, subnet := range reservedIPSubnet {
		allocatedLocalIPAddresses[subnetString] = subnet
	}

	allocatedIP, allocationErr = allocationhelper.AllocateSubnetFromNetworkSpace(localIPSubnet, allocatedLocalIPAddresses, 32)
	if allocationErr != nil {
		markProcessingFailed = true
		failureDetail = &api.JsonErr{
			Code:       api.StatusInternalServerError,
			HTTPStatus: http.StatusInternalServerError,
			Text: fmt.Sprintf(
				"Failed to allocate address from node local address space due to error:%s", allocationErr.Error())}
		goto Cleanup
	}

	repoNCConfig.LocalIP = allocatedIP.IP.String()
	repoNCConfig.State = datamodel.Associate
	repoNCConfig.Status = datamodel.Pending
	repoNCConfig.FailureDetail = api.JsonErr{}

Cleanup:

	if markProcessingFailed {
		repoNCConfig.Status = datamodel.Failed
		repoNCConfig.FailureDetail = *failureDetail
	}

	if err = datamodel.PersistNCConfiguration(repoNCConfig, datamodel.DontAddToPendingQ); err != nil {
		return false, err
	}

	return markProcessingFailed, nil
}

func processNetworkContainerReserve(
	repoNCConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	var (
		err                  *api.JsonErr
		failureDetail        *api.JsonErr
		markProcessingFailed bool
		repoReservation      *datamodel.ReservationConfiguration
	)

	sessionInfo := map[string]string{
		datamodel.ReservationRepositoryName: repoNCConfig.ReservationID,
		datamodel.NetworkRepositoryName:     repoNCConfig.NetworkID,
	}

	// Acquire Node lock if the ContainerType is WebApps
	if repoNCConfig.GetAssociationInfo().ContainerType == cns.WebApps {
		sessionInfo[datamodel.NodeRepositoryName] = repoNCConfig.GetAssociationInfo().NodeID
	}

	sessionManager.BeginWriteSession(sessionInfo)
	defer sessionManager.EndWriteSession(sessionInfo)

	repoReservation, err = datamodel.TryQueryIPReservation(
		repoNCConfig.NetworkID,
		repoNCConfig.ReservationID)
	if err != nil {
		return false, err
	}

	if repoReservation == nil {
		markProcessingFailed = true
		failureDetail = &api.JsonErr{
			Code:       api.ReservationNotFound,
			HTTPStatus: http.StatusBadRequest,
			Text: fmt.Sprintf(
				"Failed to find reservation %s associated with network container %s",
				repoNCConfig.ReservationID,
				repoNCConfig.NetworkContainerID)}

		goto Cleanup
	}

	// If the reservation status is pending, process IP allocation
	if repoReservation.Status == datamodel.Pending {
		ipReservationAllocationComplete, err := ipAllocator.ProcessIPReservationAllocation(repoReservation)

		// Return if the IP reservation allocation did not complete. This indicates that
		// the operation ran into retriable error.
		if !ipReservationAllocationComplete {
			return false, err
		}
	}

	if repoReservation.Status == datamodel.Completed {
		repoNCConfig.State = datamodel.AllocateLocalIP
		repoNCConfig.Status = datamodel.Pending
		repoNCConfig.FailureDetail = api.JsonErr{}

		repoSubnetConfiguration, err := datamodel.TryQuerySubnetConfiguration(
			repoNCConfig.NetworkID,
			repoReservation.AllocationRequest.SubnetName)
		if err != nil {
			return true, err
		}
		if repoSubnetConfiguration == nil {
			msg := fmt.Sprintf(
				"Configuration for Subnet %s in Network %v not found.",
				repoReservation.AllocationRequest.SubnetName,
				repoNCConfig.NetworkID)
			markProcessingFailed = true
			failureDetail = &api.JsonErr{
				Code:       api.SubnetNotFound,
				Text:       msg,
				HTTPStatus: http.StatusBadRequest}
		}

		repoNCConfig.ProgrammedTagsInfo = make(map[string]uint64)

		for _, tag := range repoSubnetConfiguration.Tags {
			tagsDefinitionKey := datamodel.GetTagDefintionKey(tag.Type, tag.Name)
			repoNCConfig.ProgrammedTagsInfo[tagsDefinitionKey] = 0
		}
	}

	if repoReservation.Status == datamodel.Failed {
		markProcessingFailed = true
		failureDetail = &repoReservation.FailureDetail
	}

Cleanup:
	if markProcessingFailed {
		repoNCConfig.Status = datamodel.Failed
		repoNCConfig.FailureDetail = *failureDetail
	}

	if err = datamodel.PersistNCConfiguration(repoNCConfig, datamodel.DontAddToPendingQ); err != nil {
		return false, err
	}

	return markProcessingFailed, nil
}

func processNetworkContainerUnReserve(
	repoNCConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	var err *api.JsonErr
	if repoNCConfig.ImplicitReservation {
		sessionInfo := map[string]string{
			datamodel.ReservationRepositoryName: repoNCConfig.ReservationID,
			datamodel.NetworkRepositoryName:     repoNCConfig.NetworkID,
		}

		// Acquire Node lock only if needed
		reservationConfig, err :=
			datamodel.TryQueryIPReservation(repoNCConfig.NetworkID, repoNCConfig.ReservationID)
		if err == nil && reservationConfig != nil {
			if len(reservationConfig.AllocationRequest.NodeConstraint) != 0 {
				sessionInfo[datamodel.NodeRepositoryName] = repoNCConfig.GetAssociationInfo().NodeID
			}
		}

		sessionManager.BeginWriteSession(sessionInfo)
		defer sessionManager.EndWriteSession(sessionInfo)

		err = datamodel.DeleteIPReservation(repoNCConfig.NetworkID, repoNCConfig.ReservationID, true)
	} else {
		sessionInfo := map[string]string{
			datamodel.ReservationRepositoryName: repoNCConfig.ReservationID,
		}
		sessionManager.BeginWriteSession(sessionInfo)
		defer sessionManager.EndWriteSession(sessionInfo)

		err = datamodel.UpdateNCToReservationAssociation(repoNCConfig, true)
	}
	if err != nil {
		return false, err
	}

	repoNCConfig.State = datamodel.Purge
	repoNCConfig.Status = datamodel.Pending
	repoNCConfig.FailureDetail = api.JsonErr{}

	err = datamodel.PersistNCConfiguration(repoNCConfig, datamodel.DontAddToPendingQ)
	if err != nil {
		return false, err
	}

	return false, nil
}

func processNetworkContainerPurge(
	repoNCConfig *datamodel.NCConfiguration,
	createdDateTime time.Time) (bool, *api.JsonErr) {
	sessionInfo := map[string]string{datamodel.NodeRepositoryName: repoNCConfig.GetAssociationInfo().NodeID}
	sessionManager.BeginWriteSession(sessionInfo)

	var err *api.JsonErr
	defer func() {
		sessionManager.EndWriteSession(sessionInfo)
		if err != nil {
			logger.Errorf("processNetworkContainerPurge failed for NC: %s with error: %v", repoNCConfig.NetworkContainerID, err)
		}
	}()

	err = datamodel.PurgeNetworkContainerAssociations(repoNCConfig, updateVlan(repoNCConfig.GetAssociationInfo().ContainerType))
	if err != nil {
		return false, err
	}

	err = datamodel.DeleteNetworkContainer(repoNCConfig.NetworkID, repoNCConfig.NetworkContainerID)
	if err != nil {
		return false, err
	}

	deletionTimeMs := time.Since(createdDateTime).Milliseconds()
	sendNCMetric(aiwrapper.NCProcessTimeMetric, float64(deletionTimeMs),
		aiwrapper.Succeeded, "Container"+datamodel.Purge.String(), repoNCConfig, nil)

	return true, nil
}

func hasTagsChanged(resolvedTags map[string]datamodel.TagDefinition, currentTags map[string]uint64) bool {
	if len(resolvedTags) != len(currentTags) {
		return true
	}

	for tagName, tagDefinition := range resolvedTags {
		currentVersion, found := currentTags[tagName]
		if !found {
			logger.Debugf("TagName:%v not found", tagName)
			return true
		}

		if currentVersion != tagDefinition.Version {
			logger.Debugf("TagName:%v version change detected, Current:%v Updated:%v", tagName, currentVersion, tagDefinition.Version)
			return true
		}
	}

	return false
}

func getLocalIPGateway(ipsubnet string) (string, error) {
	_, ipNet, err := net.ParseCIDR(ipsubnet)
	if err != nil {
		logger.Errorf("parseCIDR for %v failed %v", ipsubnet, err)
		return "", err
	}

	ipstart := utils.IP2Int(ipNet.IP)
	ipstart = ipstart + 1
	return utils.Int2IP(ipstart).String(), nil
}

func processNetworkContainerAssociate(
	repoNCConfig *datamodel.NCConfiguration,
	sessionInfo map[string]string) (bool, *api.JsonErr) {
	sessionInfoRead := map[string]string{
		datamodel.ReservationRepositoryName: repoNCConfig.ReservationID,
		datamodel.NetworkRepositoryName:     repoNCConfig.NetworkID,
		datamodel.NodeRepositoryName:        repoNCConfig.GetAssociationInfo().NodeID,
	}
	sessionManager.BeginReadSession(sessionInfoRead)
	ncGoalState, isRetriableError, ncGoalStateErr := datamodel.GetNCGoalState(repoNCConfig)
	sessionManager.EndReadSession(sessionInfoRead)

	if ncGoalStateErr != nil {
		if isRetriableError {
			return false, ncGoalStateErr
		}

		repoNCConfig.Status = datamodel.Failed
		repoNCConfig.FailureDetail = *ncGoalStateErr
		return true, datamodel.PersistNCConfiguration(repoNCConfig, datamodel.DontAddToPendingQ)
	}

	cnetAddressesCount := len(ncGoalState.CNetSubnets)
	updatedTagInfo := make(map[string]uint64)
	resolvedTags := make(map[string]datamodel.TagDefinition)
	var tagsResolutionErr *api.JsonErr
	for _, tag := range ncGoalState.TagsInfo {
		tagDefinition, tagErr := RegisterAndGetTagDefinition(tag)
		if tagErr == nil {
			tagKey := datamodel.GetTagDefintionKey(tag.Type, tag.Name)
			resolvedTags[tagKey] = tagDefinition
			cnetAddressesCount += len(resolvedTags[tagKey].Values)
			updatedTagInfo[tagKey] = tagDefinition.Version
		} else {
			tagsResolutionErr = tagErr
		}
	}

	if tagsResolutionErr != nil {
		return false, tagsResolutionErr
	}

	if repoNCConfig.ProgrammedTagsInfo == nil {
		repoNCConfig.ProgrammedTagsInfo = make(map[string]uint64)
	}

	if ncGoalState.VnetVersion != repoNCConfig.ProgrammedVnetVersion ||
		hasTagsChanged(resolvedTags, repoNCConfig.ProgrammedTagsInfo) {
		repoNCConfig.CNSVersion++
	}

	var body bytes.Buffer
	var ipConfig cns.IPConfiguration
	var ipSubnet cns.IPSubnet
	var resp cns.CreateNetworkContainerResponse
	var respErr error

	ipConfig.DNSServers = ncGoalState.DNSServers
	ipConfig.GatewayIPAddress = ncGoalState.GatewayIP
	allocatedIP := strings.Split(ncGoalState.AllocatedIPSubnet, "/")[0]
	_, baseSubnetPrefix, err := utils.GetIPAddressStringAndPrefix(ncGoalState.BaseSubnet)
	if err != nil {
		msg := fmt.Sprintf("Failed to parse base subnet string %v due to error %v",
			ncGoalState.BaseSubnet,
			err.Error())
		failureDetail := api.JsonErr{
			Code:       api.InvalidIPSubnet,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
		repoNCConfig.Status = datamodel.Failed
		repoNCConfig.FailureDetail = failureDetail
		return true, datamodel.PersistNCConfiguration(repoNCConfig, datamodel.DontAddToPendingQ)
	}

	ipSubnet.IPAddress = allocatedIP
	ipSubnet.PrefixLength = uint8(baseSubnetPrefix)
	ipConfig.IPSubnet = ipSubnet
	cnetAddresses := make([]string, cnetAddressesCount)
	cnsCnetAddresses := make([]cns.IPSubnet, cnetAddressesCount)
	currentIndex := 0

	for _, cnetAddressString := range ncGoalState.CNetSubnets {
		cnetAddresses[currentIndex] = cnetAddressString
		currentIndex++
	}

	for _, tagDefinitions := range resolvedTags {
		for _, tagAddressString := range tagDefinitions.Values {
			cnetAddresses[currentIndex] = tagAddressString
			currentIndex++
		}
	}

	for index, cnetAddressString := range cnetAddresses {
		ipAddressString, prefix, parserErr := utils.GetIPAddressStringAndPrefix(cnetAddressString)
		if parserErr != nil {
			msg := fmt.Sprintf("Failed to parse Vnet subnet string %v due to error %v",
				cnetAddressString,
				err.Error())
			failureDetail := api.JsonErr{
				Code:       api.InvalidIPSubnet,
				Text:       msg,
				HTTPStatus: http.StatusBadRequest}
			repoNCConfig.Status = datamodel.Failed
			repoNCConfig.FailureDetail = failureDetail
			return true, datamodel.PersistNCConfiguration(repoNCConfig, datamodel.DontAddToPendingQ)
		}

		cnsIPSubnet := cns.IPSubnet{IPAddress: ipAddressString, PrefixLength: uint8(prefix)}
		cnsCnetAddresses[index] = cnsIPSubnet
	}

	localIPSubnet := cns.IPSubnet{
		IPAddress:    ncGoalState.LocalIP,
		PrefixLength: localIPSubnetPrefixLength}
	localIPPrefix := fmt.Sprintf("%s/%d", localIPSubnet.IPAddress, localIPSubnet.PrefixLength)
	localGWIP, err := getLocalIPGateway(localIPPrefix)
	if err != nil {
		return false, &api.JsonErr{
			HTTPStatus: http.StatusInternalServerError,
			Text:       fmt.Sprintf("getlocalgatewayIP failed: %v", err),
			Code:       api.StatusInternalServerError,
		}
	}

	localIPConfiguaration := cns.IPConfiguration{
		IPSubnet:         localIPSubnet,
		GatewayIPAddress: localGWIP,
	}

	allowHostToNCCommunicationStatus := repoNCConfig.GetAllowHostToNCCommunicationStatus()
	allowNCToHostCommunicationStatus := repoNCConfig.GetAllowNCToHostCommunicationStatus()
	endpointPolicies := repoNCConfig.GetEndpointPolicies()

	containerTypes.Store(ncGoalState.ContainerType, true)

	info := &cns.CreateNetworkContainerRequest{
		Version:                    strconv.FormatUint(repoNCConfig.CNSVersion, 10),
		NetworkContainerType:       ncGoalState.ContainerType,
		NetworkContainerid:         cns.SwiftPrefix + ncGoalState.NetworkContainerID,
		PrimaryInterfaceIdentifier: ncGoalState.AssociatedInterfaceID,
		IPConfiguration:            ipConfig,
		AuthorizationToken:         "",
		CnetAddressSpace:           cnsCnetAddresses,
		OrchestratorContext:        ncGoalState.OrchestratorContext,
		MultiTenancyInfo:           cns.MultiTenancyInfo{EncapType: cns.Vlan, ID: ncGoalState.VlanID},
		LocalIPConfiguration:       localIPConfiguaration,
		AllowHostToNCCommunication: allowHostToNCCommunicationStatus,
		AllowNCToHostCommunication: allowNCToHostCommunicationStatus,
		EndpointPolicies:           endpointPolicies}

	json.NewEncoder(&body).Encode(info)
	createOrUpdateNCURL := fmt.Sprintf(
		config.CNSSettings.ConnectionStrings.CreateOrUpdateNetworkContainerURLFmt,
		ncGoalState.NodeInterfaceID)

	// Release the session locks when calling in to cns
	sessionManager.EndWriteSession(sessionInfo)

	logger.Debugf("[CNS] create/update NC %s", ncGoalState.NetworkContainerID)
	w, respErr := httpClient.Post(createOrUpdateNCURL, defaultContentType, &body)

	// Reacquire the session locks post calling in to cns
	sessionManager.BeginWriteSession(sessionInfo)

	if respErr != nil {
		msg := fmt.Sprintf("Post to URL %s for NC %v failed with error %v",
			createOrUpdateNCURL,
			ncGoalState.NetworkContainerID,
			respErr.Error())
		return false, &api.JsonErr{
			Code:       api.AssociationFailed,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
	}

	defer w.Body.Close()
	respErr = utils.DecodeResponse(w, &resp)
	if respErr != nil {
		msg := fmt.Sprintf("Associate NC %v to node %v failed with error %s",
			ncGoalState.NetworkContainerID,
			ncGoalState.NodeInterfaceID,
			respErr.Error())
		return false, &api.JsonErr{
			Code:       api.AssociationFailed,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
	}

	logger.Debugf("[CNS][Response] create/update NC %s %+v", ncGoalState.NetworkContainerID, resp)

	// Retrieve the latest NC configuration as the current copy of the configuration
	// could have been updated while we were in the process of calling cns.
	retrievedRepoNCConfig, ncRetrievalErr := datamodel.TryQueryNetworkContainer(
		ncGoalState.NetworkID,
		ncGoalState.NetworkContainerID)
	if ncRetrievalErr != nil {
		msg := fmt.Sprintf(
			"Associate NC %v to node %v failed with error %s",
			ncGoalState.NetworkContainerID,
			ncGoalState.NodeInterfaceID,
			ncRetrievalErr.String())
		return false, &api.JsonErr{
			Code:       api.AssociationFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	if retrievedRepoNCConfig == nil {
		msg := fmt.Sprintf(
			"Associate NC %v to node %v failed as NC configuration was not found",
			ncGoalState.NetworkContainerID,
			ncGoalState.NodeInterfaceID)
		return true, &api.JsonErr{
			Code:       api.AssociationFailed,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
	}

	// Copy over the delete flag as NC could have been deleted when we were invoking CNS to create NC.
	repoNCConfig.MarkedForDelete = retrievedRepoNCConfig.MarkedForDelete

	// If the NC is marked for deletion, copy the state as it could have been changed too
	if repoNCConfig.MarkedForDelete {
		repoNCConfig.State = retrievedRepoNCConfig.State
	}

	// If the NC gets marked for delete while calling CNS, in case of error that request needs to be
	// marked as retriable error to avoid losing it and delete NC getting stuck
	if resp.Response.ReturnCode != 0 && !repoNCConfig.MarkedForDelete {
		msg := fmt.Sprintf("Associate NC %v to node %v failed with response %+v",
			ncGoalState.NetworkContainerID,
			ncGoalState.NodeInterfaceID,
			resp.Response)
		failureDetail := api.JsonErr{
			Code:       api.AssociationFailed,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
		repoNCConfig.Status = datamodel.Failed
		repoNCConfig.FailureDetail = failureDetail
		return true, datamodel.PersistNCConfiguration(repoNCConfig, datamodel.DontAddToPendingQ)
	}

	if repoNCConfig.State == datamodel.RefreshAssociation {
		repoNCConfig.State = datamodel.Program
	} else if repoNCConfig.State == datamodel.Associate {
		repoNCConfig.State = datamodel.Publish
	}

	repoNCConfig.ProgrammedTagsInfo = updatedTagInfo
	repoNCConfig.ProgrammedVnetVersion = ncGoalState.VnetVersion
	repoNCConfig.Status = datamodel.Pending
	repoNCConfig.FailureDetail = api.JsonErr{}

	return false, datamodel.PersistNCConfiguration(repoNCConfig, datamodel.DontAddToPendingQ)
}

func processNetworkContainerDisAssociate(
	repoNCConfig *datamodel.NCConfiguration,
	sessionInfo map[string]string) (bool, *api.JsonErr) {
	nodeID := repoNCConfig.GetAssociationInfo().NodeID

	sessionInfoRead := map[string]string{datamodel.NodeRepositoryName: nodeID}
	sessionManager.BeginReadSession(sessionInfoRead)
	// Query NodeConfiguration table with nodeID and handle errors.
	repoNodeConfig, err := datamodel.TryQueryNodeConfiguration(nodeID)
	sessionManager.EndReadSession(sessionInfoRead)

	if err != nil {
		return false, err
	} else if repoNodeConfig == nil {
		msg := fmt.Sprintf("Node configuration for node %v not found.",
			nodeID)
		failureDetail := api.JsonErr{
			Code:       api.NodeNotFound,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
		repoNCConfig.Status = datamodel.Failed
		repoNCConfig.FailureDetail = failureDetail
		return true, datamodel.PersistNCConfiguration(repoNCConfig, datamodel.DontAddToPendingQ)
	}

	primaryInterfaceID := repoNodeConfig.NodeInfo.IPAddresses[0]
	info := &cns.DeleteNetworkContainerRequest{
		NetworkContainerid: "Swift_" + repoNCConfig.NetworkContainerID,
	}

	sessionManager.EndWriteSession(sessionInfo)

	var body bytes.Buffer
	json.NewEncoder(&body).Encode(info)
	deleteNCURL := fmt.Sprintf(
		config.CNSSettings.ConnectionStrings.DeleteNetworkContainerURLFmt,
		primaryInterfaceID)

	logger.Debugf("[CNS] delete NC %s", repoNCConfig.NetworkContainerID)
	w, respErr := httpClient.Post(deleteNCURL, "application/json", &body)

	sessionManager.BeginWriteSession(sessionInfo)

	if respErr != nil {
		repoNCConfig.FailureDetail = api.JsonErr{
			Code: api.DisAssociationFailed,
			Text: fmt.Sprintf("Failed to disassociate NC %s on node %v due to error %s",
				info.NetworkContainerid,
				primaryInterfaceID,
				respErr.Error()),
			HTTPStatus: http.StatusInternalServerError}
		datamodel.PersistNCConfiguration(repoNCConfig, datamodel.DontAddToPendingQ)
		return false, &repoNCConfig.FailureDetail
	}

	defer w.Body.Close()

	var resp cns.DeleteNetworkContainerResponse
	decodeErr := utils.DecodeResponse(w, &resp)
	if decodeErr != nil {
		repoNCConfig.FailureDetail = api.JsonErr{
			Code:       api.DisAssociationFailed,
			Text:       decodeErr.Error(),
			HTTPStatus: http.StatusBadRequest}
		datamodel.PersistNCConfiguration(repoNCConfig, datamodel.DontAddToPendingQ)
		return false, &repoNCConfig.FailureDetail
	}

	logger.Debugf("[CNS] delete NC %s response %+v", repoNCConfig.NetworkContainerID, resp)

	if resp.Response.ReturnCode != 0 {
		msg := fmt.Sprintf("[CNS] Disassociate request associated with network container Id:%v returned status %v. "+
			"Message:%v. "+
			"Marking as non retriable error",
			repoNCConfig.NetworkContainerID,
			resp.Response.Message,
			resp.Response.ReturnCode)
		repoNCConfig.Status = datamodel.Failed
		repoNCConfig.FailureDetail = api.JsonErr{
			Code:       api.DisAssociationFailed,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
		return false, datamodel.PersistNCConfiguration(repoNCConfig, datamodel.DontAddToPendingQ)
	}

	logger.Debugf("[CNS] DisAssociated NC %s from node %s",
		repoNCConfig.NetworkContainerID,
		repoNCConfig.GetAssociationInfo().NodeID)
	repoNCConfig.State = datamodel.UnReserve
	repoNCConfig.Status = datamodel.Pending
	repoNCConfig.FailureDetail = api.JsonErr{}
	return false, datamodel.PersistNCConfiguration(repoNCConfig, datamodel.DontAddToPendingQ)
}

func getDimensions(repoNCConfig *datamodel.NCConfiguration) map[string]string {
	customDimensions := make(map[string]string)
	associationInfo := repoNCConfig.GetAssociationInfo()

	customDimensions[aiwrapper.NetworkContainerIDStr] = repoNCConfig.NetworkContainerID
	customDimensions[aiwrapper.NetworkIDStr] = repoNCConfig.NetworkID
	customDimensions[aiwrapper.ContainerTypeStr] = associationInfo.ContainerType
	customDimensions[aiwrapper.NodeIDStr] = associationInfo.NodeID
	customDimensions[aiwrapper.DNCVersionStr] = utils.Version

	return customDimensions
}

func sendNCMetric(
	metricName string,
	metricValue float64,
	status, ncState string,
	repoNCConfig *datamodel.NCConfiguration,
	err *api.JsonErr) {
	metric := aitelemetry.Metric{
		Name:             metricName,
		Value:            metricValue,
		CustomDimensions: getDimensions(repoNCConfig),
	}

	metric.CustomDimensions[aiwrapper.StatusStr] = status
	metric.CustomDimensions[aiwrapper.NCStateStr] = ncState

	errorCode := "NoError"

	if err != nil {
		errorCode = err.Code.String()
	}

	metric.CustomDimensions[aiwrapper.NCOperationErrorCodeStr] = errorCode
	go aiwrapper.SendMetric(metric)

	loggershim.LS.Debugf("[DNCOperationStatus] OperationName: %s, NetworkContainerId: %s, Status: %s, ErrorCode: %s", ncState, repoNCConfig.NetworkContainerID, status, errorCode)
}

func processNetworkContainerPublish(
	repoNCConfig *datamodel.NCConfiguration,
	sessionInfo map[string]string,
	stateTransitionDateTime time.Time,
	createdDateTime time.Time) (bool, *api.JsonErr) {
	sessionInfoRead := map[string]string{
		datamodel.ReservationRepositoryName: repoNCConfig.ReservationID,
		datamodel.NetworkRepositoryName:     repoNCConfig.NetworkID,
		datamodel.NodeRepositoryName:        repoNCConfig.GetAssociationInfo().NodeID,
	}
	sessionManager.BeginReadSession(sessionInfoRead)
	ncGoalState, isRetriableError, ncGoalStateErr := datamodel.GetNCGoalState(repoNCConfig)
	sessionManager.EndReadSession(sessionInfoRead)

	if ncGoalStateErr != nil {
		if isRetriableError {
			return false, ncGoalStateErr
		}

		repoNCConfig.Status = datamodel.Failed
		repoNCConfig.FailureDetail = *ncGoalStateErr
		return true, datamodel.PersistNCConfiguration(repoNCConfig, datamodel.DontAddToPendingQ)
	}

	var (
		resp               *http.Response
		respErr            error
		body               bytes.Buffer
		publishViaCNS      bool
		publishViaCNSError bool
	)

	var ncRequest nmagentcontracts.CreateOrUpdateNetworkContainerRequest
	ncRequest.IPV4Addresses = make([]string, 1)
	ncRequest.IPV4Addresses[0] = strings.Split(ncGoalState.AllocatedIPSubnet, "/")[0]
	ncRequest.Policies = make([]string, 0)
	ncRequest.SubnetName = ncGoalState.BaseSubnetName
	ncRequest.Version = strconv.FormatUint(ncGoalState.VnetVersion, 10)
	ncRequest.VirtualNetworkID = ncGoalState.NetworkID
	ncRequest.VlanID = ncGoalState.VlanID

	json.NewEncoder(&body).Encode(ncRequest)

	createNCURL := fmt.Sprintf(
		config.HostAgentSettings.ConnectionStrings.PutNetworkContainerURLFmt,
		config.HostAgentSettings.WireServerIP,
		ncGoalState.AssociatedInterfaceID,
		ncGoalState.NetworkContainerID,
		ncGoalState.AccessToken)

	if config.CNSSettings.ConnectionStrings.PublishNetworkContainerURLFmt != "" {
		publishViaCNS = true
	}

	logger.Debugf("[NMAgent] Creating NC: %s, publishViaCNS: %t, body:%s",
		ncGoalState.NetworkContainerID, publishViaCNS, ncRequest.String())

	if publishViaCNS {
		// Publish NC through CNS. CNS typically runs on the worker node and this path helps
		// minimize the wireserver throttling.
		// Create JoinNetworkURL
		joinNetworkURL := fmt.Sprintf(
			config.HostAgentSettings.ConnectionStrings.JoinNetworkURLFmt,
			config.HostAgentSettings.WireServerIP,
			ncGoalState.NetworkID)

		publishNCRequest := &cns.PublishNetworkContainerRequest{
			NetworkID:                         ncGoalState.NetworkID,
			NetworkContainerID:                ncGoalState.NetworkContainerID,
			JoinNetworkURL:                    joinNetworkURL,
			CreateNetworkContainerURL:         createNCURL,
			CreateNetworkContainerRequestBody: body.Bytes(),
		}

		// Make sure the publishing data is correctly encoded.
		// If not, fail the request with fatal error.
		var publishNCRequestBody bytes.Buffer
		if encodeErr := json.NewEncoder(&publishNCRequestBody).Encode(publishNCRequest); encodeErr != nil {
			msg := fmt.Sprintf("Publish operation failed for NC %s due to encoding error: %s",
				ncGoalState.NetworkContainerID,
				encodeErr.Error())
			loggershim.LS.Errorf(msg)
			repoNCConfig.Status = datamodel.Failed
			repoNCConfig.FailureDetail = api.JsonErr{
				Code:       api.PublishingFailed,
				Text:       msg,
				HTTPStatus: http.StatusBadRequest}
			return true, datamodel.PersistNCConfiguration(repoNCConfig, datamodel.DontAddToPendingQ)
		}

		publishNCURL := fmt.Sprintf(
			config.CNSSettings.ConnectionStrings.PublishNetworkContainerURLFmt,
			ncGoalState.NodeInterfaceID)

		logger.Debugf("[CNS] Publish NC %s, body:%s", ncGoalState.NetworkContainerID, ncRequest.String())
		// Release the session locks before making call to CNS
		sessionManager.EndWriteSession(sessionInfo)

		respPublishCNS, respPublishCNSErr :=
			httpClient.Post(publishNCURL, defaultContentType, &publishNCRequestBody)

		// Reacquire the session locks after receiving response from CNS
		sessionManager.BeginWriteSession(sessionInfo)

		var httpNetError bool
		if respPublishCNSErr != nil {
			// Return as retriable error if the error is not http network error.
			// If the error is http network error, publish NC will be done from the DNC node
			if _, httpNetError = respPublishCNSErr.(net.Error); !httpNetError {
				return false, &api.JsonErr{
					Code: api.PublishingFailed,
					Text: fmt.Sprintf("Failed to publish NC %s via CNS. Error: %s",
						ncGoalState.NetworkContainerID, respPublishCNSErr.Error()),
					HTTPStatus: http.StatusInternalServerError}
			}
		} else {
			// Close the response body if the error is nil
			defer respPublishCNS.Body.Close()
		}

		// Check if CNS used doesn't support the API
		var apiNotSupportedInCNS bool
		if respPublishCNS != nil && respPublishCNS.StatusCode == http.StatusNotFound {
			apiNotSupportedInCNS = true
		}

		// If HTTP call to CNS returns network error or if the API is not supported by CNS,
		// set the flag to publish NC from DNC node
		if httpNetError || apiNotSupportedInCNS {
			publishViaCNSError = true
			logger.Debugf("[CNS] Publish via CNS failed. httpNetError: %t, apiNotSupportedInCNS: %t "+
				"Publishing NC: %s from DNC node", httpNetError, apiNotSupportedInCNS, ncGoalState.NetworkContainerID)
		}

		// Proceed with decoding if there is no fatal error with publishing via CNS
		if !publishViaCNSError {
			var publishNCResponse cns.PublishNetworkContainerResponse
			if decodeRespErr := utils.DecodeResponse(respPublishCNS, &publishNCResponse); decodeRespErr != nil {
				msg := fmt.Sprintf("Failed to decode publish NC %s response. Error: %s",
					repoNCConfig.NetworkContainerID,
					decodeRespErr.Error())
				return false, &api.JsonErr{
					Code:       api.DecodeResponseFailed,
					Text:       msg,
					HTTPStatus: http.StatusInternalServerError}
			}

			logger.Debugf("[CNS][Response] Publish NC %s, %+v", ncGoalState.NetworkContainerID, publishNCResponse)

			// Construct response error and response from PublishNetworkContainerResponse received from CNS.
			if publishNCResponse.PublishErrorStr != "" {
				respErr = fmt.Errorf(publishNCResponse.PublishErrorStr)
			}

			resp = &http.Response{
				StatusCode: publishNCResponse.PublishStatusCode,
				Body:       ioutil.NopCloser(bytes.NewBuffer(publishNCResponse.PublishResponseBody)),
			}

			// If the network publish failed, decode the response and return the error
			if publishNCResponse.Response.ReturnCode == restserver.NetworkJoinFailed {
				logger.Errorf("[CNS] Publish NC %s failed due to network join error. HttpStatusCode: %d, Error: %s",
					ncGoalState.NetworkContainerID, publishNCResponse.PublishStatusCode, publishNCResponse.PublishErrorStr)

				isRetriable, errJoinNetwork, _ := decodeNetworkJoinResponse(ncGoalState.NetworkID, resp, respErr)
				return !isRetriable, errJoinNetwork // !isRetriable indicates not to remove request from pending queue
			}

			// If the error is unexpected failure, mark this as retriable error
			if publishNCResponse.Response.ReturnCode == restserver.UnexpectedError {
				msg := fmt.Sprintf("Failed to publish NC %s via CNS. Error: %s",
					repoNCConfig.NetworkContainerID,
					publishNCResponse.Response.Message)
				return false, &api.JsonErr{
					Code:       api.PublishingFailed,
					Text:       msg,
					HTTPStatus: http.StatusInternalServerError}
			}
		}
	}

	// If the publish from CNS is not supported or if the publish call to CNS returned http network error,
	// publish the NC from the DNC node.
	if !publishViaCNS || publishViaCNSError {
		// Release the session locks before making call to NMAgent
		sessionManager.EndWriteSession(sessionInfo)

		// If the API to publish NC through CNS is not supported / used in the config file,
		// publish the NC by making the call to NMAgent on the same host as DNC
		resp, respErr = httpClient.Post(createNCURL, "application/json", &body)

		// Reacquire the session locks after receiving response from NMAgent
		sessionManager.BeginWriteSession(sessionInfo)
	}

	if respErr != nil {
		return false, &api.JsonErr{
			Code:       api.PublishingFailed,
			Text:       fmt.Sprintf("Failed to publish NC %s with error %v", ncGoalState.NetworkContainerID, respErr.Error()),
			HTTPStatus: http.StatusInternalServerError}
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, &api.JsonErr{
			Code: api.PublishingFailed,
			Text: fmt.Sprintf("Failed to publish NC %s to with error code %d",
				repoNCConfig.NetworkContainerID,
				resp.StatusCode),
			HTTPStatus: http.StatusInternalServerError}
	}

	var createNetworkContainerResponse nmagentcontracts.CreateNetworkContainerResponse
	decodeRespErr := utils.DecodeResponse(resp, &createNetworkContainerResponse)
	if decodeRespErr != nil {
		msg := fmt.Sprintf("Failed to decode create NC %s response due to error %s.",
			repoNCConfig.NetworkContainerID,
			decodeRespErr.Error())
		return false, &api.JsonErr{
			Code:       api.DecodeResponseFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	logger.Debugf("[NMAgent][Response] %s, publishViaCNS: %t",
		createNetworkContainerResponse.String(), publishViaCNS)

	putNCResponseCode, parseErr := strconv.ParseInt(createNetworkContainerResponse.ResponseCode, 10, 32)
	if parseErr != nil {
		msg := fmt.Sprintf(
			"Failed to convert response code %s to integer when parsing create NC %s response",
			createNetworkContainerResponse.ResponseCode,
			repoNCConfig.NetworkContainerID)
		return false, &api.JsonErr{
			Code:       api.ParseResponseCodeFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	// Retrieve the latest NC configuration as the current copy of the configuration
	// could have been updated while we were in the process of calling cns.
	retrievedRepoNCConfig, ncRetrievalErr := datamodel.TryQueryNetworkContainer(
		ncGoalState.NetworkID,
		ncGoalState.NetworkContainerID)
	if ncRetrievalErr != nil {
		msg := fmt.Sprintf(
			"Publish NC %v failed with error %s",
			ncGoalState.NetworkContainerID,
			ncRetrievalErr.String())
		return false, &api.JsonErr{
			Code:       api.PublishingFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	if retrievedRepoNCConfig == nil {
		msg := fmt.Sprintf(
			"Publish NC %v failed as NC configuration was not found",
			ncGoalState.NetworkContainerID)
		return true, &api.JsonErr{
			Code:       api.PublishingFailed,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
	}

	// Copy over the delete flag as NC could have been deleted while calling CNS / NMAgent for publish.
	repoNCConfig.MarkedForDelete = retrievedRepoNCConfig.MarkedForDelete

	// If the NC is marked for deletion, copy the state as it could have been changed too
	if repoNCConfig.MarkedForDelete {
		repoNCConfig.State = retrievedRepoNCConfig.State
	}

	if putNCResponseCode != http.StatusOK {
		// If the response code is authentication failure, retry the request for the preset duration.
		// If authentication failure is still seen fail the request
		// This workaround is needed because sometimes there is a latency in sending the new authentication token
		// to pubsub. This workaround hides that till the latency is addressed.
		// If the NC gets marked for delete while calling CNS / NMAgent, in case of error that request needs to be
		// marked as retriable error to avoid losing it and delete NC getting stuck
		if putNCResponseCode == http.StatusUnauthorized {
			if (time.Since(stateTransitionDateTime).Seconds() < unauthorizedErrRetryDurationSec) ||
				repoNCConfig.MarkedForDelete {
				msg := fmt.Sprintf("Create NC %s pending. Response code: %d, time elapsed: %.0f sec",
					repoNCConfig.NetworkContainerID,
					putNCResponseCode,
					time.Since(stateTransitionDateTime).Seconds())
				return false, &api.JsonErr{
					Code:       api.ProcessingPending,
					Text:       msg,
					HTTPStatus: http.StatusUnauthorized}
			}

			// Mark the request as non-retriable if it fails for retry duration
			msg := fmt.Sprintf("Failed to publish NC: %s with status code: %v. Marking as non retriable.",
				repoNCConfig.NetworkContainerID, putNCResponseCode)
			repoNCConfig.Status = datamodel.Failed
			repoNCConfig.FailureDetail = api.JsonErr{
				Code:       api.PublishingFailed,
				Text:       msg,
				HTTPStatus: http.StatusUnauthorized}

			return true, datamodel.PersistNCConfiguration(repoNCConfig, datamodel.DontAddToPendingQ)
		}

		// Retry if NMAgent error response code is anything other than http.StatusUnauthorized
		msg := fmt.Sprintf("Failed to publish NC: %s with status code: %v. Marking as retriable.",
			repoNCConfig.NetworkContainerID, putNCResponseCode)

		return false, &api.JsonErr{
			Code:       api.PublishingFailed,
			Text:       msg,
			HTTPStatus: int(putNCResponseCode)}
	}

	if repoNCConfig.State == datamodel.Publish {
		repoNCConfig.State = datamodel.Program
	}
	repoNCConfig.Status = datamodel.Pending
	repoNCConfig.FailureDetail = api.JsonErr{}

	creationTimeMs := time.Since(createdDateTime).Milliseconds()
	sendNCMetric(aiwrapper.NCProcessTimeMetric, float64(creationTimeMs),
		aiwrapper.Succeeded, "Container"+datamodel.Program.String(), repoNCConfig, nil)

	// Log NC Snapshot
	aiwrapper.LogNCSnapsnot(ncGoalState.NetworkContainerID, ncGoalState.NetworkID, ncGoalState.BaseSubnetName, ncGoalState.AssociatedNodeID)

	return false, datamodel.PersistNCConfiguration(repoNCConfig, datamodel.DontAddToPendingQ)
}

func processNetworkContainerUnPublish(
	repoNCConfig *datamodel.NCConfiguration,
	sessionInfo map[string]string,
	stateTransitionDateTime time.Time) (bool, *api.JsonErr) {
	sessionInfoRead := map[string]string{
		datamodel.ReservationRepositoryName: repoNCConfig.ReservationID,
		datamodel.NetworkRepositoryName:     repoNCConfig.NetworkID,
		datamodel.NodeRepositoryName:        repoNCConfig.GetAssociationInfo().NodeID,
	}
	sessionManager.BeginReadSession(sessionInfoRead)
	ncGoalState, isRetriableError, ncGoalStateErr := datamodel.GetNCGoalState(repoNCConfig)
	sessionManager.EndReadSession(sessionInfoRead)

	if ncGoalStateErr != nil {
		if isRetriableError {
			return false, ncGoalStateErr
		}

		repoNCConfig.Status = datamodel.Failed
		repoNCConfig.FailureDetail = *ncGoalStateErr
		return true, datamodel.PersistNCConfiguration(repoNCConfig, datamodel.DontAddToPendingQ)
	}

	var (
		resp                 *http.Response
		respErr              error
		body                 bytes.Buffer
		unpublishViaCNS      bool
		unpublishViaCNSError bool
	)

	// Empty body is required as wireserver cannot handle a post without the body.
	json.NewEncoder(&body).Encode("")
	deleteNCURL := fmt.Sprintf(
		config.HostAgentSettings.ConnectionStrings.DeleteNetworkContainerURLFmt,
		config.HostAgentSettings.WireServerIP,
		ncGoalState.AssociatedInterfaceID,
		ncGoalState.NetworkContainerID,
		ncGoalState.AccessToken)

	if config.CNSSettings.ConnectionStrings.UnpublishNetworkContainerURLFmt != "" {
		unpublishViaCNS = true
	}

	logger.Debugf("[NMAgent] deleting NC %s, unpublishViaCNS: %t",
		ncGoalState.NetworkContainerID, unpublishViaCNS)

	if unpublishViaCNS {
		// Unpublish NC through CNS. CNS typically runs on the worker node and this path helps
		// minimize the wireserver throttling.
		// Create JoinNetworkURL
		joinNetworkURL := fmt.Sprintf(
			config.HostAgentSettings.ConnectionStrings.JoinNetworkURLFmt,
			config.HostAgentSettings.WireServerIP,
			ncGoalState.NetworkID)

		unpublishNCRequest := &cns.UnpublishNetworkContainerRequest{
			NetworkID:                 ncGoalState.NetworkID,
			NetworkContainerID:        ncGoalState.NetworkContainerID,
			JoinNetworkURL:            joinNetworkURL,
			DeleteNetworkContainerURL: deleteNCURL,
		}

		var unpublishNCRequestBody bytes.Buffer
		json.NewEncoder(&unpublishNCRequestBody).Encode(unpublishNCRequest)

		unpublishNCURL := fmt.Sprintf(
			config.CNSSettings.ConnectionStrings.UnpublishNetworkContainerURLFmt,
			ncGoalState.NodeInterfaceID)

		logger.Debugf("[CNS] Unpublish NC %s", ncGoalState.NetworkContainerID)
		// Release the session locks before making call to NMAgent
		sessionManager.EndWriteSession(sessionInfo)

		respUnpublishCNS, respUnpublishCNSErr :=
			httpClient.Post(unpublishNCURL, defaultContentType, &unpublishNCRequestBody)

		// Reacquire the session locks after receiving response from NMAgent
		sessionManager.BeginWriteSession(sessionInfo)

		var httpNetError bool
		if respUnpublishCNSErr != nil {
			// Return as retriable error if the error is not http network error.
			// If the error is http network error, unpublish NC will be done from the DNC node
			_, httpNetError = respUnpublishCNSErr.(net.Error)
			if !httpNetError {
				return false, &api.JsonErr{
					Code: api.UnpublishingFailed,
					Text: fmt.Sprintf("Failed to unpublish NC %s via CNS. Error: %s",
						ncGoalState.NetworkContainerID, respUnpublishCNSErr.Error()),
					HTTPStatus: http.StatusInternalServerError}
			}
		} else {
			// Close the response body if the error is nil
			defer respUnpublishCNS.Body.Close()
		}

		// Check if CNS used doesn't support the API
		var apiNotSupportedInCNS bool
		if respUnpublishCNS != nil && respUnpublishCNS.StatusCode == http.StatusNotFound {
			apiNotSupportedInCNS = true
		}

		// If HTTP call to CNS returns network error or if the API is not supported by CNS,
		// set the flag to unpublish NC from DNC node
		if httpNetError || apiNotSupportedInCNS {
			unpublishViaCNSError = true
			logger.Debugf("[CNS] Unpublish via CNS failed. httpNetError: %t, apiNotSupportedInCNS: %t "+
				"Unpublishing NC: %s from DNC node", httpNetError, apiNotSupportedInCNS, ncGoalState.NetworkContainerID)
		}

		// Proceed with decoding if there is no fatal error with unpublishing via CNS
		if !unpublishViaCNSError {
			var unpublishNCResponse cns.UnpublishNetworkContainerResponse
			if decodeRespErr := utils.DecodeResponse(respUnpublishCNS, &unpublishNCResponse); decodeRespErr != nil {
				msg := fmt.Sprintf("Failed to decode unpublish NC %s response. Error: %s",
					repoNCConfig.NetworkContainerID,
					decodeRespErr.Error())
				return false, &api.JsonErr{
					Code:       api.DecodeResponseFailed,
					Text:       msg,
					HTTPStatus: http.StatusInternalServerError}
			}

			logger.Debugf("[CNS][Response] Unpublish NC %s, %+v", ncGoalState.NetworkContainerID, unpublishNCResponse)

			// Construct response error and response from UnpublishNetworkContainerResponse received from CNS.
			if unpublishNCResponse.UnpublishErrorStr != "" {
				respErr = fmt.Errorf(unpublishNCResponse.UnpublishErrorStr)
			}

			resp = &http.Response{
				StatusCode: unpublishNCResponse.UnpublishStatusCode,
				Body:       ioutil.NopCloser(bytes.NewBuffer(unpublishNCResponse.UnpublishResponseBody)),
			}

			// If the network publish failed, decode the response and return the error
			if unpublishNCResponse.Response.ReturnCode == restserver.NetworkJoinFailed {
				logger.Errorf("[CNS] Unpublish NC %s failed due to network join error. HttpStatusCode: %d, Error: %s",
					ncGoalState.NetworkContainerID, unpublishNCResponse.UnpublishStatusCode, unpublishNCResponse.UnpublishErrorStr)

				isRetriable, errJoinNetwork, _ := decodeNetworkJoinResponse(ncGoalState.NetworkID, resp, respErr)
				return !isRetriable, errJoinNetwork // !isRetriable indicates not to remove request from pending queue
			}

			// If the error is unexpected failure, mark this as retriable error
			if unpublishNCResponse.Response.ReturnCode == restserver.UnexpectedError {
				msg := fmt.Sprintf("Failed to unpublish NC %s via CNS. Error: %s",
					repoNCConfig.NetworkContainerID,
					unpublishNCResponse.Response.Message)
				return false, &api.JsonErr{
					Code:       api.UnpublishingFailed,
					Text:       msg,
					HTTPStatus: http.StatusInternalServerError}
			}
		}
	}

	// If the unpublish from CNS is not supported or if the unpublish call to CNS returned http network error,
	// unpublish the NC from the DNC node.
	if !unpublishViaCNS || unpublishViaCNSError {
		// Release the session locks before making call to NMAgent
		sessionManager.EndWriteSession(sessionInfo)

		// If the API to unpublish NC through CNS is not supported / used in the config file,
		// unpublish the NC by making the call to NMAgent on the same host as DNC
		resp, respErr = httpClient.Post(deleteNCURL, "application/json", &body)

		// Reacquire the session locks after receiving response from NMAgent
		sessionManager.BeginWriteSession(sessionInfo)
	}

	if respErr != nil {
		return false, &api.JsonErr{
			Code: api.UnpublishingFailed,
			Text: fmt.Sprintf("Failed to unpublish NC %s with error %v",
				repoNCConfig.NetworkContainerID,
				respErr.Error()),
			HTTPStatus: http.StatusInternalServerError}
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, &api.JsonErr{
			Code: api.UnpublishingFailed,
			Text: fmt.Sprintf("Failed to unpublish NC %s with error code %d",
				repoNCConfig.NetworkContainerID,
				resp.StatusCode),
			HTTPStatus: http.StatusInternalServerError}
	}

	var deleteNCResponse nmagentcontracts.DeleteNetworkContainerResponse
	decodeRespErr := utils.DecodeResponse(resp, &deleteNCResponse)
	if decodeRespErr != nil {
		msg := fmt.Sprintf("Failed to decode delete NC %s response due to error %s",
			repoNCConfig.NetworkContainerID,
			decodeRespErr.Error())
		return false, &api.JsonErr{
			Code:       api.DecodeResponseFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	logger.Debugf("[NMAgent][Response] %s for NC: %s, unpublishViaCNS: %t",
		deleteNCResponse.String(),
		repoNCConfig.NetworkContainerID,
		unpublishViaCNS)

	deleteNCResponseCode, parseErr := strconv.ParseInt(deleteNCResponse.ResponseCode, 10, 64)
	if parseErr != nil {
		msg := fmt.Sprintf(
			"Failed to convert response code %s  to integer when parsing delete NC response for NC %s, Marking this error as retriable.",
			deleteNCResponse.ResponseCode, repoNCConfig.NetworkContainerID)
		return false, &api.JsonErr{
			Code:       api.ParseResponseCodeFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	if deleteNCResponseCode != http.StatusOK {
		// If the response code is authentication failure, retry the request for the preset duration.
		// If authentication failure is still seen fail the request
		// This workaround is needed because sometimes there is a latency in sending the new authentication token
		// to pubsub. This workaround hides that till the latency is addressed.
		if deleteNCResponseCode == http.StatusUnauthorized {
			if time.Since(stateTransitionDateTime).Seconds() < unauthorizedErrRetryDurationSec {
				msg := fmt.Sprintf("Delete NC %s pending. Response code: %d, time elapsed: %.0f sec",
					repoNCConfig.NetworkContainerID,
					deleteNCResponseCode,
					time.Since(stateTransitionDateTime).Seconds())
				return false, &api.JsonErr{
					Code:       api.ProcessingPending,
					Text:       msg,
					HTTPStatus: http.StatusUnauthorized}
			}

			// Mark the request as non-retriable if it fails for retry duration
			msg := fmt.Sprintf("Failed to unpublish NC: %s with status code: %v. Marking as non retriable.",
				repoNCConfig.NetworkContainerID, deleteNCResponseCode)
			repoNCConfig.Status = datamodel.Failed
			repoNCConfig.FailureDetail = api.JsonErr{
				Code:       api.UnpublishingFailed,
				Text:       msg,
				HTTPStatus: http.StatusUnauthorized}
			return true, datamodel.PersistNCConfiguration(repoNCConfig, datamodel.DontAddToPendingQ)
		}

		// Retry if NMAgent error response code is anything other than http.StatusUnauthorized
		msg := fmt.Sprintf("Failed to unpublish NC: %s with status code: %v. Marking as retriable.",
			repoNCConfig.NetworkContainerID, deleteNCResponseCode)

		return false, &api.JsonErr{
			Code:       api.UnpublishingFailed,
			Text:       msg,
			HTTPStatus: int(deleteNCResponseCode)}
	}

	repoNCConfig.State = datamodel.DisAssociate
	repoNCConfig.Status = datamodel.Pending
	repoNCConfig.FailureDetail = api.JsonErr{}
	return false, datamodel.PersistNCConfiguration(repoNCConfig, datamodel.DontAddToPendingQ)
}

func processNetworkContainerProgram(
	repoNCConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	repoNCConfig.State = datamodel.Program
	repoNCConfig.Status = datamodel.Completed
	repoNCConfig.FailureDetail = api.JsonErr{}
	return true, datamodel.PersistNCConfiguration(repoNCConfig, datamodel.DontAddToPendingQ)
}

func processNetworkContainerRequest(
	requestContext *datamodel.RequestContext,
	networkContainerKey string,
	sessionInfo map[string]string) (bool, *api.JsonErr) {
	var (
		requestID                     = networkContainerKey
		createdDateTime               = requestContext.CreatedDateTime
		stateTransitionDateTime       = requestContext.StateTransitionDateTime
		removeRequestFromPendingQueue = false
		err                           *api.JsonErr
		repoNCConfig                  *datamodel.NCConfiguration
	)

	defer func() {
		if repoNCConfig != nil {
			if removeRequestFromPendingQueue && repoNCConfig.Status == datamodel.Failed {
				processingTime := time.Since(createdDateTime).Milliseconds()
				var ncState string
				if datamodel.IsDeleteInProgress(repoNCConfig.State) {
					ncState = "Container" + datamodel.Purge.String()
					logger.Errorf("NC deletion failed for NC: %s",
						datamodel.GetNCIdFromKey(networkContainerKey))
				} else {
					ncState = "Container" + datamodel.Program.String()
					logger.Errorf("NC creation failed for NC: %s",
						datamodel.GetNCIdFromKey(networkContainerKey))
				}
				sendNCMetric(aiwrapper.NCProcessTimeMetric, float64(processingTime),
					aiwrapper.Failed, ncState, repoNCConfig, err)
			}
		}
	}()

	// Continue advancing the NC state to completion while there is no error (err == nil) and
	// indication to remove the request from the pending queue (!removeRequestFromPendingQueue)
	for !removeRequestFromPendingQueue && err == nil {
		repoNCConfig, err = datamodel.TryQueryNetworkContainerByKey(networkContainerKey)

		if err != nil {
			return false, err
		}

		if repoNCConfig == nil {
			logger.Errorf("Failed to find network container configuration in repository for ID %s", networkContainerKey)
			return true, nil
		}

		// Check if NC has been marked for delete and if so ensure that we are in the correct deletion state.
		if repoNCConfig.MarkedForDelete && !datamodel.IsDeleteInProgress(repoNCConfig.State) {
			revertState, err := datamodel.GetRevertState(repoNCConfig.State)
			if err != nil {
				logger.Errorf("Failed to get the deletion state for NC: %s, current state: %s",
					repoNCConfig.NetworkContainerID, repoNCConfig.State)
				return false, err
			}
			repoNCConfig.State = revertState
		}

		if repoNCConfig.Status == datamodel.Failed {
			return true, nil
		}

		ncState := repoNCConfig.State

		switch repoNCConfig.State {
		case datamodel.Prepare:
			removeRequestFromPendingQueue, err = preparer.PrepareNetworkContainer(repoNCConfig)

		case datamodel.Reserve:
			removeRequestFromPendingQueue, err = processNetworkContainerReserve(repoNCConfig)

		case datamodel.AllocateLocalIP:
			removeRequestFromPendingQueue, err = processAllocateLocalIP(repoNCConfig)

		case datamodel.Associate:
			fallthrough

		case datamodel.RefreshAssociation:
			removeRequestFromPendingQueue, err = processNetworkContainerAssociate(repoNCConfig, sessionInfo)

		case datamodel.Publish:
			removeRequestFromPendingQueue, err = processNetworkContainerPublish(repoNCConfig, sessionInfo, stateTransitionDateTime, createdDateTime)

		case datamodel.Program:
			removeRequestFromPendingQueue, err = processNetworkContainerProgram(repoNCConfig)

		case datamodel.UnPublish:
			removeRequestFromPendingQueue, err = processNetworkContainerUnPublish(repoNCConfig, sessionInfo, stateTransitionDateTime)

		case datamodel.DisAssociate:
			removeRequestFromPendingQueue, err = processNetworkContainerDisAssociate(repoNCConfig, sessionInfo)

		case datamodel.UnReserve:
			removeRequestFromPendingQueue, err = processNetworkContainerUnReserve(repoNCConfig)

		case datamodel.Purge:
			removeRequestFromPendingQueue, err = processNetworkContainerPurge(repoNCConfig, createdDateTime)

		default:
			logger.Errorf("NC: %s processing failed due to invalid state %s", repoNCConfig.NetworkContainerID, repoNCConfig.State)
			return true, nil
		}

		var processStatus string
		if !removeRequestFromPendingQueue && err == nil {
			processStatus = aiwrapper.Succeeded
		} else {
			processStatus = aiwrapper.Failed
		}

		// Record the time taken for NC state processing and send the NC process time metric
		recordNetworkContainerProcessingTime(repoNCConfig, ncState,
			processStatus, createdDateTime, stateTransitionDateTime, err)

		if !removeRequestFromPendingQueue && err == nil {
			requestContext.LastRetriedDateTime = time.Time{}
			requestContext.RetryInterval = requestContextRetryInterval
			requestContext.StateTransitionDateTime = time.Now().UTC()

			datamodel.AddRequestToPendingQueue(requestID, requestContext)

			// Update stateTransitionDateTime to be used during the successful state transition
			stateTransitionDateTime = requestContext.StateTransitionDateTime
		}
	}

	return removeRequestFromPendingQueue, err
}

// Record the time taken for NC state processing and send the NC process time metric
func recordNetworkContainerProcessingTime(
	repoNCConfig *datamodel.NCConfiguration,
	ncState datamodel.NetworkContainerState,
	processStatus string,
	createdDateTime, stateTransitionDateTime time.Time,
	err *api.JsonErr) {
	if ncState == datamodel.Program ||
		ncState == datamodel.Purge {
		// Program and purge times are calculated based on request create time. This provides
		// the total time taken to program (create) and delete the NC.
		utils.RecordTime(createdDateTime, "Container"+ncState.String(), repoNCConfig.NetworkContainerID)
	} else {
		utils.RecordTime(stateTransitionDateTime, "Container"+ncState.String(), repoNCConfig.NetworkContainerID)
	}

	// Upload state process time metric for all the states. Program and Purge times are calulcated
	// based on request create time and RefreshAssociation is excluded to avoid flooding the metric
	// data as every NC will periodically go through refreshAssociation
	if ncState != datamodel.Program &&
		ncState != datamodel.Purge &&
		ncState != datamodel.RefreshAssociation {
		processTimeMs := time.Since(stateTransitionDateTime).Milliseconds()
		sendNCMetric(aiwrapper.NCProcessTimeMetric,
			float64(processTimeMs), processStatus, "Container"+ncState.String(), repoNCConfig, err)
	}
}

func updateVlan(containerType string) bool {
	switch containerType {
	case cns.AzureContainerInstance:
		fallthrough
	case cns.Docker:
		fallthrough
	case cns.JobObject:
		fallthrough
	case cns.COW:
		return true
	default:
		return false
	}
}

func monitorRequests() {
	// Sync map for keeping track of in-process requests
	var inProcessRequestsMap sync.Map

	maxConcurrentRequestsToProcess := defaultMaxConcurrentRequestsToProcess
	if config.DNCSettings.ConcurrentRequestsToProcess != 0 {
		maxConcurrentRequestsToProcess = config.DNCSettings.ConcurrentRequestsToProcess
	}

	// Dummy channel to coordinate the number of concurrent goroutines.
	// This channel should be buffered otherwise we will be immediately blocked
	// when trying to fill it.
	concurrentGoroutines := make(chan struct{}, maxConcurrentRequestsToProcess)

	for i := 0; i < maxConcurrentRequestsToProcess; i++ {
		concurrentGoroutines <- struct{}{}
	}

	monitorTimerChannel := time.NewTicker(time.Millisecond * 1000).C
	for {
		select {
		case <-monitorTimerChannel:
			if replicaInstance.IsReplicaPrimary() {
				processPendingRequests(concurrentGoroutines, &inProcessRequestsMap)
			}
			break

		case <-startProcessing:
			networkMonitorStartProcessing <- true
			tagMonitorStartProcessing <- true
			refreshNCStartProcessing <- true
			processPendingRequests(concurrentGoroutines, &inProcessRequestsMap)
			break

		case <-cancelProcessing:
			networkMonitorStopProcessing <- true
			tagMonitorStopProcessing <- true
			refreshNCStopProcessing <- true
			telemetryStopProcessing <- true
			logger.Debugf("Received signal to exit")
			break
		}
	}
}

// Monitor that periodically processes pending requests in the work item queue.
func processPendingRequests(
	concurrentGoroutines chan struct{},
	inProcessRequestsMap *sync.Map) {
	if initiateMigration {
		if err := datamodel.ProcessSharedSubnetDelegationMigration(); err != nil {
			logger.Errorf("ProcessSharedSubnetDelegationMigration failed %v", err)
			return
		}

		initiateMigration = false
	}

	requests, _ := datamodel.GetPendingRequests()
	if len(requests) == 0 {
		return
	}

	requestsToProcess := make([]string, 0, len(requests))
	currentBatchRequestCount := 0

	for requestID, requestContext := range requests {
		_, isRequestInProcess := inProcessRequestsMap.Load(requestID)

		if !isRequestInProcess &&
			time.Now().UTC().Sub(requestContext.LastRetriedDateTime) > requestContext.RetryInterval {
			requestsToProcess = append(requestsToProcess, requestID)

			// Add the request in the in-process map
			inProcessRequestsMap.Store(requestID, struct{}{})
			currentBatchRequestCount++
		}
	}

	logger.Debugf("Number of requests to process in the current batch: %d", currentBatchRequestCount)

	for i := 0; i < currentBatchRequestCount; i++ {
		// Return if the replica is not primary
		if !replicaInstance.IsReplicaPrimary() {
			// Remove the requestIDs from the inProcessRequestMap which will not be processed
			for reqIndex := i; reqIndex < currentBatchRequestCount; reqIndex++ {
				logger.Debugf("Removing unprocessed request from the current batch: %s",
					requestsToProcess[reqIndex])
				inProcessRequestsMap.Delete(requestsToProcess[reqIndex])
			}

			break
		}

		// Try to receive from the concurrentGoroutines channel. When we have something,
		// it means we can start a new goroutine because another one finished.
		// Otherwise, it will block the execution until an execution
		// spot is available.
		<-concurrentGoroutines
		requestID := requestsToProcess[i]
		requestContext := requests[requestID]

		go func(rid string, rc datamodel.RequestContext, requestsMap *sync.Map, goroutines chan struct{}) {
			switch rc.Type {
			case datamodel.NetworkContainerRequest:
				logger.Debugf("Processing %s [%s] on Node [%s]",
					rc.Type.String(),
					rid,
					rc.NodeID)
			case datamodel.VirtualNetworkRequest:
				fallthrough
			case datamodel.ReservationRequest:
				logger.Debugf("Processing %s [%s]",
					rc.Type.String(),
					rid)
			}

			processRequest(rid, &rc)
			// Remove the request from the in-process map
			requestsMap.Delete(rid)
			goroutines <- struct{}{}
		}(requestID, requestContext, inProcessRequestsMap, concurrentGoroutines)
	}
}
