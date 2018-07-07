package main

import (
	"bytes"
	"dnc/api"
	"dnc/datamodel"
	"dnc/nmagentcontracts"
	"dnc/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-container-networking/cns"
	"github.com/google/uuid"
)

// NCGoalState - Struct to cache the Network Container Goal State
type NCGoalState struct {
	NetworkContainerID    string
	Version               uint64
	NetworkID             string
	AllocatedIPSubnet     string
	BaseSubnet            string
	BaseSubnetName        string
	AccessToken           string
	GatewayIP             string
	DNSServers            []string
	AssociatedNodeID      string
	AssociatedInterfaceID string
	CNetSubnets           []string
	VNetSubnets           []string
	NodeInterfaceID       string
	TagsInfo              []nmagentcontracts.Tag
	ContainerType         string
	OrchestratorContext   json.RawMessage
	VlanID                int
	VnetVersion           uint64
}

func getSessionInfo(request *datamodel.RequestContext) map[string]string {
	sessionInfo := make(map[string]string)
	if len(request.NetworkID) != 0 {
		sessionInfo[datamodel.NetworkRepositoryName] = request.NetworkID
	}
	if len(request.NodeID) != 0 {
		sessionInfo[datamodel.NodeRepositoryName] = request.NodeID
	}

	return sessionInfo
}

func processRequest(requestID string, requestContext *datamodel.RequestContext) {
	var removeWorkItem bool
	var err *api.JsonErr

	sessionInfo := getSessionInfo(requestContext)
	sessionManager.BeginWriteSession(sessionInfo)
	defer sessionManager.EndWriteSession(sessionInfo)

	// Update retry/request interval (by doubling it) unless it's greater than 5 minutes
	if requestContext.RetryInterval < 5*time.Minute {
		requestContext.RetryInterval *= 2
	}

	switch requestContext.Type {
	case datamodel.VirtualNetworkRequest:
		removeWorkItem, err = processVirtualNetworksPendingJoin(requestContext.NetworkID)
		break

	case datamodel.ReservationRequest:
		removeWorkItem, err = processIPReservationAllocation(requestContext.NetworkID, requestID)
		break

	case datamodel.NetworkContainerRequest:
		removeWorkItem, err = processNetworkContainerRequest(
			requestContext.NetworkID,
			requestID,
			sessionInfo)
		break

	}

	if err == nil && removeWorkItem {
		datamodel.RemovePendingRequest(requestID)
	} else {
		if err != nil {
			logger.Errorf("%+v", err)
		} else {
			logger.Debugf("Resetting the retry interval as the operation has succeeded.")
			requestContext.RetryInterval = requestContextRetryInterval
		}

		datamodel.AddRequestToPendingQueue(requestID, requestContext)
	}
}

// getNCGoalState - Get NC goal state from different tables.
// Return the following.
// 1. Constructed NC goal state
// 2. If there was an error, True if the error is retriable , False otherwise.
// 3. Error.
func getNCGoalState(repoNCConfig *datamodel.NCConfiguration) (*NCGoalState, bool, *api.JsonErr) {
	logger.Debugf("Retrieving Network Container Goal state for container %s\n", repoNCConfig.NetworkContainerID)
	nodeID := repoNCConfig.GetAssociationInfo().NodeID
	repoReservationConfiguration, err :=
		datamodel.TryQueryIPReservation(repoNCConfig.NetworkID, repoNCConfig.ReservationID)
	if err != nil {
		logger.Errorf("Retrieval of IPReservation %s associated with the NetworkContainer %s failed with error:%s."+
			"Marking this error as retriable.\n",
			repoNCConfig.ReservationID,
			repoNCConfig.NetworkContainerID,
			err.String())
		return nil, true, err
	} else if repoReservationConfiguration == nil {
		logger.Errorf("IPReservation for Id %s associated with the NetworkContainer %s failed with error:%s. "+
			"Marking this error as non retriable.\n",
			repoNCConfig.ReservationID,
			repoNCConfig.NetworkContainerID,
			err.String())

		msg := fmt.Sprintf(
			"IPReservation for Id %s associated with Network Container %s in Network %s not found.",
			repoNCConfig.ReservationID,
			repoNCConfig.NetworkContainerID,
			repoNCConfig.NetworkID)

		return nil, false, &api.JsonErr{
			Code:       api.ReservationNotFound,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
	}

	logger.Debugf("Retrieved reservation config:%s\n", repoReservationConfiguration)

	// Query NodeConfiguration table with nodeid and handle errors.
	repoNodeConfig, err := datamodel.TryQueryNodeConfiguration(nodeID)
	if err != nil {
		logger.Errorf("Retrieval of node configuration for Node %s associated with the NetworkContainer %s failed with error:%s. "+
			"Marking this error as retriable.\n",
			nodeID,
			repoNCConfig.NetworkContainerID,
			err.String())

		return nil, true, err
	} else if repoNodeConfig == nil {
		logger.Errorf("Configuration for Node %s associated with the NetworkContainer %s not found. "+
			"Marking this error as non retriable.\n",
			nodeID,
			repoNCConfig.NetworkContainerID)

		msg := fmt.Sprintf(
			"Configuration for Node %v associated with the NetworkContainer %v not found.",
			nodeID,
			repoNCConfig.NetworkContainerID)

		return nil, false, &api.JsonErr{
			Code:       api.NodeNotFound,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
	}

	logger.Debugf("Retrieved repo node config:%s\n", repoNodeConfig)

	repoSubnetConfiguration, err := datamodel.TryQuerySubnetConfiguration(
		repoNCConfig.NetworkID,
		repoReservationConfiguration.AllocationRequest.SubnetName)
	if err != nil {
		logger.Errorf("Retrieval of configuration for subnet %s associated with the Network %s failed with error:%s. "+
			"Marking this error as retriable.\n",
			repoReservationConfiguration.AllocationRequest.SubnetName,
			repoNCConfig.NetworkID,
			err.String())
		return nil, true, err
	} else if repoSubnetConfiguration == nil {
		logger.Errorf("Configuration for subnet %s associated with the Network %s not found. "+
			"Marking this error as non retriable.\n",
			repoReservationConfiguration.AllocationRequest.SubnetName,
			repoNCConfig.NetworkID)

		msg := fmt.Sprintf(
			"Configuration for Subnet %v in Network %v not found.",
			repoReservationConfiguration.AllocationRequest.SubnetName,
			repoNCConfig.NetworkID)
		return nil, false, &api.JsonErr{
			Code:       api.SubnetNotFound,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
	}

	logger.Debugf("Retrieved subnet config:%s\n", repoSubnetConfiguration)
	if repoSubnetConfiguration.Tags == nil {
		repoSubnetConfiguration.Tags = make([]nmagentcontracts.Tag, 0)
	}

	tags := make([]nmagentcontracts.Tag, len(repoSubnetConfiguration.Tags))
	for index, tag := range repoSubnetConfiguration.Tags {
		tags[index] = tag
	}

	repoNetworkConfiguration, err := datamodel.TryQueryNetworkConfiguration(repoNCConfig.NetworkID)
	if err != nil {
		logger.Errorf("Retrieval of configuration for Network %s failed with error:%s. "+
			"Marking this error as retriable.\n",
			repoNCConfig.NetworkID,
			err.String())
		return nil, false, err
	} else if repoNetworkConfiguration == nil {
		logger.Errorf("Configuration for Network %s not found. "+
			"Marking this error as non retriable.\n",
			repoNCConfig.NetworkID)

		msg := fmt.Sprintf(
			"Configuration for Network %v not found.",
			repoNCConfig.NetworkID)
		return nil, false, &api.JsonErr{
			Code:       api.NetworkNotFound,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
	}

	logger.Debugf("Retrieved network configuration:%s\n", repoNetworkConfiguration)
	dnsServers, err := datamodel.GetDNSServers(repoNetworkConfiguration, repoSubnetConfiguration)
	var associatedInterfaceID string
	assocInfo := repoNCConfig.GetAssociationInfo()

	if len(assocInfo.InterfaceID) == 0 {
		associatedInterfaceID = repoNodeConfig.NodeInfo.IPAddresses[0]
	} else {
		associatedInterfaceID = assocInfo.InterfaceID
	}

	ncGoalState := &NCGoalState{
		NetworkContainerID:    repoNCConfig.NetworkContainerID,
		Version:               repoNCConfig.CNSVersion,
		NetworkID:             repoNCConfig.NetworkID,
		AllocatedIPSubnet:     repoReservationConfiguration.AllocatedSubnet,
		BaseSubnet:            repoSubnetConfiguration.IPSubnet,
		BaseSubnetName:        repoSubnetConfiguration.SubnetName,
		AccessToken:           repoSubnetConfiguration.AccessToken,
		GatewayIP:             repoSubnetConfiguration.Gateway,
		DNSServers:            dnsServers,
		AssociatedNodeID:      nodeID,
		AssociatedInterfaceID: associatedInterfaceID,
		NodeInterfaceID:       repoNodeConfig.NodeInfo.IPAddresses[0],
		CNetSubnets:           strings.Split(repoNetworkConfiguration.CNetSpace, ","),
		VNetSubnets:           strings.Split(repoNetworkConfiguration.VNetSpace, ","),
		TagsInfo:              tags,
		ContainerType:         assocInfo.ContainerType,
		OrchestratorContext:   assocInfo.OrchestratorContext,
		VnetVersion:           repoNetworkConfiguration.Version,
		VlanID:                int(repoNodeConfig.NetworkVlanIDMap[repoNCConfig.NetworkID])}

	return ncGoalState, false, nil
}

// Post request to host agentg to express the intent for joining a network.
func postRequestToAgentForNetworkJoin(networkID string) (bool, *api.JsonErr) {
	joinURL := fmt.Sprintf(
		config.HostAgentSettings.ConnectionStrings.JoinNetworkURLFmt,
		config.HostAgentSettings.WireServerIP,
		networkID)

	logger.Debugf("postRequestToAgentForNetworkJoin url %s", joinURL)
	// Empty body is required as wireserver cannot handle a post without the body.
	var body bytes.Buffer
	json.NewEncoder(&body).Encode("")
	joinResponse, joinRespErr := http.Post(joinURL, "application/json", &body)
	if joinRespErr != nil {
		msg := fmt.Sprintf(
			"Join for network [%s] failed with reponse error [%s]. Marking this error as retriable",
			networkID,
			joinRespErr.Error())
		logger.Errorf("postRequestToAgentForNetworkJoin %s", msg)

		return true, &api.JsonErr{
			Code:       api.JoinDenied,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	defer joinResponse.Body.Close()

	if joinResponse.StatusCode == http.StatusProcessing {
		msg := fmt.Sprintf(
			"Join for network [%s] is pending. Marking this error as retriable",
			networkID)

		return true, &api.JsonErr{
			Code:       api.JoinPending,
			Text:       msg,
			HTTPStatus: http.StatusProcessing}
	}

	if joinResponse.StatusCode == http.StatusUnauthorized {
		msg := fmt.Sprintf("Join for network [%s] is unauthorized. Marking this error as non retriable.", networkID)
		return false, &api.JsonErr{
			Code:       api.NetworkJoinFailed,
			Text:       msg,
			HTTPStatus: http.StatusUnauthorized}
	}

	if joinResponse.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Join for network [%s] failed with status [%d]. Marking this error as retriable.",
			networkID,
			joinResponse.StatusCode)
		return true, &api.JsonErr{
			Code:       api.NetworkJoinFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	return false, nil
}

// Retrieve network configuration from host agent.
func getNetworkConfigurationFromAgent(networkID string) (bool, *api.JsonErr, *nmagentcontracts.VirtualNetworkResponse) {
	getURL := fmt.Sprintf(
		config.HostAgentSettings.ConnectionStrings.GetNetworkConfigurationURLFmt,
		config.HostAgentSettings.WireServerIP,
		networkID)
	getResponse, getRespErr := http.Get(getURL)
	if getRespErr != nil {
		msg := fmt.Sprintf(
			"Retrieval of network configuration for network [%s] failed with error [%s]. Marking this error as retriable.",
			networkID,
			getRespErr.Error())

		logger.Errorf("getNetworkConfigurationFromAgent failed with error %s", msg)
		return true, &api.JsonErr{
			Code:       api.NetworkConfigurationRetrievalFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil
	}

	defer getResponse.Body.Close()

	var networkConfigurationResponse nmagentcontracts.VirtualNetworkResponse
	decodeRespErr := utils.DecodeResponse(getResponse, &networkConfigurationResponse)
	if decodeRespErr != nil {
		msg := fmt.Sprintf("Failed to decode get virtual network configuration response for network %s due to error %s. "+
			"Marking this error as retriable.",
			networkID,
			decodeRespErr.Error())
		logger.Errorf("getNetworkConfigurationFromAgent failed with error  %s\n", msg)

		return true, &api.JsonErr{
			Code:       api.DeocodeResponseFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil
	}

	logger.Debugf("Received response for network %s Configuration:%s\n", networkID, networkConfigurationResponse.String())
	getNetworkResponseCode, parseErr := strconv.ParseInt(networkConfigurationResponse.ResponseCode, 10, 64)
	if parseErr != nil {
		msg := fmt.Sprintf(
			"Failed to convert response code %s  to integer when parsing network configuration response for network %s. "+
				"Marking this error as retriable.",
			networkConfigurationResponse.ResponseCode,
			networkID)
		logger.Errorf("getNetworkConfigurationFromAgent failed with error  %s\n", msg)

		return true, &api.JsonErr{
			Code:       api.ParseResponseCodeFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil
	}

	if getNetworkResponseCode == http.StatusProcessing {
		msg := fmt.Sprintf(
			"GetNetworkConfiguration for network %s is pending", networkID)
		logger.Errorf("%s\n", msg)

		return true, &api.JsonErr{
			Code:       api.ProcessingPending,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil
	}

	if getNetworkResponseCode == http.StatusUnauthorized {
		msg := fmt.Sprintf("Join for network %s is unauthorized", networkID)
		logger.Errorf("getNetworkConfigurationFromAgent failed with error %s\n", msg)

		return false, &api.JsonErr{
			Code:       api.NetworkJoinFailed,
			Text:       msg,
			HTTPStatus: http.StatusForbidden}, nil
	}

	if getNetworkResponseCode != http.StatusOK {
		msg := fmt.Sprintf("Failed to retrieve network due to error %d", getNetworkResponseCode)

		logger.Errorf("getNetworkConfigurationFromAgent failed with error %s\n", msg)
		return false, &api.JsonErr{
			Code:       api.NetworkJoinFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil
	}

	logger.Debugf("Retrieved virtual network configuration %s\n", networkConfigurationResponse)
	return false, nil, &networkConfigurationResponse
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
			logger.Errorf(
				"Failed to retrieve subnet configuration for subnet %s in network %s due to error %s\n",
				subnetName,
				repoNetworkConfig.NetworkID,
				subnetCfgErr)

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
			// (i.e) no reservations exist, then marke the subnet for update.
			if len(repoSubnetConfig.Reservations) == 0 {
				logger.Debugf("Subnet %s in configuration does not have any allocations\n", subnet.SubnetName)
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
				logger.Errorf("%s\n", msg)

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
				logger.Errorf("%s\n", msg)

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
				logger.Errorf("%s\n", msg)

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
		if !exists {
			if len(repoSubnetConfig.Reservations) > 0 {
				msg := fmt.Sprintf(
					"Subnet %s is removed but there are allocations. SubnetConfig:%s",
					repoSubnetConfig.SubnetName,
					repoSubnetConfig.String())
				logger.Errorf("%s\n", msg)

				return false, &api.JsonErr{
					Code:       api.NetworkJoinFailed,
					Text:       msg,
					HTTPStatus: http.StatusInternalServerError}, nil, nil, nil
			}

			subnetsToRemove[subnetName] = true
		}
	}

	return false, nil, subnetsToAdd, subnetsToRemove, subnetsToUpdate
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
	// add a dummy configuration and mark it as pendiong processing once
	// a request is received.
	repoNetworkConfiguration, err := datamodel.TryQueryNetworkConfiguration(networkID)
	if err != nil {
		logger.Errorf("Network configuration retrieval for network %s failed with error %s\n",
			networkID,
			err.String())
		return false, err
	}

	if repoNetworkConfiguration == nil {
		logger.Debugf("Failed to find configuration in repository for network %s\n", networkID)
		return true, nil
	}

	isRetriableError, failureDetail = postRequestToAgentForNetworkJoin(networkID)
	if failureDetail != nil {
		logger.Errorf("postRequestToAgentForNetworkJoin failed with error: %s", failureDetail.String())
		if isRetriableError {
			return false, failureDetail
		}
		goto Cleanup
	}

	isRetriableError, failureDetail, networkResponse = getNetworkConfigurationFromAgent(networkID)
	if failureDetail != nil {
		logger.Errorf("getNetworkConfigurationFromAgent failed with error: %s", failureDetail.String())
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
		logger.Errorf("validateAndRetrieveNetworkConfigurationUpdate failed with error: %s", failureDetail.String())
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
		ipSubnetMap[subnet.SubnetName] =
			datamodel.GetSubnetKey(networkID, subnet.SubnetName)
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
	logger.Debugf("Adding vnetconfig:%v\n", networkResponse)
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
		logger.Errorf("Failed to persist network configuration for network %s due to error %s",
			networkID,
			failureDetail.String())
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
			Tags:              tags}
		failureDetail = datamodel.PutSubnetConfiguration(&subnetConfig)
		if failureDetail != nil {
			goto Cleanup
		}
	}

	for subnetName, currentRepoSubnetCfg := range subnetsToUpdate {
		ipSubnetMap[subnetName] = datamodel.GetSubnetKey(networkID, subnetName)
		updatedSubnetCfg := subnetsInUpdatedResponse[subnetName]
		currentRepoSubnetCfg.Tags = updatedSubnetCfg.Tags
		datamodel.PutSubnetConfiguration(currentRepoSubnetCfg)
		if failureDetail != nil {
			goto Cleanup
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
		err = datamodel.PersistNetworkConfiguration(repoNetworkConfiguration)
		if err != nil {
			return false, err
		}
	}

	logger.Debugf("Successfully added vnetconfig:%s\n", repoNetworkConfiguration.String())
	return true, nil
}

func generateAndPersistImplicitReservationRequest(repoNCConfig *datamodel.NCConfiguration) (
	*datamodel.ReservationConfiguration, *api.JsonErr) {
	reservationConfiguration := datamodel.ReservationConfiguration{
		ReservationID:     repoNCConfig.ReservationID,
		NetworkID:         repoNCConfig.NetworkID,
		ReservationState:  datamodel.Allocate,
		Status:            datamodel.Pending,
		AllocationRequest: repoNCConfig.RequestV2.AllocationRequest,
		AssociatedNCID:    repoNCConfig.NetworkContainerID}

	nodeID := repoNCConfig.GetAssociationInfo().NodeID
	err := datamodel.PutIPReservationConfiguration(&reservationConfiguration)
	if err != nil {
		return nil, err
	}

	err = datamodel.UpdateNodeConfigurationNetworkContainerSet(
		nodeID,
		reservationConfiguration.NetworkID,
		reservationConfiguration.AssociatedNCID,
		reservationConfiguration.ReservationID,
		repoNCConfig.RequestV2.AssociationInfo.ContainerType,
		repoNCConfig.RequestV2.AssociationInfo.ContainerType == cns.AzureContainerInstance,
		false)
	if err != nil {
		logger.Errorf("Failed to add network container to %s to node configuration:%s. Error %v\n",
			reservationConfiguration.AssociatedNCID,
			nodeID,
			err)
		return nil, err
	}

	reservationKey := datamodel.GetReservationKey(reservationConfiguration.NetworkID, reservationConfiguration.ReservationID)
	requestContext := datamodel.RequestContext{
		Type:      datamodel.ReservationRequest,
		NetworkID: repoNCConfig.NetworkID,
		NodeID:    nodeID}

	err = datamodel.AddRequestToPendingQueue(reservationKey, &requestContext)
	if err != nil {
		logger.Errorf("Failed to add the request %s to the the pending process queue\n", requestContext)
		return nil, err
	}

	return &reservationConfiguration, nil
}

func processNetworkContainerV1(repoNCConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	var markPrepareFailed bool
	var failureDetail *api.JsonErr
	var reservationConfiguration *datamodel.ReservationConfiguration
	associationInfo := repoNCConfig.GetAssociationInfo()
	err := datamodel.UpdateNodeConfigurationNetworkContainerSet(
		associationInfo.NodeID,
		repoNCConfig.NetworkID,
		repoNCConfig.NetworkContainerID,
		repoNCConfig.RequestV1.ReservationID,
		associationInfo.ContainerType,
		associationInfo.ContainerType == cns.AzureContainerInstance,
		false)
	if err != nil {
		if err.HTTPStatus == http.StatusInternalServerError {
			return false, err
		}
		markPrepareFailed = true
		failureDetail = err
		goto Cleanup
	}

	reservationConfiguration, err = datamodel.QueryIPReservation(
		repoNCConfig.NetworkID,
		repoNCConfig.ReservationID)
	if err != nil {
		logger.Errorf(
			"Failed to retrieve reservation configuration for resevation %s.Error %s\n",
			repoNCConfig.ReservationID,
			err.String())

		if err.HTTPStatus == http.StatusInternalServerError {
			return false, err
		}

		markPrepareFailed = true
		failureDetail = err
		goto Cleanup
	}

	reservationConfiguration.AssociatedNCID = repoNCConfig.NetworkContainerID
	err = datamodel.PutIPReservationConfiguration(reservationConfiguration)
	if err != nil {
		logger.Errorf(
			"Failed to put reservation configuration %s. Error %s\n",
			reservationConfiguration.String(),
			err.String())
		return false, err
	}

	repoNCConfig.State = datamodel.Reserve
	repoNCConfig.Status = datamodel.Pending

Cleanup:

	if markPrepareFailed {
		repoNCConfig.Status = datamodel.Failed
		logger.Errorf("IPReservation failure Detail %s\n", failureDetail.String())
		repoNCConfig.FailureDetail = *failureDetail
	}

	err = datamodel.PersistNCConfiguration(repoNCConfig, false)
	if err != nil {
		return false, err
	}

	return markPrepareFailed, nil
}

func processNetworkContainerV2(repoNCConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	var markPrepareFailed bool
	var failureDetail *api.JsonErr

	if len(repoNCConfig.ReservationID) == 0 {
		reservationIDUUID, err := uuid.NewUUID()
		if err != nil {
			return false, &api.JsonErr{
				Code:       api.StatusInternalServerError,
				HTTPStatus: http.StatusInternalServerError,
				Text: fmt.Sprintf(
					"Failed to generate UUID due to error:%s", err.Error())}
		}

		repoNCConfig.ReservationID = reservationIDUUID.String()
	}

	reservationConfiguration, err := datamodel.TryQueryIPReservation(
		repoNCConfig.NetworkID,
		repoNCConfig.ReservationID)
	if err != nil {
		return false, err
	}

	if reservationConfiguration == nil {
		logger.Debugf(
			"Creating Reservation NetworkID:%s ReservationID:%s\n",
			repoNCConfig.NetworkID,
			repoNCConfig.ReservationID)

		reservationConfiguration, err = generateAndPersistImplicitReservationRequest(
			repoNCConfig)
		if err != nil {
			return false, err
		}
	} else {
		if reservationConfiguration.Status == datamodel.Pending {
			reservationKey := datamodel.GetReservationKey(
				repoNCConfig.NetworkID,
				repoNCConfig.ReservationID)
			requestCfg, err := datamodel.TryGetRequestFromPendingQueue(
				reservationKey)
			if err != nil {
				return false, err
			}

			if requestCfg == nil {
				requestContext := datamodel.RequestContext{
					Type:      datamodel.ReservationRequest,
					NetworkID: repoNCConfig.NetworkID,
					NodeID:    repoNCConfig.GetAssociationInfo().NodeID}

				err := datamodel.AddRequestToPendingQueue(
					reservationKey,
					&requestContext)
				if err != nil {
					return false, err
				}
			}
		}
	}

	repoNCConfig.State = datamodel.Reserve
	repoNCConfig.Status = datamodel.Pending

	if markPrepareFailed {
		repoNCConfig.Status = datamodel.Failed
		repoNCConfig.FailureDetail = *failureDetail
	}

	err = datamodel.PersistNCConfiguration(repoNCConfig, false)
	if err != nil {
		return false, err
	}

	return markPrepareFailed, nil
}

func prepareNetworkContainer(repoNCConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	if repoNCConfig.Status == datamodel.Failed {
		logger.Errorf("NC in failed state during prepare. Failure detail %s\n", repoNCConfig.FailureDetail.String())
		return true, nil
	}

	if !repoNCConfig.ImplicitReservation {
		return processNetworkContainerV1(repoNCConfig)
	}

	return processNetworkContainerV2(repoNCConfig)
}

func processNetworkContainerReserve(
	repoNCConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	var markProcessingFailed bool
	var failureDetail *api.JsonErr
	repoReservation, err := datamodel.TryQueryIPReservation(
		repoNCConfig.NetworkID,
		repoNCConfig.ReservationID)
	if err != nil {
		return false, err
	}

	if repoNCConfig == nil {
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

	if repoReservation.Status == datamodel.Completed {
		repoNCConfig.State = datamodel.Associate
		repoNCConfig.Status = datamodel.Pending

		repoSubnetConfiguration, err := datamodel.TryQuerySubnetConfiguration(
			repoNCConfig.NetworkID,
			repoReservation.AllocationRequest.SubnetName)
		if err != nil {
			logger.Errorf("Retrieval of configuration for subnet %s associated with the Network %s failed with error:%s. "+
				"Marking this error as retriable.\n",
				repoReservation.AllocationRequest.SubnetName,
				repoNCConfig.NetworkID,
				err.String())
			return true, err
		}
		if repoSubnetConfiguration == nil {

			msg := fmt.Sprintf(
				"Configuration for Subnet %s in Network %v not found.",
				repoReservation.AllocationRequest.SubnetName,
				repoNCConfig.NetworkID)
			logger.Errorf("%s. Marking this error as non retriable.\n", msg)

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
		repoNCConfig.Status = datamodel.Failed
		repoNCConfig.FailureDetail = repoReservation.FailureDetail
	}

Cleanup:
	if markProcessingFailed {
		repoNCConfig.Status = datamodel.Failed
		repoNCConfig.FailureDetail = *failureDetail
	}

	err = datamodel.PersistNCConfiguration(repoNCConfig, false)
	if err != nil {
		return false, err
	}

	return markProcessingFailed, nil
}

func processNetworkContainerUnReserve(repoNCConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	err := datamodel.PurgeNetworkContainerAssociations(repoNCConfig, repoNCConfig.GetAssociationInfo().ContainerType == cns.AzureContainerInstance)
	if err != nil {
		return false, err
	}

	repoNCConfig.State = datamodel.Purge
	repoNCConfig.Status = datamodel.Pending

	err = datamodel.PersistNCConfiguration(repoNCConfig, false)
	if err != nil {
		return false, err
	}

	return false, nil
}

func processNetworkContainerPurge(repoNCConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	err := datamodel.PurgeNetworkContainerAssociations(repoNCConfig, repoNCConfig.GetAssociationInfo().ContainerType == cns.AzureContainerInstance)
	if err != nil {
		return false, err
	}

	err = datamodel.DeleteNetworkContainer(repoNCConfig.NetworkID, repoNCConfig.NetworkContainerID)
	if err != nil {
		return false, err
	}

	return true, nil
}

func hasTagsChanged(resolvedTags map[string]datamodel.TagDefinition, currentTags map[string]uint64) bool {
	if len(resolvedTags) != len(currentTags) {
		return true
	}

	for tagName, tagDefinition := range resolvedTags {
		currentVersion, found := currentTags[tagName]
		if !found {
			return true
		}

		if currentVersion != tagDefinition.Version {
			return true
		}
	}

	return false
}

func processNetworkContainerAssociate(
	repoNCConfig *datamodel.NCConfiguration,
	sessionInfo map[string]string) (bool, *api.JsonErr) {
	ncGoalState, isRetriableError, ncGoalStateErr := getNCGoalState(repoNCConfig)
	if ncGoalStateErr != nil {
		if isRetriableError {
			return false, ncGoalStateErr
		}

		repoNCConfig.Status = datamodel.Failed
		repoNCConfig.FailureDetail = *ncGoalStateErr
		return true, datamodel.PersistNCConfiguration(repoNCConfig, false)
	}

	cnetAddressesCount := len(ncGoalState.CNetSubnets)
	resolvedTags := make(map[string]datamodel.TagDefinition)
	for _, tag := range ncGoalState.TagsInfo {
		tagDefinition, tagErr := RegisterAndGetTagDefinition(tag)
		if tagErr != nil {
			return false, tagErr
		}

		tagKey := datamodel.GetTagDefintionKey(tag.Type, tag.Name)
		resolvedTags[tagKey] = tagDefinition
		cnetAddressesCount += len(resolvedTags[tagKey].Values)
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
		logger.Errorf("%s, Marking as non retriable error\n", msg)
		failureDetail := api.JsonErr{
			Code:       api.InvalidIPSubnet,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
		repoNCConfig.Status = datamodel.Failed
		repoNCConfig.FailureDetail = failureDetail
		return true, datamodel.PersistNCConfiguration(repoNCConfig, false)
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
		logger.Debugf("Parsing cnetAddress:%s\n", cnetAddressString)
		ipAddressString, prefix, parserErr := utils.GetIPAddressStringAndPrefix(cnetAddressString)
		if parserErr != nil {
			msg := fmt.Sprintf("Failed to parse Vnet subnet string %v due to error %v",
				cnetAddressString,
				err.Error())
			logger.Errorf("%s, Marking as non retriable error\n", msg)
			failureDetail := api.JsonErr{
				Code:       api.InvalidIPSubnet,
				Text:       msg,
				HTTPStatus: http.StatusBadRequest}
			repoNCConfig.Status = datamodel.Failed
			repoNCConfig.FailureDetail = failureDetail
			return true, datamodel.PersistNCConfiguration(repoNCConfig, false)
		}

		cnsIPSubnet := cns.IPSubnet{IPAddress: ipAddressString, PrefixLength: uint8(prefix)}
		cnsCnetAddresses[index] = cnsIPSubnet
	}

	info := &cns.CreateNetworkContainerRequest{
		Version:                    strconv.FormatUint(repoNCConfig.CNSVersion, 10),
		NetworkContainerType:       ncGoalState.ContainerType,
		NetworkContainerid:         "Swift_" + ncGoalState.NetworkContainerID,
		PrimaryInterfaceIdentifier: ncGoalState.AssociatedInterfaceID,
		IPConfiguration:            ipConfig,
		AuthorizationToken:         ncGoalState.AccessToken,
		CnetAddressSpace:           cnsCnetAddresses,
		OrchestratorContext:        ncGoalState.OrchestratorContext,
		MultiTenancyInfo:           cns.MultiTenancyInfo{EncapType: cns.Vlan, ID: ncGoalState.VlanID}}

	json.NewEncoder(&body).Encode(info)
	createOrUpdateNCURL := fmt.Sprintf(
		config.CNSSettings.ConnectionStrings.CreateOrUpdateNetworkContainerURLFmt,
		ncGoalState.NodeInterfaceID)

	// Release the session locks when calling in to cns
	sessionManager.EndWriteSession(sessionInfo)
	sessionLockReleased := true
	defer func() {
		if sessionLockReleased {
			sessionManager.BeginWriteSession(sessionInfo)
		}
	}()

	w, respErr := http.Post(createOrUpdateNCURL, "application/json; charset=utf-8", &body)

	// Reacquire the session locks post calling in to cns
	sessionManager.BeginWriteSession(sessionInfo)
	sessionLockReleased = false

	if respErr != nil {
		msg := fmt.Sprintf("Post to URL %s for NC %v failed with error %v",
			createOrUpdateNCURL,
			ncGoalState.NetworkContainerID,
			respErr.Error())
		logger.Errorf("%s, Marking as retriable error\n", msg)
		return false, &api.JsonErr{
			Code:       api.AssociationFailed,
			Text:       respErr.Error(),
			HTTPStatus: http.StatusBadRequest}
	}

	defer w.Body.Close()
	respErr = utils.DecodeResponse(w, &resp)
	if respErr != nil {
		msg := fmt.Sprintf("Associate NC %v to node %v failed with error %s",
			ncGoalState.NetworkContainerID,
			ncGoalState.NodeInterfaceID,
			respErr.Error())
		logger.Errorf("%s, Marking as retriable error\n", msg)

		return false, &api.JsonErr{
			Code:       api.AssociationFailed,
			Text:       respErr.Error(),
			HTTPStatus: http.StatusBadRequest}
	}

	if resp.Response.ReturnCode != 0 {
		msg := fmt.Sprintf("Associate NC %v to node %v failed with response %+v",
			ncGoalState.NetworkContainerID,
			ncGoalState.NodeInterfaceID,
			resp.Response)
		logger.Errorf("%s, Marking as non retriable error\n", msg)
		failureDetail := api.JsonErr{
			Code:       api.AssociationFailed,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
		repoNCConfig.Status = datamodel.Failed
		repoNCConfig.FailureDetail = failureDetail
		return true, datamodel.PersistNCConfiguration(repoNCConfig, false)
	}

	if repoNCConfig.State == datamodel.RefreshAssociation {
		repoNCConfig.State = datamodel.Program
	} else {
		repoNCConfig.State = datamodel.Publish
	}

	repoNCConfig.ProgrammedVnetVersion = ncGoalState.VnetVersion
	repoNCConfig.Status = datamodel.Pending
	return false, datamodel.PersistNCConfiguration(repoNCConfig, false)
}

func processNetworkContainerDisAssociate(
	repoNCConfig *datamodel.NCConfiguration,
	sessionInfo map[string]string) (bool, *api.JsonErr) {
	nodeID := repoNCConfig.GetAssociationInfo().NodeID

	// Query NodeConfiguration table with nodeID and handle errors.
	repoNodeConfig, err := datamodel.TryQueryNodeConfiguration(nodeID)

	if err != nil {
		logger.Errorf("Retrieval of node configuration for node %s failed with error %v."+
			"Marking error as retriable.\n",
			nodeID,
			err)
		return false, err
	} else if repoNodeConfig == nil {
		msg := fmt.Sprintf("Node configuration for node %v not found.",
			nodeID)

		logger.Errorf("%s, Marking as non retriable error\n", msg)
		failureDetail := api.JsonErr{
			Code:       api.NodeNotFound,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
		repoNCConfig.Status = datamodel.Failed
		repoNCConfig.FailureDetail = failureDetail
		return true, datamodel.PersistNCConfiguration(repoNCConfig, false)
	}

	primaryInterfaceID := repoNodeConfig.NodeInfo.IPAddresses[0]
	info := &cns.DeleteNetworkContainerRequest{
		NetworkContainerid: "Swift_" + repoNCConfig.NetworkContainerID,
	}

	sessionManager.EndWriteSession(sessionInfo)
	sessionLockReleased := true
	defer func() {
		if sessionLockReleased {
			sessionManager.BeginWriteSession(sessionInfo)
		}
	}()
	var body bytes.Buffer
	json.NewEncoder(&body).Encode(info)
	deleteNCURL := fmt.Sprintf(
		config.CNSSettings.ConnectionStrings.DeleteNetworkContainerURLFmt,
		primaryInterfaceID)

	w, respErr := http.Post(deleteNCURL, "application/json", &body)
	sessionManager.BeginWriteSession(sessionInfo)
	sessionLockReleased = false
	if respErr != nil {
		logger.Errorf("Failed to disassociate network container %s on node %v due to error. Marked as retriable.\n",
			info.NetworkContainerid,
			primaryInterfaceID)
		return false, &api.JsonErr{
			Code:       api.DisAssociationFailed,
			Text:       respErr.Error(),
			HTTPStatus: http.StatusInternalServerError}
	}

	defer w.Body.Close()

	var resp cns.DeleteNetworkContainerResponse
	decodeErr := utils.DecodeResponse(w, &resp)
	if decodeErr != nil {
		logger.Errorf("Failed to decode disassociate response for request with network container Id:%s. "+
			"Marking as retriable error. \n",
			repoNCConfig.NetworkContainerID)
		return false, &api.JsonErr{
			Code:       api.DisAssociationFailed,
			Text:       decodeErr.Error(),
			HTTPStatus: http.StatusBadRequest}
	}

	if resp.Response.ReturnCode != 0 {
		msg := fmt.Sprintf("Disassociate request associated with network container Id:%v returned status %v. "+
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

		return false, datamodel.PersistNCConfiguration(repoNCConfig, false)
	}

	logger.Debugf("DisAssociated NC %s from node %s\n",
		repoNCConfig.NetworkContainerID,
		repoNCConfig.GetAssociationInfo().NodeID)
	repoNCConfig.State = datamodel.UnReserve
	repoNCConfig.Status = datamodel.Pending
	return false, datamodel.PersistNCConfiguration(repoNCConfig, false)
}

func processNetworkContainerPublish(repoNCConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	ncGoalState, isRetriableError, ncGoalStateErr := getNCGoalState(repoNCConfig)
	if ncGoalStateErr != nil {
		if isRetriableError {
			return false, ncGoalStateErr
		}

		repoNCConfig.Status = datamodel.Failed
		repoNCConfig.FailureDetail = *ncGoalStateErr
		return true, datamodel.PersistNCConfiguration(repoNCConfig, false)
	}

	var body bytes.Buffer
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
		ncGoalState.NodeInterfaceID,
		ncGoalState.NetworkContainerID,
		ncGoalState.AccessToken)
	logger.Debugf("Create Url:%s, body:%s\n", createNCURL, ncRequest.String())

	resp, respErr := http.Post(createNCURL, "application/json", &body)
	if respErr != nil {
		logger.Errorf("Failed to publish network container goal state for Id %s to host agent with error %v "+
			"Marking as retriable error \n",
			repoNCConfig.NetworkContainerID,
			respErr.Error())
		return false, &api.JsonErr{
			Code:       api.PublishingFailed,
			Text:       respErr.Error(),
			HTTPStatus: http.StatusInternalServerError}
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Errorf("Failed to publish network container goal state for Id %s to host agent with error code %d "+
			"Marking as retriable error \n",
			repoNCConfig.NetworkContainerID,
			resp.StatusCode)
		return false, &api.JsonErr{
			Code:       api.PublishingFailed,
			Text:       resp.Status,
			HTTPStatus: http.StatusInternalServerError}
	}

	var createNetworkContainerResponse nmagentcontracts.CreateNetworkContainerResponse
	decodeRespErr := utils.DecodeResponse(resp, &createNetworkContainerResponse)
	if decodeRespErr != nil {
		msg := fmt.Sprintf("Failed to decode get put NC response for NC %s due to error %s.",
			repoNCConfig.NetworkContainerID,
			decodeRespErr.Error())
		return false, &api.JsonErr{
			Code:       api.DeocodeResponseFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	logger.Debugf("Received response for create network container response %s\n", createNetworkContainerResponse.String())
	putNCResponseCode, parseErr := strconv.ParseInt(createNetworkContainerResponse.ResponseCode, 10, 64)
	if parseErr != nil {
		msg := fmt.Sprintf(
			"Failed to convert response code %s  to integer when parsing put NC response for NC %s.",
			createNetworkContainerResponse.ResponseCode,
			repoNCConfig.NetworkContainerID)
		return false, &api.JsonErr{
			Code:       api.ParseResponseCodeFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	if putNCResponseCode != http.StatusOK {
		msg := fmt.Sprintf("Failed to publish NC with status code %v.", putNCResponseCode)
		repoNCConfig.Status = datamodel.Failed
		repoNCConfig.FailureDetail = api.JsonErr{
			Code:       api.PublishingFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
		return true, datamodel.PersistNCConfiguration(repoNCConfig, false)
	}

	logger.Debugf("CreateNetworkContainer NMAgent Response: %+v", resp)
	repoNCConfig.State = datamodel.Program
	repoNCConfig.Status = datamodel.Pending
	return false, datamodel.PersistNCConfiguration(repoNCConfig, false)
}

func processNetworkContainerUnPublish(repoNCConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	ncGoalState, isRetriableError, ncGoalStateErr := getNCGoalState(repoNCConfig)
	if ncGoalStateErr != nil {
		if isRetriableError {
			return false, ncGoalStateErr
		}

		repoNCConfig.Status = datamodel.Failed
		repoNCConfig.FailureDetail = *ncGoalStateErr
		return true, datamodel.PersistNCConfiguration(repoNCConfig, false)
	}

	// Empty body is required as wireserver cannot handle a post without the body.
	var body bytes.Buffer
	json.NewEncoder(&body).Encode("")
	deleteNCURL := fmt.Sprintf(
		config.HostAgentSettings.ConnectionStrings.DeleteNetworkContainerURLFmt,
		config.HostAgentSettings.WireServerIP,
		ncGoalState.NodeInterfaceID,
		ncGoalState.NetworkContainerID,
		ncGoalState.AccessToken)
	logger.Debugf("Unpublishing: Url:%s\n", deleteNCURL)

	resp, respErr := http.Post(deleteNCURL, "application/json", &body)
	if respErr != nil {
		logger.Errorf("Failed to unpublish network container goal state for Id %s to host agent with error %v "+
			"Marking as retriable error.\n",
			repoNCConfig.NetworkContainerID,
			respErr.Error())
		return false, &api.JsonErr{
			Code:       api.PublishingFailed,
			Text:       respErr.Error(),
			HTTPStatus: http.StatusInternalServerError}
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		logger.Errorf("Failed to publish network container goal state for Id %s to host agent with error code %d "+
			"Marking as retriable error \n",
			repoNCConfig.NetworkContainerID,
			resp.StatusCode)
		return false, &api.JsonErr{
			Code:       api.PublishingFailed,
			Text:       resp.Status,
			HTTPStatus: http.StatusInternalServerError}
	}

	if resp.StatusCode != http.StatusOK {
		logger.Errorf("Failed to unpublish network container goal state for Id %s to host agent with error code %d.\n",
			repoNCConfig.NetworkContainerID,
			resp.StatusCode)
		return false, &api.JsonErr{
			Code:       api.UnpublishingFailed,
			Text:       resp.Status,
			HTTPStatus: http.StatusInternalServerError}
	}

	var deleteNCResponse nmagentcontracts.DeleteNetworkContainerResponse
	decodeRespErr := utils.DecodeResponse(resp, &deleteNCResponse)
	if decodeRespErr != nil {
		msg := fmt.Sprintf("Failed to decode get delete NC response for NC %s due to error %s. Marking this error as retriable",
			repoNCConfig.NetworkContainerID,
			decodeRespErr.Error())
		return false, &api.JsonErr{
			Code:       api.DeocodeResponseFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	logger.Debugf("Received response for network response :%s for NC:%s\n",
		deleteNCResponse.String(),
		repoNCConfig.NetworkContainerID)
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

	logger.Debugf("Retrieved NC response:%v\n", deleteNCResponseCode)
	if deleteNCResponseCode != http.StatusOK {
		msg := fmt.Sprintf("Failed to unpublish NC with status code %v. Marking as non retriable.", deleteNCResponseCode)
		repoNCConfig.Status = datamodel.Failed
		repoNCConfig.FailureDetail = api.JsonErr{
			Code:       api.UnpublishingFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
		return false, datamodel.PersistNCConfiguration(repoNCConfig, false)
	}

	repoNCConfig.State = datamodel.DisAssociate
	repoNCConfig.Status = datamodel.Pending
	return false, datamodel.PersistNCConfiguration(repoNCConfig, false)
}

func processNetworkContainerRevert(repoNCConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	associationInfo := repoNCConfig.GetAssociationInfo()
	logger.Debugf("Reverting NC %s to node %s\n", repoNCConfig.NetworkContainerID, associationInfo.NodeID)
	repoNCConfig.State = datamodel.UnPublish
	repoNCConfig.Status = datamodel.Pending
	return true, datamodel.PersistNCConfiguration(repoNCConfig, false)
}

func processNetworkContainerProgram(repoNCConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	associationInfo := repoNCConfig.GetAssociationInfo()
	logger.Debugf("Programming NC %s to node %s\n", repoNCConfig.NetworkContainerID, associationInfo.NodeID)
	repoNCConfig.State = datamodel.Program
	repoNCConfig.Status = datamodel.Completed
	return true, datamodel.PersistNCConfiguration(repoNCConfig, false)
}

func processNetworkContainerRequest(
	networkID,
	networkContainerKey string,
	sessionInfo map[string]string) (bool, *api.JsonErr) {
	repoNCConfig, err := datamodel.TryQueryNetworkContainerByKey(networkContainerKey)
	if err != nil {
		logger.Errorf("Failed to get network container configuration for ID:%s. Error:%s\n",
			networkContainerKey,
			err.String())
		return false, err
	}

	// Check if NC has been marked for delete and ensure that we are in the
	// correct revert state.
	if repoNCConfig.MarkedForDelete {
		if !datamodel.IsDeleteInProgress(repoNCConfig.State) {
			revertState, err := datamodel.GetRevertState(repoNCConfig.State)
			if err != nil {
				return false, err
			}
			repoNCConfig.State = revertState
		}
	}
	switch repoNCConfig.State {
	case datamodel.Prepare:
		if repoNCConfig.Status == datamodel.Failed {
			return true, nil
		}
		return prepareNetworkContainer(repoNCConfig)

	case datamodel.Reserve:
		if repoNCConfig.Status == datamodel.Failed {
			return true, nil
		}
		return processNetworkContainerReserve(repoNCConfig)

	case datamodel.Associate:
		fallthrough

	case datamodel.RefreshAssociation:
		if repoNCConfig.Status == datamodel.Failed {
			return true, nil
		}

		return processNetworkContainerAssociate(repoNCConfig, sessionInfo)

	case datamodel.Publish:
		if repoNCConfig.Status == datamodel.Failed {
			return true, nil
		}

		return processNetworkContainerPublish(repoNCConfig)

	case datamodel.Program:
		if repoNCConfig.Status == datamodel.Failed {
			return true, nil
		}

		return processNetworkContainerProgram(repoNCConfig)

	case datamodel.Revert:
		return processNetworkContainerRevert(repoNCConfig)

	case datamodel.UnPublish:
		return processNetworkContainerUnPublish(repoNCConfig)

	case datamodel.DisAssociate:
		return processNetworkContainerDisAssociate(repoNCConfig, sessionInfo)

	case datamodel.UnReserve:
		return processNetworkContainerUnReserve(repoNCConfig)

	case datamodel.Purge:
		return processNetworkContainerPurge(repoNCConfig)

	}

	return true, nil
}

func monitorRequests() {
	monitorTimerChannel := time.NewTicker(time.Millisecond * 1000).C
	for {
		select {
		case <-monitorTimerChannel:
			if replicaInstance.IsReplicaPrimary() {
				processPendingRequests()
			}
			break

		case <-startProcessing:
			networkMonitorStartProcessing <- true
			tagMonitorStartProcessing <- true
			refreshNCStartProcessing <- true
			processPendingRequests()
			logger.Debugf("Received signal to process pending work\n")
			break

		case <-cancelProcessing:
			networkMonitorStopProcessing <- true
			tagMonitorStopProcessing <- true
			refreshNCStopProcessing <- true
			logger.Debugf("Received signal to exit\n")
			break
		}
	}
}

// Monitor that periodically processes pending reqguests in the work item queue.
func processPendingRequests() {
	requests, _ := datamodel.GetPendingRequests()
	if len(requests) == 0 {
		return
	}

	requestsToProcess := make([]string, 0, len(requests))
	for requestID, requestContext := range requests {
		if time.Now().UTC().Sub(requestContext.LastRetriedDateTime) > requestContext.RetryInterval {
			logger.Debugf("Processing request for requestID:[%s], %s\n", requestID, requestContext.String())
			requestsToProcess = append(requestsToProcess, requestID)
		}
	}

	maxConcurrentRequestsToProcess := 5
	if config.DNCSettings.ConcurrentRequestsToProcess != 0 {
		maxConcurrentRequestsToProcess = config.DNCSettings.ConcurrentRequestsToProcess
	}

	if len(requestsToProcess) < maxConcurrentRequestsToProcess {
		maxConcurrentRequestsToProcess = len(requestsToProcess)
	}

	// Dummy channel to coordinate the number of concurrent goroutines.
	// This channel should be buffered otherwise we will be immediately blocked
	// when trying to fill it.
	concurrentGoroutines := make(chan struct{}, maxConcurrentRequestsToProcess)
	// Fill the dummy channel with maxNbConcurrentGoroutines empty struct.
	for i := 0; i < maxConcurrentRequestsToProcess; i++ {
		concurrentGoroutines <- struct{}{}
	}

	// The done channel indicates when a single goroutine has
	// finished its job.
	done := make(chan bool)
	// The waitForAllJobs channel allows the main program
	// to wait until we have indeed done all the jobs.
	waitForAllJobs := make(chan bool)

	// Collect all the jobs, and since the job is finished, we can
	// release another spot for a goroutine.
	go func() {
		for i := 0; i < len(requestsToProcess); i++ {
			<-done
			// Say that another goroutine can now start.
			concurrentGoroutines <- struct{}{}
		}
		// We have collected all the jobs, the program
		// can now terminate
		waitForAllJobs <- true
	}()

	// Try to start nbJobs jobs
	for i := 0; i < len(requestsToProcess); i++ {
		logger.Debugf("ID: %v: waiting to be processed", requestsToProcess[i])
		// Try to receive from the concurrentGoroutines channel. When we have something,
		// it means we can start a new goroutine because another one finished.
		// Otherwise, it will block the execution until an execution
		// spot is available.
		<-concurrentGoroutines
		logger.Debugf("Prcessing ID: %v", requestsToProcess[i])
		requestID := requestsToProcess[i]
		requestContext := requests[requestID]
		go func(requestID string, requestContext datamodel.RequestContext) {

			processRequest(requestID, &requestContext)
			logger.Debugf("ID:%v processed", requestID)
			done <- true
		}(requestID, requestContext)
	}

	// Wait for all jobs to finish
	<-waitForAllJobs
}
