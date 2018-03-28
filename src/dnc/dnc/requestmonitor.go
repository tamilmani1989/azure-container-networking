package main

import (
	"bytes"
	"dnc/allocationhelper"
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

// NetworkContainerGoalState - Struct to cache the Network Container Goal State
type NCGoalState struct {
	NetworkContainerID    string
	Version               string
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
}

// Monitor that periodically processes pending requests in the work item queue.
func processPendingRequests() {
	requests, _ := datamodel.GetPendingRequests()

	for key, request := range requests {
		var removeWorkItem bool
		var err *api.JsonErr
		logger.Log("Processing request : %s\n", request.String())

		switch request.Type {
		case datamodel.VirtualNetworkRequest:
			removeWorkItem, err = processVirtualNetworksPendingJoin(request.NetworkID)
			break

		case datamodel.ReservationRequest:
			removeWorkItem, err = processIPReservationAllocation(request.NetworkID, key)
			break

		case datamodel.NetworkContainerRequest:
			removeWorkItem, err = processNetworkContainerRequest(request.NetworkID, key)
			break

		}

		if err == nil && removeWorkItem {
			datamodel.RemovePendingRequest(key)
		}
	}
}

// getNCGoalState - Get NC goal state from different tables.
// Return the following.
// 1. Constructed NC goal state
// 2. If there was an error, True if the error is retriable , False otherwise.
// 3. Error.
func getNCGoalState(repoNCConfig *datamodel.NCConfiguration) (*NCGoalState, bool, *api.JsonErr) {
	logger.Log("Retrieving Network Container Goal state for container %s\n", repoNCConfig.NetworkContainerID)
	nodeID := repoNCConfig.GetAssociationInfo().NodeID

	repoReservationConfiguration, err :=
		datamodel.TryQueryIPReservation(repoNCConfig.NetworkID, repoNCConfig.ReservationID)
	if err != nil {
		logger.Log("Retrieval of IPReservation %s associated with the NetworkContainer %s failed with error:%s.\n"+
			"Marking this error as retriable.\n",
			repoNCConfig.ReservationID,
			repoNCConfig.NetworkContainerID,
			err.String())
		return nil, true, err
	} else if repoReservationConfiguration == nil {
		logger.Log("IPReservation for Id %s associated with the NetworkContainer %s failed with error:%s. Marking this error as non retriable.\n",
			repoNCConfig.ReservationID,
			repoNCConfig.NetworkContainerID,
			err.String())

		msg := fmt.Sprintf(
			"IPReservation for Id %s associated with Network Container %s in Network %s not found.\n",
			repoNCConfig.ReservationID,
			repoNCConfig.NetworkContainerID,
			repoNCConfig.NetworkID)

		return nil, false, &api.JsonErr{
			Code:       api.ReservationNotFound,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
	}
	logger.Log("Retrieved reservation config: %+v\n", repoReservationConfiguration)

	// Query NodeConfiguration table with nodeid and handle errors.
	repoNodeConfig, err := datamodel.TryQueryNodeConfiguration(nodeID)
	if err != nil {
		logger.Log("Retrieval of node configuration for Node %s associated with the NetworkContainer %s failed with error:%s. Marking this error as retriable.\n",
			nodeID,
			repoNCConfig.NetworkContainerID,
			err.String())

		return nil, true, err
	} else if repoNodeConfig == nil {
		logger.Log("Configuration for Node %s associated with the NetworkContainer %s not found. Marking this error as non retriable.\n",
			nodeID,
			repoNCConfig.NetworkContainerID)

		msg := fmt.Sprintf(
			"Configuration for Node %s associated with the NetworkContainer %s not found.\n",
			nodeID,
			repoNCConfig.NetworkContainerID)

		return nil, false, &api.JsonErr{
			Code:       api.NodeNotFound,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
	}

	logger.Log("Retrieved repo node config:%+v\n", repoNodeConfig)

	repoSubnetConfiguration, err := datamodel.TryQuerySubnetConfiguration(
		repoNCConfig.NetworkID,
		repoReservationConfiguration.AllocationRequest.SubnetName)
	if err != nil {
		logger.Log("Retrieval of configuration for subnet %s associated with the Network %s failed with error:%s. Marking this error as retriable.\n",
			repoNCConfig.NetworkID,
			repoReservationConfiguration.AllocationRequest.SubnetName,
			err.String())
		return nil, true, err
	} else if repoSubnetConfiguration == nil {
		logger.Log("Configuration for subnet %s associated with the Network %s not found. Marking this error as non retriable.\n",
			repoReservationConfiguration.AllocationRequest.SubnetName,
			repoNCConfig.NetworkID)

		msg := fmt.Sprintf(
			"Configuration for Subnet %s in Network %s not found.\n",
			repoReservationConfiguration.AllocationRequest.SubnetName,
			repoNCConfig.NetworkID)
		return nil, false, &api.JsonErr{
			Code:       api.SubnetNotFound,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
	}

	logger.Log("Retrieved subnet config:%s\n", repoSubnetConfiguration)

	repoNetworkConfiguration, err := datamodel.TryQueryNetworkConfiguration(repoNCConfig.NetworkID)
	if err != nil {
		logger.Log("Retrieval of configuration for Network %s failed with error:%s. "+
			"Marking this error as retriable.\n",
			repoNCConfig.NetworkID,
			err.String())
		return nil, false, err
	} else if repoNetworkConfiguration == nil {
		logger.Log("Configuration for Network %s not found. "+
			"Marking this error as non retriable.\n",
			repoNCConfig.NetworkID)

		msg := fmt.Sprintf(
			"Configuration for Network %s not found.\n",
			repoNCConfig.NetworkID)
		return nil, false, &api.JsonErr{
			Code:       api.NetworkNotFound,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
	}
	logger.Log("\nRetrieved network configuration:%+v\n", repoNetworkConfiguration)

	var associatedInterfaceID string
	if len(repoNCConfig.GetAssociationInfo().InterfaceID) == 0 {
		associatedInterfaceID = repoNodeConfig.NodeInfo.IPAddresses[0]
	} else {
		associatedInterfaceID = repoNCConfig.GetAssociationInfo().InterfaceID
	}

	ncGoalState := &NCGoalState{
		NetworkContainerID:    repoNCConfig.NetworkContainerID,
		Version:               string(repoNCConfig.Version),
		NetworkID:             repoNCConfig.NetworkID,
		AllocatedIPSubnet:     repoReservationConfiguration.AllocatedSubnet,
		BaseSubnet:            repoSubnetConfiguration.IPSubnet,
		BaseSubnetName:        repoSubnetConfiguration.SubnetName,
		AccessToken:           repoSubnetConfiguration.AccessToken,
		GatewayIP:             repoSubnetConfiguration.Gateway,
		DNSServers:            repoSubnetConfiguration.DNSServers,
		AssociatedNodeID:      nodeID,
		AssociatedInterfaceID: associatedInterfaceID,
		NodeInterfaceID:       repoNodeConfig.NodeInfo.IPAddresses[0],
		CNetSubnets:           strings.Split(repoNetworkConfiguration.CNetSpace, ","),
		VNetSubnets:           strings.Split(repoNetworkConfiguration.VNetSpace, ",")}

	return ncGoalState, false, nil
}

// Post request to host agent to express the intent for joining a network.
func postRequestToAgentForNetworkJoin(networkID string) (bool, *api.JsonErr) {
	joinURL := fmt.Sprintf(nmagentcontracts.JoinNetworkURLFmt, networkID)

	// Empty body is required as wireserver cannot handle a post without the body.
	var body bytes.Buffer
	json.NewEncoder(&body).Encode("")
	joinResponse, joinRespErr := http.Post(joinURL, "application/json", &body)
	if joinRespErr != nil {
		msg := fmt.Sprintf(
			"Join for network [%s] failed with reponse error [%s]. Marking this error as retriable",
			networkID,
			joinRespErr.Error())

		return true, &api.JsonErr{
			Code:       api.JoinDenied,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

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
	getURL := fmt.Sprintf(nmagentcontracts.GetNetworkConfigurationURLFmt, networkID)
	getResponse, getRespErr := http.Get(getURL)
	if getRespErr != nil {
		msg := fmt.Sprintf(
			"Retrieval of network configuration for network [%s] failed with error [%s]. Marking this error as retriable.",
			networkID,
			getRespErr.Error())
		return true, &api.JsonErr{
			Code:       api.NetworkConfigurationRetrievalFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil
	}

	var networkConfigurationResponse nmagentcontracts.VirtualNetworkResponse
	decodeRespErr := decodeResponse(getResponse, &networkConfigurationResponse)
	if decodeRespErr != nil {
		msg := fmt.Sprintf("Failed to decode get virtual network configuration response for network %s due to error %s. Marking this error as retriable",
			networkID,
			decodeRespErr.Error())
		return true, &api.JsonErr{
			Code:       api.DeocodeResponseFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil
	}

	logger.Log("Received response for network %s Configuration:%s\n", networkID, networkConfigurationResponse.String())
	getNetworkResponseCode, parseErr := strconv.ParseInt(networkConfigurationResponse.ResponseCode, 10, 64)
	if parseErr != nil {
		msg := fmt.Sprintf(
			"Failed to convert response code %s  to integer when parsing network configuration response for network %s, Marking this error as retriable.",
			networkConfigurationResponse.ResponseCode, networkID)
		return true, &api.JsonErr{
			Code:       api.ParseResponseCodeFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil
	}

	if getNetworkResponseCode == http.StatusProcessing {
		msg := fmt.Sprintf(
			"GetNetworkConfiguration for network %s is pending", networkID)

		return true, &api.JsonErr{
			Code:       api.ProcessingPending,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil
	}

	if getNetworkResponseCode == http.StatusUnauthorized {
		msg := fmt.Sprintf("Join for network [%s] is unauthorized", networkID)
		return false, &api.JsonErr{
			Code:       api.NetworkJoinFailed,
			Text:       msg,
			HTTPStatus: http.StatusForbidden}, nil
	}

	if getNetworkResponseCode != http.StatusOK {
		msg := fmt.Sprintf("Failed to retrieve network due to error %d", getNetworkResponseCode)
		return false, &api.JsonErr{
			Code:       api.NetworkJoinFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil
	}

	logger.Log("Retrieved virtual network configuration %s\n", networkConfigurationResponse)
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

	// Retrieve the current subnets from the repository.
	for _, subnetName := range repoNetworkConfig.Subnets {
		subnetConfiguration, subnetCfgErr := datamodel.TryQuerySubnetConfiguration(
			repoNetworkConfig.NetworkID,
			subnetName)
		if subnetCfgErr != nil {
			logger.Log(
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
				logger.Log("Marking subnet %s in configuration for add as there are no allocations\n", subnet.SubnetName)
				subnetsToAdd[subnet.SubnetName] = subnet
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
				msg := fmt.Sprintf("Failed to parse IPSubnet %s from repo config %s",
					subnet.AddressPrefix,
					repoSubnetConfig.String())
				return false, &api.JsonErr{
					Code:       api.NetworkJoinFailed,
					Text:       msg,
					HTTPStatus: http.StatusInternalServerError}, nil, nil, nil
			}

			if utils.AreIPSubnetsEqual(currentSubnet, updatedSubnet) {
				msg := fmt.Sprintf(
					"Update of subnet %s is not allowed as there are allocations. Current subnet:%s, Updated subnet:%s",
					repoSubnetConfig.SubnetName,
					currentSubnet.String(),
					updatedSubnet.String())
				return false, &api.JsonErr{
					Code:       api.NetworkJoinFailed,
					Text:       msg,
					HTTPStatus: http.StatusInternalServerError}, nil, nil, nil
			}
			subnetsToUpdate[subnet.SubnetName] = repoSubnetConfig
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
	var subnetsInUpdatedResponse map[string]*nmagentcontracts.Subnet
	var ipSubnetMap map[string]string
	var repoNetworkConfiguration *datamodel.NetworkConfiguration
	var netCfgErr *api.JsonErr
	var vnetVersion uint64

	// Network configuration is expected to be present as we will always
	// add a dummy configuration and mark it as pendiong processing once
	// a request is received.
	networkConfiguration, err := datamodel.TryQueryNetworkConfiguration(networkID)
	if err != nil {
		logger.Log("Network configuration retrieval for network %s failed with error %s\n",
			networkID,
			err.String())
		return false, err
	}

	if networkConfiguration == nil {
		logger.Log("Failed to find configuration in repository for network %s\n", networkID)
		return true, nil
	}

	isRetriableError, failureDetail = postRequestToAgentForNetworkJoin(networkID)
	if failureDetail != nil {
		if isRetriableError {
			return false, err
		}
		goto Cleanup
	}

	isRetriableError, failureDetail, networkResponse = getNetworkConfigurationFromAgent(networkID)
	if failureDetail != nil {
		if isRetriableError {
			return false, err
		}
		goto Cleanup
	}

	repoNetworkConfiguration, netCfgErr = datamodel.TryQueryNetworkConfiguration(networkID)
	if netCfgErr != nil {
		logger.Log("Failed to retrieve subnet configuration for network %s due to error %s\n", networkID, netCfgErr.String())
		return false, nil
	}

	if repoNetworkConfiguration == nil {
		subnetsToAdd = make(map[string]nmagentcontracts.Subnet)
		subnetsToRemove = make(map[string]bool)
		subnetsToUpdate = make(map[string]*datamodel.SubnetConfiguration)
		for _, subnet := range networkResponse.Subnets {
			subnetsToAdd[subnet.SubnetName] = subnet
		}
	} else {
		isRetriableError, failureDetail, subnetsToAdd, subnetsToRemove, subnetsToUpdate =
			validateAndRetrieveNetworkConfigurationUpdate(
				repoNetworkConfiguration,
				networkResponse)

		if failureDetail != nil {
			if isRetriableError {
				return false, err
			}
			goto Cleanup
		}
	}

	for subnetName := range subnetsToRemove {
		failureDetail = datamodel.DeleteSubnetConfiguration(networkID, subnetName)
		if failureDetail != nil {
			goto Cleanup
		}
	}

	ipSubnetMap = make(map[string]string)
	subnetsInUpdatedResponse = make(map[string]*nmagentcontracts.Subnet)
	for _, subnet := range networkResponse.Subnets {
		subnetsInUpdatedResponse[subnet.SubnetName] = &subnet
		ipSubnetMap[subnet.SubnetName] =
			datamodel.GetSubnetKey(networkID, subnet.SubnetName)
	}

	// Commiting virtual network config in two phases.
	// First phase, update only the configuration and the subnet
	// list and the state is still pending join.
	// This helps in driving cleanup as we have handle to all the subnets
	// and we can handle any faulures that happens when persisting subnet
	// configurations.

	logger.Log("Adding vnetconfig:%s", networkResponse.VnetVersion)
	vnetVersion, _ = strconv.ParseUint(networkResponse.VnetVersion, 10, 64)
	repoNetworkConfiguration = &datamodel.NetworkConfiguration{
		NetworkID: networkID,
		Version:   vnetVersion,
		CNetSpace: networkResponse.CNetSpace,
		VNetSpace: networkResponse.VNetSpace,
		Subnets:   ipSubnetMap}

	failureDetail = datamodel.PersistNetworkConfiguration(repoNetworkConfiguration)
	if failureDetail != nil {
		goto Cleanup
	}

	for subnetName := range subnetsToAdd {
		updatedSubnetCfg := subnetsInUpdatedResponse[subnetName]
		var dnsServers []string
		if len(networkResponse.DNSServers) != 0 {
			dnsServers = strings.Split(networkResponse.DNSServers, ",")
		} else {
			ipSubnet, err := utils.ParseIPNet(updatedSubnetCfg.AddressPrefix)
			if err != nil {
				failureDetail = &api.JsonErr{
					Code:       api.InvalidIPSubnet,
					HTTPStatus: http.StatusInternalServerError,
					Text: fmt.Sprintf("Failed to pasre subnet %s due to error %s",
						updatedSubnetCfg.AddressPrefix, err.Error())}
			}
			dnsServers = allocationhelper.GetDNSServerAddressesForSubnet(ipSubnet)
		}

		subnetConfig := datamodel.SubnetConfiguration{
			NetworkID:         networkID,
			SubnetName:        subnetName,
			IPSubnet:          updatedSubnetCfg.AddressPrefix,
			IsDelegatedSubnet: true,
			DNSServers:        dnsServers,
			Gateway:           updatedSubnetCfg.DefaultGateway}
		failureDetail = datamodel.PutSubnetConfiguration(&subnetConfig)
		if failureDetail != nil {
			goto Cleanup
		}
	}

	for subnetName, currentRepoSubnetCfg := range subnetsToUpdate {
		ipSubnetMap[subnetName] = datamodel.GetSubnetKey(networkID, subnetName)
		updatedSubnetCfg := subnetsInUpdatedResponse[subnetName]
		currentRepoSubnetCfg.DNSServers = strings.Split(networkResponse.DNSServers, ",")
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
	if failureDetail != nil {
		goto Cleanup
	}

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

	logger.Log("Successfully added vnetconfig: %s\n", repoNetworkConfiguration.String())
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
		false)
	if err != nil {
		logger.Log("Failed to add network container to node configuration %s\n", err.String())
		return nil, err
	}

	reservationKey := datamodel.GetReservationKey(reservationConfiguration.NetworkID, reservationConfiguration.ReservationID)
	requestContext := datamodel.RequestContext{
		Type:      datamodel.ReservationRequest,
		NetworkID: repoNCConfig.NetworkID,
		NodeID:    repoNCConfig.GetAssociationInfo().NodeID}

	err = datamodel.AddRequestToPendingQueue(reservationKey, &requestContext)
	if err != nil {
		logger.Log("Failed to add the request to the the pending process queue\n")
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
		logger.Log(
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
		logger.Log(
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
		logger.Log("Failure Detail %s\n", failureDetail.String())
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
		logger.Log(
			"Creating Reservation NetworkID: %s ReservationID: %s\n",
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
		logger.Log("NC in failed state during prepare. Failure detail %s\n", repoNCConfig.FailureDetail)
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
	err := datamodel.PurgeNetworkContainerAssociations(repoNCConfig)
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
	err := datamodel.PurgeNetworkContainerAssociations(repoNCConfig)
	if err != nil {
		return false, err
	}

	err = datamodel.DeleteNetworkContainer(repoNCConfig.NetworkID, repoNCConfig.NetworkContainerID)
	if err != nil {
		return false, err
	}

	return true, nil
}

func processNetworkContainerAssociate(repoNCConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	ncGoalState, isRetriableError, ncGoalStateErr := getNCGoalState(repoNCConfig)
	if ncGoalStateErr != nil {
		if isRetriableError {
			return false, ncGoalStateErr
		} else {
			repoNCConfig.Status = datamodel.Failed
			repoNCConfig.FailureDetail = *ncGoalStateErr
			return true, datamodel.PersistNCConfiguration(repoNCConfig, false)
		}
	}

	var body bytes.Buffer
	var ipConfig cns.IPConfiguration
	var ipSubnet cns.IPSubnet
	var resp cns.Response
	var respErr error
	vnetAddresses := make([]cns.IPSubnet, len(ncGoalState.VNetSubnets))

	ipConfig.DNSServers = ncGoalState.DNSServers
	ipConfig.GatewayIPAddress = ncGoalState.GatewayIP
	allocatedIP := strings.Split(ncGoalState.AllocatedIPSubnet, "/")[0]
	_, baseSubnetPrefix, err := utils.GetIPAddressStringAndPrefix(ncGoalState.BaseSubnet)
	if err != nil {
		msg := fmt.Sprintf("Failed to parse base subnet string %s due to error %s",
			ncGoalState.BaseSubnet,
			err.Error())
		logger.Log("%s, Marking as non retriable error\n", msg)
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
	for index, vnetAddressString := range ncGoalState.VNetSubnets {
		logger.Log("Parsing vnetAddress: %s\n", vnetAddressString)
		ipAddressString, prefix, parserErr := utils.GetIPAddressStringAndPrefix(vnetAddressString)
		if parserErr != nil {
			msg := fmt.Sprintf("Failed to parse Vnet subnet string %s due to error %s",
				vnetAddressString,
				err.Error())
			logger.Log("%s, Marking as non retriable error\n", msg)
			failureDetail := api.JsonErr{
				Code:       api.InvalidIPSubnet,
				Text:       msg,
				HTTPStatus: http.StatusBadRequest}
			repoNCConfig.Status = datamodel.Failed
			repoNCConfig.FailureDetail = failureDetail
			return true, datamodel.PersistNCConfiguration(repoNCConfig, false)
		}

		cnsIPSubnet := cns.IPSubnet{IPAddress: ipAddressString, PrefixLength: uint8(prefix)}
		vnetAddresses[index] = cnsIPSubnet
	}

	info := &cns.CreateNetworkContainerRequest{
		Version:                    ncGoalState.Version,
		NetworkContainerType:       "WebApps",
		NetworkContainerid:         "Swift_" + ncGoalState.NetworkContainerID,
		PrimaryInterfaceIdentifier: ncGoalState.AssociatedInterfaceID,
		IPConfiguration:            ipConfig,
		AuthorizationToken:         ncGoalState.AccessToken,
		VnetAddressSpace:           vnetAddresses}

	json.NewEncoder(&body).Encode(info)
	cerateOrUpdateNCURL := "http://" + ncGoalState.NodeInterfaceID + ":10090/network/createorupdatenetworkcontainer"
	w, respErr := http.Post(cerateOrUpdateNCURL, "application/json", &body)

	if respErr != nil {
		msg := fmt.Sprintf("Post to URL %s for NC %s failed with error %s",
			cerateOrUpdateNCURL,
			ncGoalState.NetworkContainerID,
			err.Error())
		logger.Log("%s, Marking as retriable error\n", msg)
		return false, &api.JsonErr{
			Code:       api.AssociationFailed,
			Text:       respErr.Error(),
			HTTPStatus: http.StatusBadRequest}
	}

	respErr = decodeResponse(w, &resp)
	if resp.ReturnCode != 0 {
		msg := fmt.Sprintf("Associate NC %s to node %s failed with error code %d",
			ncGoalState.NetworkContainerID,
			ncGoalState.NodeInterfaceID,
			resp.ReturnCode)
		logger.Log("%s, Marking as non retriable error\n", msg)
		failureDetail := api.JsonErr{
			Code:       api.AssociationFailed,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
		repoNCConfig.Status = datamodel.Failed
		repoNCConfig.FailureDetail = failureDetail
		return true, datamodel.PersistNCConfiguration(repoNCConfig, false)
	}
	if respErr != nil {
		msg := fmt.Sprintf("Associate NC %s to node %s failed with error %s",
			ncGoalState.NetworkContainerID,
			ncGoalState.NodeInterfaceID,
			respErr.Error())
		logger.Log("%s, Marking as retriable error\n", msg)

		return false, &api.JsonErr{
			Code:       api.AssociationFailed,
			Text:       respErr.Error(),
			HTTPStatus: http.StatusBadRequest}
	}

	repoNCConfig.State = datamodel.Publish
	repoNCConfig.Status = datamodel.Pending
	return false, datamodel.PersistNCConfiguration(repoNCConfig, false)
}

func decodeResponse(w *http.Response, response interface{}) error {
	if w.StatusCode != http.StatusOK {
		return fmt.Errorf("Request failed with HTTP error %d", w.StatusCode)
	}

	if w.Body == nil {
		return fmt.Errorf("Response body is empty")
	}
	return json.NewDecoder(w.Body).Decode(&response)
}

func processNetworkContainerDisAssociate(repoNCConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	nodeID := repoNCConfig.GetAssociationInfo().NodeID

	// Query NodeConfiguration table with nodeID and handle errors.
	repoNodeConfig, err := datamodel.TryQueryNodeConfiguration(nodeID)

	if err != nil {
		logger.Log("Retrieval of node configuration for node %s failed with erro %s. Marking error as retriable.\n",
			nodeID,
			err.String())
		return false, err
	} else if repoNodeConfig == nil {
		msg := fmt.Sprintf("Node configuration for node %s not found.",
			nodeID)

		logger.Log("%s, Marking as non retriable error\n", msg)
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

	var body bytes.Buffer
	json.NewEncoder(&body).Encode(info)
	w, respErr := http.Post("http://"+primaryInterfaceID+":10090/network/deletenetworkcontainer", "application/json", &body)

	if respErr != nil {
		logger.Log("Failed to disassociate network container %s on node %s due to error. Marked as retriable.\n",
			info.NetworkContainerid,
			primaryInterfaceID)
		return false, &api.JsonErr{
			Code:       api.DisAssociationFailed,
			Text:       respErr.Error(),
			HTTPStatus: http.StatusInternalServerError}
	}

	var resp cns.DeleteNetworkContainerResponse
	decodeErr := decodeResponse(w, &resp)
	if decodeErr != nil {
		logger.Log("Failed to decode disassociate response for request with network container Id:%s. Marking as retriable error\n",
			repoNCConfig.NetworkContainerID)
		return false, &api.JsonErr{
			Code:       api.DisAssociationFailed,
			Text:       decodeErr.Error(),
			HTTPStatus: http.StatusBadRequest}
	}

	if resp.Response.ReturnCode != 0 {
		msg := fmt.Sprintf("Disassociate request associated with network container Id:%s returned status %d. "+
			"Message:%+v. Marking as non retriable error",
			repoNCConfig.NetworkContainerID,
			resp.Response.ReturnCode,
			resp.Response.Message)
		repoNCConfig.Status = datamodel.Failed
		repoNCConfig.FailureDetail = api.JsonErr{
			Code:       api.DisAssociationFailed,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}

		return true, datamodel.PersistNCConfiguration(repoNCConfig, false)
	}

	logger.Log("DisAssociated NC %s from node %s\n",
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
		} else {
			repoNCConfig.Status = datamodel.Failed
			repoNCConfig.FailureDetail = *ncGoalStateErr
			return true, datamodel.PersistNCConfiguration(repoNCConfig, false)
		}
	}
	var body bytes.Buffer
	var ncRequest nmagentcontracts.CreateOrUpdateNetworkContainerRequest
	ncRequest.IPV4Addresses = make([]string, 1)
	ncRequest.IPV4Addresses[0] = strings.Split(ncGoalState.AllocatedIPSubnet, "/")[0]
	ncRequest.Policies = make([]string, 0)
	ncRequest.SubnetName = ncGoalState.BaseSubnetName
	ncRequest.Version = ncGoalState.Version
	ncRequest.VirtualNetworkID = ncGoalState.NetworkID
	json.NewEncoder(&body).Encode(ncRequest)

	createNCURL := fmt.Sprintf(nmagentcontracts.PutNetworkContainerURLFmt,
		ncGoalState.NodeInterfaceID,
		ncGoalState.NetworkContainerID,
		ncGoalState.AccessToken)
	logger.Log("Url:%s, body:%s\n", createNCURL, ncRequest.String)
	_, respErr := http.Post(createNCURL, "application/json", &body)

	if respErr != nil {
		logger.Log("Failed to publish network container goal state for Id %s to host agent with error %s. Marking as retriable error.\n",
			repoNCConfig.NetworkContainerID,
			respErr.Error())
		return false, &api.JsonErr{
			Code:       api.PublishingFailed,
			Text:       respErr.Error(),
			HTTPStatus: http.StatusInternalServerError}
	}

	repoNCConfig.State = datamodel.Program
	repoNCConfig.Status = datamodel.Pending
	return false, datamodel.PersistNCConfiguration(repoNCConfig, false)
}

func processNetworkContainerUnPublish(repoNCConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	ncGoalState, isRetriableError, ncGoalStateErr := getNCGoalState(repoNCConfig)
	if ncGoalStateErr != nil {
		if isRetriableError {
			return false, ncGoalStateErr
		} else {
			repoNCConfig.Status = datamodel.Failed
			repoNCConfig.FailureDetail = *ncGoalStateErr
			return true, datamodel.PersistNCConfiguration(repoNCConfig, false)
		}
	}

	logger.Log("Creating delete URL\n")

	// Empty body is required as wireserver cannot handle a post without the body.
	var body bytes.Buffer
	json.NewEncoder(&body).Encode("")
	deleteNCURL := fmt.Sprintf(nmagentcontracts.DeleteNetworkContainerURLFmt,
		ncGoalState.NodeInterfaceID,
		ncGoalState.NetworkContainerID,
		ncGoalState.AccessToken)
	logger.Log("******Unpublishing: Url:%s\n", deleteNCURL)
	resp, respErr := http.Post(deleteNCURL, "application/json", &body)

	if resp != nil {
		logger.Log("Received response from host for DeleteNetworkContainer: %+v", resp)
	} else {
		logger.Log("Received nil response from host for DeleteNetworkContainer.")
	}

	if respErr != nil {
		if respErr != nil {
			logger.Log("Failed to unpublish network container goal state for Id %s to host agent with error %s. Marking as retriable error.\n",
				repoNCConfig.NetworkContainerID,
				respErr.Error())
			return false, &api.JsonErr{
				Code:       api.PublishingFailed,
				Text:       respErr.Error(),
				HTTPStatus: http.StatusInternalServerError}
		}
	}

	var deleteNCResponse nmagentcontracts.DeleteNetworkContainerResponse
	decodeRespErr := decodeResponse(resp, &deleteNCResponse)
	if decodeRespErr != nil {
		msg := fmt.Sprintf("Failed to decode get delete NC response for NC %s due to error %s. Marking this error as retriable",
			repoNCConfig.NetworkContainerID,
			decodeRespErr.Error())
		return false, &api.JsonErr{
			Code:       api.DeocodeResponseFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	logger.Log("Received response for network response %s, Message:%v for NC:%s\n",
		deleteNCResponse.ResponseCode,
		deleteNCResponse.ErrorMessage,
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

	logger.Log("Retrieved NC response: %d\n", deleteNCResponseCode)
	if deleteNCResponseCode != http.StatusOK {
		msg := fmt.Sprintf("Failed to unpublish NC with status code %d. Marking as non retriable.", deleteNCResponseCode)
		repoNCConfig.Status = datamodel.Failed
		repoNCConfig.FailureDetail = api.JsonErr{
			Code:       api.UnpublishingFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
		return true, datamodel.PersistNCConfiguration(repoNCConfig, false)
	}

	repoNCConfig.State = datamodel.DisAssociate
	repoNCConfig.Status = datamodel.Pending
	return false, datamodel.PersistNCConfiguration(repoNCConfig, false)
}

func processNetworkContainerRevert(repoNCConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	associationInfo := repoNCConfig.GetAssociationInfo()
	logger.Log("Reverting NC %s to node %s\n", repoNCConfig.NetworkContainerID, associationInfo.NodeID)
	repoNCConfig.State = datamodel.UnPublish
	repoNCConfig.Status = datamodel.Pending
	return true, datamodel.PersistNCConfiguration(repoNCConfig, false)
}

func processNetworkContainerProgram(repoNCConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	associationInfo := repoNCConfig.GetAssociationInfo()
	logger.Log("Publishing NC %s to node %s\n", repoNCConfig.NetworkContainerID, associationInfo.NodeID)
	repoNCConfig.State = datamodel.Program
	repoNCConfig.Status = datamodel.Completed
	return true, datamodel.PersistNCConfiguration(repoNCConfig, false)
}

func processNetworkContainerRequest(networkID, networkContainerkey string) (bool, *api.JsonErr) {
	repoNCConfig, err := datamodel.TryQueryNetworkContainerByKey(networkContainerkey)
	if err != nil {
		logger.Log("Failed to get network container by key. Error:%s\n", err.String())
		return false, err
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
		if repoNCConfig.Status == datamodel.Failed {
			return true, nil
		}

		return processNetworkContainerAssociate(repoNCConfig)

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
		return processNetworkContainerDisAssociate(repoNCConfig)

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
			processPendingRequests()
			logger.Log("Received signal to process pending work\n")
			break
		case <-cancelProcessing:
			logger.Log("Received signal to exit\n")
			break
		}
	}
}
