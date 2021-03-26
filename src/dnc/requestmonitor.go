package main

import (
	"bytes"
	"dnc/aiwrapper"
	"dnc/allocationhelper"
	"dnc/api"
	"dnc/configuration"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/networkcontainer"
	"dnc/nmagentcontracts"
	"dnc/pubsubproxy"
	"dnc/resourceupdatehandler/networkupdatehandler"
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
	featureMigrationComplete bool
	containerTypes           sync.Map
)

const (
	// NMAgent using 169.254.0.0 to 169.254.3.255. So allocating a range not conflicting with that.
	localIPSubnetAddressString            = "169.254.128.0/17"
	imdsIPSubnetString                    = "169.254.169.254/32"
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
		_, success := networkupdatehandler.RefreshNetworkConfiguration(datamodel.GetNetworkKey(requestContext.NetworkID), true)
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

	return networkupdatehandler.ProcessVirtualNetworkJoin(networkID, repoNetworkConfiguration)
}

func getAllocatedLocalAddressesOnNode(nodeConfiguration datamodel.NodeConfiguration) (map[string]*net.IPNet, *api.JsonErr) {
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
		greErr,
		allocationErr error
		vlanID uint16
	)

	nodeID := repoNCConfig.GetAssociationInfo().NodeID

	sessionInfo := map[string]string{datamodel.NodeRepositoryName: nodeID}
	sessionManager.BeginWriteSession(sessionInfo)
	defer sessionManager.EndWriteSession(sessionInfo)

	nodeConfiguration, err := datamodel.NodeConfigRepo.QueryNodeConfiguration(nodeID)
	if err != nil {
		return false, err
	}

	// get all allocated local addresses for a node
	allocatedLocalIPAddresses, err := getAllocatedLocalAddressesOnNode(*nodeConfiguration)
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
	if nodeConfiguration.InterfaceGreKeyMap == nil {
		nodeConfiguration.InterfaceGreKeyMap = make(map[string]map[string]uint16)
	}

	if nodeConfiguration.NetworkVlanIDMap != nil {
		vlanID = nodeConfiguration.NetworkVlanIDMap[repoNCConfig.NetworkID]
	}

	greErr = datamodel.CheckGreConditions(
		nodeConfiguration.NodeID,
		repoNCConfig.NetworkContainerID,
		nodeConfiguration.UpdateGreCapable,
		repoNCConfig.GreKey,
		vlanID)

	//Allocate GRE Key for this NC
	if greErr == nil {
		interfaceID := ""
		if !repoNCConfig.ImplicitReservation {
			interfaceID = repoNCConfig.RequestV1.AssociationInfo.InterfaceID
		} else {
			interfaceID = repoNCConfig.RequestV2.AssociationInfo.InterfaceID
		}

		if interfaceID != "" {

			if nodeConfiguration.InterfaceGreKeyMap[interfaceID] == nil {
				nodeConfiguration.InterfaceGreKeyMap[interfaceID] = make(map[string]uint16)
			}

			greKey, nextGreErr := datamodel.GetNextGreKey(nodeConfiguration.InterfaceGreKeyMap[interfaceID])
			if nextGreErr != nil {
				markProcessingFailed = true
				failureDetail = &api.JsonErr{
					Code:       nextGreErr.Code,
					HTTPStatus: http.StatusInternalServerError,
					Text: fmt.Sprintf(
						"%s NetworkContainer: %s, Node: %s", nextGreErr.Text, repoNCConfig.NetworkContainerID, nodeID)}
				goto Cleanup

			}
			repoNCConfig.GreKey = greKey
			nodeConfiguration.InterfaceGreKeyMap[interfaceID][repoNCConfig.ReservationID] = greKey
			logger.Debugf("Assigning GreKey for NC: %s, Node: %s, interface ID is %s, reservation ID is %s, GreKey is %d",
				repoNCConfig.NetworkContainerID, nodeID, interfaceID, repoNCConfig.ReservationID, greKey)

			nodeErr := datamodel.NodeConfigRepo.PersistNodeConfiguration(nodeConfiguration)
			if nodeErr != nil {
				return false, nodeErr
			}
		} else {
			logger.Debugf("Could not find Interface AssiciationInfo while assinging GreKey for NC: %s, Node: %s",
				repoNCConfig.NetworkContainerID, nodeID)
		}

	} else {
		logger.Debugf(greErr.Error())
	}

	if config.DNCSettings.Managed {
		repoNCConfig.State = datamodel.Publish
	} else {
		repoNCConfig.State = datamodel.Associate
	}

	repoNCConfig.LocalIP = allocatedIP.IP.String()
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

// This will create the pending IP cleanup job for ToBeDeleted Secondary IPs during NC update.
func processPrepareSecondaryIpCleanup(
	repoNCConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {

	if config.PubSubProxySettings.EnablePubSubProxy {
		ncKey := datamodel.GetNCKey(repoNCConfig.NetworkID, repoNCConfig.NetworkContainerID)
		loggershim.LS.Debugf("processPrepareSecondaryIpCleanup, NCKey %s", ncKey)

		// get reservation
		var subnetName = ""
		if repoNCConfig.ImplicitReservation {
			subnetName = repoNCConfig.RequestV2.AllocationRequest.SubnetName
		} else {
			loggershim.LS.Debugf("processPrepareSecondaryIpCleanup: Getting Subnet name from Reservation record, nckey %s, reservationId %s",
				ncKey,
				repoNCConfig.ReservationID)
			repoReservation, err := datamodel.TryQueryIPReservation(repoNCConfig.NetworkID, repoNCConfig.ReservationID)
			if err != nil {
				loggershim.LS.Errorf("processPrepareSecondaryIpCleanup: Failed to query associated Reservation repo, NcKey %s, reservationId %s, err %+v",
					ncKey,
					repoNCConfig.ReservationID,
					err)
				return false, err
			}

			if repoReservation.RowKey == "" {
				loggershim.LS.Errorf("processPrepareSecondaryIpCleanup: associated Reservation config not found, NcKey %s, reservationId %s, err %+v",
					ncKey,
					repoNCConfig.ReservationID,
					err)
				return false, err
			}

			subnetName = repoReservation.Configuration.AllocationRequest.SubnetName
		}

		if subnetName == "" {
			return true, api.InternalServerErrorf(api.InvalidSubnetName,
				"ProcessPrepareSecondaryIpCleanup: associated subnetName is empty, NcKey %s, reservationId %s",
				ncKey, repoNCConfig.ReservationID)
		}

		// We have already updated IpAllocations table with ToBeDeletedOwner for ToBeDeletedIpIds
		// in Reserve state, thus it is safe to request all pending deleting IPs and
		// create or Merge the PendingResourceCleanupJob
		err := ipAllocator.CreateIpCleanupRequest(repoNCConfig.NetworkID, subnetName, repoNCConfig.NetworkContainerID)
		if err != nil {
			loggershim.LS.Errorf("processPrepareSecondaryIpCleanup: Failed to create the cleanup request err:  %+v",
				err)
			return false, err
		}
	}

	repoNCConfig.State = datamodel.Program
	repoNCConfig.Status = datamodel.Pending
	repoNCConfig.FailureDetail = api.JsonErr{}
	return false, datamodel.PersistNCConfiguration(repoNCConfig, datamodel.DontAddToPendingQ)
}

func processNetworkContainerReserve(
	repoNCConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	var (
		err                  *api.JsonErr
		failureDetail        *api.JsonErr
		markProcessingFailed bool
		repoReservation      datamodel.ReservationRow
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

	repoReservation, err = datamodel.TryQueryIPReservation(repoNCConfig.NetworkID, repoNCConfig.ReservationID)
	if err != nil {
		return false, err
	}

	if repoReservation.Configuration == nil {
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

	if len(repoNCConfig.RequestV2.ToBeDeletedIPIds) > 0 {
		loggershim.LS.Debugf("processNetworkContainerReserve: Process deleting secondary Ips NetworkId %s, ReservationId %s",
			repoNCConfig.NetworkID, repoNCConfig.ReservationID)

		// In case pubsub poxy is enabled, wait for publish job to finish before deleting the ips during scale down
		// Thus set the ToBeDeletedOwner for the deleting ips
		if config.PubSubProxySettings.EnablePubSubProxy {
			// It is okay if the associated ips are empty, it will still get cleaned up by the background cleanup monitor
			var ipsPendingCleanup = make(map[string]string)
			for _, allocID := range repoNCConfig.RequestV2.ToBeDeletedIPIds {
				if ipAddress, found := repoReservation.Configuration.SecondaryAllocations[allocID]; found {
					ipsPendingCleanup[allocID] = ipAddress
				}
			}

			reservationKey := datamodel.GetReservationKey(repoNCConfig.NetworkID, repoNCConfig.ReservationID)
			ncKey := datamodel.GetNCKey(repoNCConfig.NetworkID, repoNCConfig.NetworkContainerID)
			err = ipAllocator.SetToBeDeleteOwnerToDeletingIps(
				repoReservation.Configuration.NetworkID,
				repoReservation.Configuration.AllocationRequest.SubnetName,
				reservationKey,
				ncKey,
				ipsPendingCleanup)
			if err != nil {
				loggershim.LS.Errorf("processNetworkContainerReserve: Failed to set ToBeDeletedOwner for associated ips, reservationKey %s, deleting ips %+v, err %+v",
					reservationKey,
					ipsPendingCleanup,
					err)
				return false, err
			}
		} else {
			err := ipAllocator.DeallocateIPs(*repoReservation.Configuration, repoNCConfig.RequestV2.ToBeDeletedIPIds)
			if err != nil {
				return false, err
			}
		}

		// Now deleting
		for _, allocID := range repoNCConfig.RequestV2.ToBeDeletedIPIds {
			delete(repoReservation.Configuration.SecondaryAllocations, allocID)
		}
		if err := datamodel.PutIPReservationConfiguration(repoReservation.Configuration); err != nil {
			return false, err
		}
	}

	// If the reservation status is pending, process IP allocation
	if repoReservation.Configuration.Status == datamodel.Pending {
		ipReservationAllocationComplete, err := ipAllocator.ProcessIPReservationAllocation(repoReservation.Configuration)

		// Return if the IP reservation allocation did not complete. This indicates that
		// the operation ran into retriable error.
		if !ipReservationAllocationComplete {
			return false, err
		}
	}

	if repoReservation.Configuration.Status == datamodel.Completed {
		repoNCConfig.State = datamodel.AllocateLocalIP
		repoNCConfig.Status = datamodel.Pending
		repoNCConfig.FailureDetail = api.JsonErr{}

		repoSubnetConfiguration, err := datamodel.TryQuerySubnetConfiguration(
			repoNCConfig.NetworkID,
			repoReservation.Configuration.AllocationRequest.SubnetName)
		if err != nil {
			return true, err
		}
		if repoSubnetConfiguration == nil {
			msg := fmt.Sprintf(
				"Configuration for Subnet %s in Network %v not found.",
				repoReservation.Configuration.AllocationRequest.SubnetName,
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

	if repoReservation.Configuration.Status == datamodel.Failed {
		markProcessingFailed = true
		failureDetail = &repoReservation.Configuration.FailureDetail
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

func markNCAsFailed(ncConfig *datamodel.NCConfiguration, err api.JsonErr) (bool, *api.JsonErr) {
	ncConfig.Status = datamodel.Failed
	ncConfig.FailureDetail = err
	if err := datamodel.PersistNCConfiguration(ncConfig, datamodel.DontAddToPendingQ); err != nil {
		return false, err // don't remove from queue until failure can be persisted
	}
	return true, nil
}

func processNetworkContainerBatchReserve(nc *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	sessionInfo := map[string]string{datamodel.ReservationRepositoryName: nc.ReservationID}

	sessionManager.BeginWriteSession(sessionInfo)
	defer sessionManager.EndWriteSession(sessionInfo)

	reservation, apiErr := datamodel.TryQueryIPReservation(nc.NetworkID, nc.ReservationID)
	if apiErr != nil {
		return false, apiErr
	}

	if reservation.Configuration == nil {
		return markNCAsFailed(nc, *api.NotFoundf(api.ReservationNotFound, "could not find reservation %s for nc %s", nc.NetworkContainerID, nc.ReservationID))
	}

	res, err := resDispatcher.Reserve(*reservation.Configuration, nc.RequestV2.ToBeDeletedIPIds)
	if err != nil {
		return false, api.InternalServerError(api.StatusInternalServerError, err.Error())
	}

	switch res.Status {
	case datamodel.Failed:
		return markNCAsFailed(nc, res.FailureDetail)
	case datamodel.Completed:
		return false, transitionNCTo(nc, datamodel.AllocateLocalIP, datamodel.Pending)
	default:
		return false, api.InternalServerError(api.StatusInternalServerError, "unexpected reservation status")
	}
}

func processNetworkContainerUnReserve(ncConfig *datamodel.NCConfiguration) *api.JsonErr {
	if !ncConfig.ImplicitReservation {
		sessionInfo := map[string]string{
			datamodel.ReservationRepositoryName: ncConfig.ReservationID,
		}
		sessionManager.BeginWriteSession(sessionInfo)
		defer sessionManager.EndWriteSession(sessionInfo)

		if err := datamodel.UpdateNCToReservationAssociation(ncConfig, true); err != nil {
			return err
		}

		return transitionNCTo(ncConfig, datamodel.Purge, datamodel.Pending)
	}

	sessionInfo := map[string]string{
		datamodel.ReservationRepositoryName: ncConfig.ReservationID,
	}

	// Acquire the lock only if DisableReservationMapInSubnetConfig is set to false.
	// Lock is required to update the SubnetConfiguration table which is required only if the flag is set to false
	if !config.DNCSettings.DisableReservationMapInSubnetConfig {
		sessionInfo[datamodel.NetworkRepositoryName] = ncConfig.NetworkID
	}

	// Acquire Node lock only if needed
	reservation, err := datamodel.TryQueryIPReservation(ncConfig.NetworkID, ncConfig.ReservationID)
	if err != nil {
		return err
	}
	if reservation.Configuration != nil && len(reservation.Configuration.AllocationRequest.NodeConstraint) != 0 {
		sessionInfo[datamodel.NodeRepositoryName] = ncConfig.GetAssociationInfo().NodeID
	}

	sessionManager.BeginWriteSession(sessionInfo)
	defer sessionManager.EndWriteSession(sessionInfo)

	if err := ipAllocator.DeleteIPReservation(reservation, true); err != nil {
		return err
	}

	return transitionNCTo(ncConfig, datamodel.Purge, datamodel.Pending)
}

func transitionNCTo(nc *datamodel.NCConfiguration, state datamodel.NetworkContainerState, status datamodel.Status) *api.JsonErr {
	nc.State = state
	nc.Status = status
	nc.FailureDetail = api.JsonErr{}

	return datamodel.PersistNCConfiguration(nc, datamodel.DontAddToPendingQ)
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

	err = datamodel.PurgeNetworkContainerAssociations(repoNCConfig, networkcontainer.ShouldUpdateVlan(repoNCConfig.GetAssociationInfo().ContainerType))
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

	containerTypes.Store(ncGoalState.ContainerType, true)

	var (
		resp cns.CreateNetworkContainerResponse
		body bytes.Buffer
	)

	info, removeRequestFromPendingQueue, err := datamodel.GetNetworkContainerRequest(ncGoalState, repoNCConfig)
	if info == nil {
		return removeRequestFromPendingQueue, err
	}

	json.NewEncoder(&body).Encode(info)
	createOrUpdateNCURL := fmt.Sprintf(
		config.CNSSettings.ConnectionStrings.CreateOrUpdateNetworkContainerURLFmt,
		ncGoalState.NodeInterfaceID)

	// Release the session locks when calling in to cns
	sessionManager.EndWriteSession(sessionInfo)

	// todo: cns interface method
	logger.Debugf("[CNS] create/update NC %s", ncGoalState.NetworkContainerID)
	w, respErr := cnsHttpClient.Post(createOrUpdateNCURL, defaultContentType, &body)

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
		// If the failure is due to Unsupported Orchestrator type, it indicates that the CNS state file is lost
		// and doesn't have the orchestrator type. Try sending the orchestrator type to CNS.
		if resp.Response.ReturnCode == restserver.UnsupportedOrchestratorType {
			logger.Debugf("Calling SetOrchestratorType for node: %s as CNS returned UnsupportedOrchestratorType error",
				repoNCConfig.GetAssociationInfo().NodeID)
			err := cnsInterface.SetOrchestratorType(repoNCConfig.GetAssociationInfo().NodeID,
				ncGoalState.NodeInterfaceID, ncGoalState.OrchestratorType)
			// If the operation succeeds or fails due to internal error, mark the request as retriable. This will
			// result in associate / refresh associate to be retried. If SetOrchestratorType succeeded, associate
			// task will succeed in the subsequent attempt. If SetOrchestratorType failed with retriable error,
			// If the operation fails due to error code returned by CNS, continue with marking the error as non-retriable
			if err == nil || err.HTTPStatus == http.StatusInternalServerError {
				if err == nil {
					logger.Debugf("Successfully set the orchestrator type: %s for Node: %s",
						ncGoalState.OrchestratorType, repoNCConfig.GetAssociationInfo().NodeID)
				} else {
					logger.Errorf("Failed to set the orchestrator type: %s for Node: %s due to error: %v",
						ncGoalState.OrchestratorType, repoNCConfig.GetAssociationInfo().NodeID, err)
				}
				return false, api.InternalServerErrorf(api.AssociationFailed,
					"Associate NC %v to node %v failed due to invalid orchestrator type set",
					ncGoalState.NetworkContainerID, repoNCConfig.GetAssociationInfo().NodeID)
			}
		}

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

	repoNCConfig.ProgrammedVnetVersion = ncGoalState.VnetVersion
	repoNCConfig.Status = datamodel.Pending
	repoNCConfig.FailureDetail = api.JsonErr{}

	return false, datamodel.PersistNCConfiguration(repoNCConfig, datamodel.DontAddToPendingQ)
}

func processNetworkContainerDisAssociate(nc *datamodel.NCConfiguration, ncLock map[string]string) (bool, *api.JsonErr) {
	if err := cnsInterface.DeleteNC(nc, ncLock); err != nil {
		// we should only retry internal errors. all else should be marked as failed
		removeFromQueue := err.HTTPStatus != http.StatusInternalServerError
		if removeFromQueue {
			nc.Status = datamodel.Failed
		}

		nc.FailureDetail = *err
		if persistErr := datamodel.PersistNCConfiguration(nc, datamodel.DontAddToPendingQ); persistErr != nil {
			return false, persistErr
		}
		return removeFromQueue, err
	}

	logger.Debugf("[CNS] DisAssociated NC %s from node %s", nc.NetworkContainerID, nc.GetAssociationInfo().NodeID)

	// In baremetal containers first deallocate MAC before freeing the IP.
	containerType := nc.GetAssociationInfo().ContainerType
	if api.IsBareMetalContainer(containerType) {
		nc.State = datamodel.DeallocateMAC
	} else {
		nc.State = datamodel.UnReserve
	}

	nc.Status = datamodel.Pending
	nc.FailureDetail = api.JsonErr{}
	return false, datamodel.PersistNCConfiguration(nc, datamodel.DontAddToPendingQ)
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

	loggershim.LS.Debugf("[DNCOperationStatus] OperationName: %s, NetworkContainerId: %s, Status: %s, ErrorCode: %s, Err:%+v", ncState, repoNCConfig.NetworkContainerID, status, errorCode, err)
}

func publishNetworkContainerWithPubSubProxy(
	ncGoalState *datamodel.NCGoalState,
	repoNCConfig *datamodel.NCConfiguration,
	sessionInfo map[string]string,
	stateTransitionDateTime time.Time,
	createdDateTime time.Time) (bool, *api.JsonErr) {
	// Release the session lock before making call to PubSubProxy
	sessionManager.EndWriteSession(sessionInfo)

	var secondaryIPs []string
	var publishError *api.JsonErr

	for _, secIP := range ncGoalState.SecondaryIPs {
		// split will always be at least len==1 with a non empty separator
		secondaryIPs = append(secondaryIPs, strings.Split(secIP, "/")[0])
	}

	publishNcRequest := pubsubproxy.PublishNetworkContainerRequest{
		NcID:             ncGoalState.NetworkContainerID,
		InfraVnetID:      ncGoalState.InfraNetworkID,
		NodeInterfaceIP:  ncGoalState.AssociatedInterfaceID,
		SubnetAuthToken:  ncGoalState.AccessToken,
		PrimaryIP:        strings.Split(ncGoalState.AllocatedIPSubnet, "/")[0],
		SecondaryIPs:     secondaryIPs,
		SubnetName:       ncGoalState.BaseSubnetName,
		Version:          strconv.FormatUint(ncGoalState.Version, 10),
		Policies:         make([]string, 0),
		VirtualNetworkID: ncGoalState.NetworkID,
		VlanID:           ncGoalState.VlanID,
		GreKey:           int(ncGoalState.GreKey),
	}

	if api.IsBareMetalContainer(repoNCConfig.GetAssociationInfo().ContainerType) {
		publishError = pubsubproxy.PublishBaremetalNetworkContainer(ncGoalState, repoNCConfig)
	} else {
		publishError = pubsubproxy.PublishNetworkContainer(publishNcRequest)
	}

	// Reacquire the session locks after making call to PubSubProxy
	sessionManager.BeginWriteSession(sessionInfo)

	if publishError != nil {
		logger.Errorf("publishNetworkContainerWithPubSubProxy failed for NC: %d, Error: %v",
			publishNcRequest.NcID, publishError)
		if publishError.HTTPStatus == http.StatusInternalServerError {
			return false, publishError
		}
	}

	// Retrieve the latest NC configuration as the current copy of the configuration
	// could have been updated while pubsubproxy processed the NC publish.
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

	if publishError != nil && publishError.HTTPStatus == http.StatusUnauthorized {
		// If the response code is authentication failure, retry the request for the preset duration.
		// If authentication failure is still seen fail the request
		// This workaround is needed because sometimes there is a latency in sending the new authentication token
		// to pubsub. This workaround hides that till the latency is addressed.
		// If the NC gets marked for delete while calling CNS / NMAgent, in case of error that request needs to be
		// marked as retriable error to avoid losing it and delete NC getting stuck
		if (time.Since(stateTransitionDateTime).Seconds() < unauthorizedErrRetryDurationSec) ||
			repoNCConfig.MarkedForDelete {
			msg := fmt.Sprintf("Create NC %s pending. Response code: %d, time elapsed: %.0f sec",
				repoNCConfig.NetworkContainerID,
				publishError.HTTPStatus,
				time.Since(stateTransitionDateTime).Seconds())
			return false, &api.JsonErr{
				Code:       api.ProcessingPending,
				Text:       msg,
				HTTPStatus: http.StatusUnauthorized}
		}

		// Mark the request as non-retriable if it fails for retry duration
		msg := fmt.Sprintf("Failed to publish NC: %s with status code: %v. Marking as non retriable.",
			repoNCConfig.NetworkContainerID, publishError.HTTPStatus)
		repoNCConfig.Status = datamodel.Failed
		repoNCConfig.FailureDetail = api.JsonErr{
			Code:       api.PublishingFailed,
			Text:       msg,
			HTTPStatus: http.StatusUnauthorized}

		return true, datamodel.PersistNCConfiguration(repoNCConfig, datamodel.DontAddToPendingQ)
	}

	if repoNCConfig.State == datamodel.Publish {
		repoNCConfig.State = datamodel.PrepareSecondaryIpCleanup
	}
	repoNCConfig.Status = datamodel.Pending
	repoNCConfig.FailureDetail = api.JsonErr{}

	creationTimeMs := time.Since(createdDateTime).Milliseconds()
	sendNCMetric(aiwrapper.NCProcessTimeMetric, float64(creationTimeMs),
		aiwrapper.Succeeded, "Container"+datamodel.Program.String(), repoNCConfig, nil)

	// Log NC Snapshot
	aiwrapper.LogNCSnapsnot(ncGoalState.NetworkContainerID, ncGoalState.NetworkID,
		ncGoalState.BaseSubnetName, ncGoalState.AssociatedNodeID)

	return false, datamodel.PersistNCConfiguration(repoNCConfig, datamodel.DontAddToPendingQ)
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

	if config.PubSubProxySettings.EnablePubSubProxy {
		return publishNetworkContainerWithPubSubProxy(ncGoalState, repoNCConfig, sessionInfo,
			stateTransitionDateTime, createdDateTime)
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
	ncRequest.Version = strconv.FormatUint(ncGoalState.Version, 10)
	ncRequest.VirtualNetworkID = ncGoalState.NetworkID
	ncRequest.VlanID = ncGoalState.VlanID
	ncRequest.GreKey = int(ncGoalState.GreKey)

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

		logger.Debugf("[CNS] Publish NC %s, body:%s ",
			ncGoalState.NetworkContainerID, ncRequest.String())
		// Release the session locks before making call to CNS
		sessionManager.EndWriteSession(sessionInfo)

		respPublishCNS, respPublishCNSErr :=
			cnsHttpClient.Post(publishNCURL, defaultContentType, &publishNCRequestBody)

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

			logger.Debugf("[CNS][Response] Publish NC: %s, Response: %+v, PublishStatusCode: %d",
				ncGoalState.NetworkContainerID, publishNCResponse.Response, publishNCResponse.PublishStatusCode)

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

				isRetriable, errJoinNetwork, _ := networkupdatehandler.DecodeNetworkJoinResponse(ncGoalState.NetworkID, resp, respErr)
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

func unpublishNetworkContainerWithPubSubProxy(
	ncGoalState *datamodel.NCGoalState,
	repoNCConfig *datamodel.NCConfiguration,
	sessionInfo map[string]string,
	stateTransitionDateTime time.Time) (bool, *api.JsonErr) {
	var (
		unpublishErr *api.JsonErr
	)
	// Release the session lock before making call to PubSubProxy
	sessionManager.EndWriteSession(sessionInfo)

	unpublishNcRequest := pubsubproxy.UnpublishNetworkContainerRequest{
		NcID:            ncGoalState.NetworkContainerID,
		InfraVnetID:     ncGoalState.InfraNetworkID,
		NodeInterfaceIP: ncGoalState.AssociatedInterfaceID,
		VnetID:          ncGoalState.NetworkID,
		SubnetName:      ncGoalState.BaseSubnetName,
		NcAllocatedIP:   ncGoalState.AllocatedIPSubnet,
		SubnetAuthToken: ncGoalState.AccessToken,
	}

	if api.IsBareMetalContainer(repoNCConfig.GetAssociationInfo().ContainerType) {
		unpublishErr = pubsubproxy.UnpublishBaremetalNetworkContainer(ncGoalState, repoNCConfig)
	} else {
		unpublishErr = pubsubproxy.UnpublishNetworkContainer(unpublishNcRequest)
	}

	// Reacquire the session locks after making call to PubSubProxy
	sessionManager.BeginWriteSession(sessionInfo)

	if unpublishErr != nil {
		logger.Errorf("unpublishNetworkContainerWithPubSubProxy failed for NC: %d, Error: %v",
			unpublishNcRequest.NcID, unpublishErr)
		if unpublishErr.HTTPStatus != http.StatusOK {
			// If the response code is authentication failure, retry the request for the preset duration.
			// If authentication failure is still seen fail the request
			// This workaround is needed because sometimes there is a latency in sending the new authentication token
			// to pubsub. This workaround hides that till the latency is addressed.
			if unpublishErr.HTTPStatus == http.StatusUnauthorized {
				if time.Since(stateTransitionDateTime).Seconds() < unauthorizedErrRetryDurationSec {
					msg := fmt.Sprintf("Delete NC %s pending. Response code: %d, time elapsed: %.0f sec",
						repoNCConfig.NetworkContainerID,
						unpublishErr.HTTPStatus,
						time.Since(stateTransitionDateTime).Seconds())
					return false, &api.JsonErr{
						Code:       api.ProcessingPending,
						Text:       msg,
						HTTPStatus: http.StatusUnauthorized}
				}

				// Mark the request as non-retriable if it fails for retry duration
				msg := fmt.Sprintf("Failed to unpublish NC: %s with status code: %v. Marking as non retriable.",
					repoNCConfig.NetworkContainerID, unpublishErr.HTTPStatus)
				repoNCConfig.Status = datamodel.Failed
				repoNCConfig.FailureDetail = api.JsonErr{
					Code:       api.UnpublishingFailed,
					Text:       msg,
					HTTPStatus: http.StatusUnauthorized}
				return true, datamodel.PersistNCConfiguration(repoNCConfig, datamodel.DontAddToPendingQ)
			}

			// Retry if error response code is anything other than http.StatusUnauthorized
			msg := fmt.Sprintf("Failed to unpublish NC: %s with status code: %v. Marking as retriable.",
				repoNCConfig.NetworkContainerID, unpublishErr.HTTPStatus)

			return false, &api.JsonErr{
				Code:       api.UnpublishingFailed,
				Text:       msg,
				HTTPStatus: unpublishErr.HTTPStatus}
		}
	}

	containerType := repoNCConfig.GetAssociationInfo().ContainerType
	if api.IsBareMetalContainer(containerType) {
		repoNCConfig.State = datamodel.DeallocateMAC
	} else {
		repoNCConfig.State = datamodel.UnReserve
	}

	repoNCConfig.Status = datamodel.Pending
	repoNCConfig.FailureDetail = api.JsonErr{}

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

	if config.PubSubProxySettings.EnablePubSubProxy {
		return unpublishNetworkContainerWithPubSubProxy(ncGoalState, repoNCConfig, sessionInfo, stateTransitionDateTime)
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
			cnsHttpClient.Post(unpublishNCURL, defaultContentType, &unpublishNCRequestBody)

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

			logger.Debugf("[CNS][Response] Unpublish NC: %s, Response: %+v, PublishStatusCode: %d",
				ncGoalState.NetworkContainerID, unpublishNCResponse.Response, unpublishNCResponse.UnpublishStatusCode)

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

				isRetriable, errJoinNetwork, _ := networkupdatehandler.DecodeNetworkJoinResponse(ncGoalState.NetworkID, resp, respErr)
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
			// todo: batch functionality is not general enough yet for non-managed cases.
			//   once specific cases have been ported, default to batch.
			if config.FeatureFlagSettings.EnableBatchAllocationProcessing {
				removeRequestFromPendingQueue, err = processNetworkContainerBatchReserve(repoNCConfig)
			} else {
				removeRequestFromPendingQueue, err = processNetworkContainerReserve(repoNCConfig)
			}

		case datamodel.AllocateLocalIP:

			// Local IP allocation is not needed for bare-metal containers.
			// In that case change the NC config state to AllocateMAC and return.
			if api.IsBareMetalContainer(repoNCConfig.GetAssociationInfo().ContainerType) {
				removeRequestFromPendingQueue, err = false, transitionNCTo(repoNCConfig, datamodel.AllocateMAC, datamodel.Pending)
			} else {
				removeRequestFromPendingQueue, err = processAllocateLocalIP(repoNCConfig)
			}

		case datamodel.AllocateMAC:
			removeRequestFromPendingQueue, err = macAllocator.ProcessAllocateMAC(repoNCConfig)

		case datamodel.Associate:
			fallthrough

		case datamodel.RefreshAssociation:
			removeRequestFromPendingQueue, err = processNetworkContainerAssociate(repoNCConfig, sessionInfo)

		case datamodel.Publish:
			removeRequestFromPendingQueue, err = processNetworkContainerPublish(repoNCConfig, sessionInfo, stateTransitionDateTime, createdDateTime)

		case datamodel.PrepareSecondaryIpCleanup:
			removeRequestFromPendingQueue, err = processPrepareSecondaryIpCleanup(repoNCConfig)

		case datamodel.Program:
			removeRequestFromPendingQueue, err = processNetworkContainerProgram(repoNCConfig)

		case datamodel.UnPublish:
			removeRequestFromPendingQueue, err = processNetworkContainerUnPublish(repoNCConfig, sessionInfo, stateTransitionDateTime)

		case datamodel.DisAssociate:
			removeRequestFromPendingQueue, err = processNetworkContainerDisAssociate(repoNCConfig, sessionInfo)

		case datamodel.DeallocateMAC:
			removeRequestFromPendingQueue, err = macAllocator.ProcessDeallocateMAC(repoNCConfig)

		case datamodel.UnReserve:
			removeRequestFromPendingQueue = false // there is no obvious case where unreserving requires removing the item from the pending queue
			err = processNetworkContainerUnReserve(repoNCConfig)

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
ExitLoop:
	for {
		select {
		case <-monitorTimerChannel:
			if replicaInstance.IsReplicaPrimary() {
				processPendingRequests(concurrentGoroutines, &inProcessRequestsMap)
			}
			break

		case <-cancelProcessing:
			if !config.PubSubProxySettings.EnablePubSubProxy {
				networkupdatehandler.NetworkMonitorStopProcessing <- true
			}

			if !config.DNCSettings.Managed {
				refreshNCStopProcessing <- true
			}

			if !disableTelemetry {
				select {
				case telemetryStopProcessing <- true:
					break
				default:
					break
				}
			}

			logger.Debugf("Received signal to exit")
			break ExitLoop
		}
	}
}

// Process and complete the given feature migration
func processFeatureMigration(feature string) {
	if feature == configuration.FeatureDisableReservationMapInSubnetConfig && !featureMigrationComplete {
		timerChannel := time.NewTicker(time.Second * 1).C
		for {
			select {
			case <-timerChannel:
				if replicaInstance.IsReplicaPrimary() {
					if err := datamodel.ProcessDisableReservationMapInSubnetConfig(feature); err == nil {
						featureMigrationComplete = true
						return
					}
				}
			}
		}
	}
}

// Monitor that periodically processes pending requests in the work item queue.
func processPendingRequests(
	concurrentGoroutines chan struct{},
	inProcessRequestsMap *sync.Map) {
	if config.DNCSettings.DisableReservationMapInSubnetConfig {
		processFeatureMigration(configuration.FeatureDisableReservationMapInSubnetConfig)
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

	if currentBatchRequestCount > 0 {
		logger.Debugf("Number of requests to process in the current batch: %d", currentBatchRequestCount)
	}

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
