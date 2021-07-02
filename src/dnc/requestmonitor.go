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
	removeFromPendingQ                    = true
	dontRemoveFromPendingQ                = false
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

	case datamodel.ReservationRequest:
		// Keep the NW and reservation lock while the current request completes the pending queue processing
		sessionInfo := map[string]string{
			datamodel.ReservationRepositoryName: datamodel.GetReservationIDFromKey(requestID),
			datamodel.NetworkRepositoryName:     requestContext.NetworkID,
		}
		sessionManager.BeginWriteSession(sessionInfo)
		defer sessionManager.EndWriteSession(sessionInfo)

		removeWorkItem, err = ipAllocator.ProcessIPReservationAllocationWithKey(requestID)

	case datamodel.ReservationSetRequest:
		reservationSetID, parseErr := datamodel.GetSetIDFromReservationSetKey(requestID)
		if parseErr != nil {
			logger.Errorf("could not get reservationSetID from request key: %s. error: %v. Removing request from pending queue",
				requestID, parseErr)
			// Remove the request from pending queue as it will never succeed
			//todo: add alerting on this error
			if err := datamodel.RemovePendingRequest(requestID); err != nil {
				logger.Errorf("Could not remove %s id %s from pending queue: %v", requestContext.Type.String(), requestID, err)
			}
		}

		// Keep the NW and reservationset lock while the current request completes the pending queue processing
		sessionInfo := map[string]string{
			datamodel.ReservationSetRepositoryName: reservationSetID,
			datamodel.NetworkRepositoryName:        requestContext.NetworkID,
		}
		sessionManager.BeginWriteSession(sessionInfo)
		defer sessionManager.EndWriteSession(sessionInfo)

		removeWorkItem, err = processReservationSetWithKey(requestID)

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

	case datamodel.SubnetCleanupRequest:
		removeWorkItem, err = processSubnetCleanupRequest(requestContext)

	case datamodel.NodeSetCleanupRequest:
		// node set lock is not required as it should already be marked for deletion.
		// any node requests associated to that node set will check that node set is active.
		removeWorkItem, err = processNodeSetCleanupRequest(requestContext.NodeSetName)
	}

	if err == nil && removeWorkItem {
		if err := datamodel.RemovePendingRequest(requestID); err != nil {
			logger.Errorf("Could not remove %s id %s from pending queue: %v", requestContext.Type.String(), requestID, err)
		}
	} else {
		if err != nil {
			requestContext.LastRetriedDateTime = time.Now().UTC()
		} else {
			requestContext.LastRetriedDateTime = time.Time{}
			requestContext.RetryInterval = requestContextRetryInterval
			requestContext.StateTransitionDateTime = time.Now().UTC()
		}

		logger.Errorf("Adding request: %+v back in pending queue due to error: %v", requestContext, err)
		if err := datamodel.AddRequestToPendingQueue(requestID, requestContext); err != nil {
			logger.Errorf("Could not add %s id %s to pending queue: %v", requestContext.Type.String(), requestID, err)
		}
	}
}

func processSubnetCleanupRequest(requestContext *datamodel.RequestContext) (bool, *api.JsonErr) {
	networkID := requestContext.NetworkID
	subnetName := requestContext.SubnetName

	// Remove the network containers in this subnet.
	ncList, err := datamodel.NcRepo.QueryNetworkContainersInSubnet(networkID, subnetName)
	if err != nil {
		return dontRemoveFromPendingQ, err
	}

	ncDeletionInitiated := false
	var errCleanup []string
	for _, nc := range ncList {
		// Create delete request for the NC if it's not marked for delete already
		if !nc.Config.MarkedForDelete {
			ncID := datamodel.GetNCIdFromKey(nc.RowKey)
			sessionInfo := map[string]string{datamodel.NCRepositoryName: ncID}
			sessionManager.BeginWriteSession(sessionInfo)
			if err := datamodel.ProcessNetworkContainerForRelease(networkID, ncID); err != nil {
				errCleanup = append(errCleanup, fmt.Sprintf("NC: %s error: %+v", nc.RowKey, err))
			}
			sessionManager.EndWriteSession(sessionInfo)
			ncDeletionInitiated = true
		}
	}

	// todo: If all the NCs are removed from the subnet, delete any explicit reservations in that subnet.
	//  this is not applicable to openai. This will be handled in a separate PR.

	if errCleanup != nil {
		errString := strings.Join(errCleanup, ", ")
		loggershim.LS.Errorf("could not trigger NC deletion request(s) for subnet cleanup: %s. error: %s",
			datamodel.GetSubnetKey(networkID, subnetName), errString)
		return dontRemoveFromPendingQ, api.InternalServerErrorf(api.StatusInternalServerError, errString)
	}

	// Return if the NC deletion was just initiated. The ncList read above is stale as those NCs which were in the failed
	// state now have their failureDetail reset. We need to read from NC table again to get the updated data
	if ncDeletionInitiated {
		return dontRemoveFromPendingQ, nil
	}

	// Check if any NC failed with non-retryable error. The delete subnet request will be failed as a result.
	var failedNC datamodel.NetworkContainerEntity
	for _, nc := range ncList {
		if nc.RowKey != "" && nc.Config.Status == datamodel.Failed {
			failedNC = nc
			break
		}
	}

	// Return for retrying the request if NCs are pending deletion and none of NCs are in the failed state
	if failedNC.RowKey == "" && len(ncList) > 0 {
		return dontRemoveFromPendingQ, api.InternalServerErrorf(api.ProcessingPending, "Deletion pending for %d NCs", len(ncList))
	}

	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	sessionManager.BeginWriteSession(sessionInfo)
	defer sessionManager.EndWriteSession(sessionInfo)

	requestedSubnetConfig, err := datamodel.TryQuerySubnetConfiguration(networkID, subnetName)
	if err != nil {
		loggershim.LS.Errorf("Failed to retrieve subnet configuration for subnet: %s in network: %s. Error: %v", subnetName, networkID, err)
		return dontRemoveFromPendingQ, err
	}

	if requestedSubnetConfig == nil {
		loggershim.LS.Errorf("[Inconsistent state] No subnet configuration for subnet: %s in network: %s", subnetName, networkID)
		return removeFromPendingQ, nil
	}

	// If the NC deletion failed, mark the subnet cleanup as failed
	if failedNC.RowKey != "" {
		requestedSubnetConfig.UpdateStatus = datamodel.SubnetCleanupFailed
		requestedSubnetConfig.FailureDetail = api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       fmt.Sprintf("could not delete NC: %s. NC failure detail: %+v", failedNC.RowKey, failedNC.Config.FailureDetail),
			HTTPStatus: http.StatusInternalServerError}
		if err := datamodel.PutSubnetConfiguration(requestedSubnetConfig); err != nil {
			loggershim.LS.Errorf("Failed to persist subnet configuration for subnet: %s in network: %s. Error: %v", subnetName, networkID, err)
			return dontRemoveFromPendingQ, api.InternalServerErrorf(api.StatusInternalServerError,
				"could not persist subnet: %s", datamodel.GetSubnetKey(networkID, subnetName))
		}

		loggershim.LS.Errorf("subnet cleanup failed for subnet: %s as NC: %s could not be deleted. NC failure details: %+v",
			datamodel.GetSubnetKey(networkID, subnetName), failedNC.RowKey, failedNC.Config.FailureDetail)
		return removeFromPendingQ, nil //todo: should the error be nil?
	}

	networkConfiguration, err := datamodel.TryQueryNetworkConfiguration(networkID)
	if err != nil {
		loggershim.LS.Errorf("Failed to retrieve network configuration for network: %s. Error: %v", networkID, err)
		return dontRemoveFromPendingQ, err
	}

	if networkConfiguration == nil {
		loggershim.LS.Debugf("[Inconsistent state] No network configuration for network: %s", networkID)
		return removeFromPendingQ, nil
	}

	// Update markedForDelete to false to allow user to be able to join this subnet again
	// Joined is set to false here so that if user calls GET status on the subnet while the cleanup is in process
	// it still returns the status as CleanupPending instead of 404.
	requestedSubnetConfig.Joined = false
	requestedSubnetConfig.MarkedForDelete = false
	requestedSubnetConfig.UpdateStatus = datamodel.SubnetUpdateCompleted
	if err := datamodel.PutSubnetConfiguration(requestedSubnetConfig); err != nil {
		loggershim.LS.Errorf("Failed to persist subnet configuration for subnet: %s in network: %s. Error: %v", subnetName, networkID, err)
		return dontRemoveFromPendingQ, api.InternalServerErrorf(api.StatusInternalServerError,
			"could not persist subnet: %s. error: %+v", datamodel.GetSubnetKey(networkID, subnetName), err)
	}

	// Delete the network configuration if there are no subnets left in 'joined' state.
	subnetConfigs, err := datamodel.TryQueryAllSubnetConfigsInNetworkConfig(networkConfiguration)
	if err != nil {
		loggershim.LS.Errorf("Failed to get all the subnet configs in network: %s. Error: %v", networkID, err)
		return dontRemoveFromPendingQ, err
	}

	// Delete the network if there are no 'Joined' subnets in this network
	for subnet, subnetConfig := range subnetConfigs {
		if subnetConfig.Joined {
			loggershim.LS.Debugf("Cannot delete the network: %s as subnet: %s. Joined: %t", networkID, subnet, subnetConfig.Joined)
			// No further processing is needed and we should remove the pending work item for subnet cleanup request
			return removeFromPendingQ, nil
		}
	}

	loggershim.LS.Debugf("Deleting network: %s", networkID)
	if config.PubSubProxySettings.EnablePubSubProxy {
		// Unsubscribe vnet notifications for PubSubProxy
		pubsubproxy.UnregisterVnet(networkID)
	}

	// Delete the network configuration and subnet configurations in the network
	if err = datamodel.DeleteNetworkConfiguration(networkID); err != nil {
		loggershim.LS.Errorf("could not delete the network: %s", networkID)
		return dontRemoveFromPendingQ, err
	}

	return removeFromPendingQ, nil
}

// Routine that processes virtual network request.
func processVirtualNetworkRequest(requestContext *datamodel.RequestContext) (bool, *api.JsonErr) {
	repoNetworkConfiguration, err := datamodel.TryQueryNetworkConfiguration(requestContext.NetworkID)
	if err != nil {
		return false, err
	}

	if repoNetworkConfiguration == nil {
		logger.Debugf("Failed to find configuration in repository for network %s", requestContext.NetworkID)
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
				repoNetworkContainerData, err := datamodel.NcRepo.TryQueryNetworkContainer(networkID, networkContainerID)
				if err != nil {
					return nil, err
				}

				if repoNetworkContainerData.RowKey == "" {
					logger.Debugf("Data for network container %s not found", networkContainerID)
					continue
				}

				if repoNetworkContainerData.Config.LocalIP != "" {
					localIPSubnetString := repoNetworkContainerData.Config.LocalIP + utils.PrefixDelimiter + utils.IPV4DefaultPrefix
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
	nc *datamodel.NetworkContainerEntity) (bool, *api.JsonErr) {
	var (
		markProcessingFailed bool
		allocatedIP          *net.IPNet
		failureDetail        *api.JsonErr
		greErr,
		allocationErr error
		vlanID uint16
	)

	nodeID := nc.Config.GetAssociationInfo().NodeID

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
		vlanID = nodeConfiguration.NetworkVlanIDMap[nc.Config.NetworkID]
	}

	greErr = datamodel.CheckGreConditions(
		nodeConfiguration.NodeID,
		nc.Config.NetworkContainerID,
		nodeConfiguration.UpdateGreCapable,
		nc.Config.GreKey,
		vlanID)

	//Allocate GRE Key for this NC
	if greErr == nil {
		interfaceID := ""
		if !nc.Config.ImplicitReservation {
			interfaceID = nc.Config.RequestV1.AssociationInfo.InterfaceID
		} else {
			interfaceID = nc.Config.RequestV2.AssociationInfo.InterfaceID
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
						"%s NetworkContainer: %s, Node: %s", nextGreErr.Text, nc.Config.NetworkContainerID, nodeID)}
				goto Cleanup

			}
			nc.Config.GreKey = greKey
			nodeConfiguration.InterfaceGreKeyMap[interfaceID][nc.Config.ReservationID] = greKey
			logger.Debugf("Assigning GreKey for NC: %s, Node: %s, interface ID is %s, reservation ID is %s, GreKey is %d",
				nc.Config.NetworkContainerID, nodeID, interfaceID, nc.Config.ReservationID, greKey)

			nodeErr := datamodel.NodeConfigRepo.PersistNodeConfiguration(nodeConfiguration)
			if nodeErr != nil {
				return false, nodeErr
			}
		} else {
			logger.Debugf("Could not find Interface AssiciationInfo while assinging GreKey for NC: %s, Node: %s",
				nc.Config.NetworkContainerID, nodeID)
		}

	} else {
		logger.Debugf(greErr.Error())
	}

	nc.Config.LocalIP = allocatedIP.IP.String()
	if config.DNCSettings.Managed {
		return false, transitionNCStateAndLog(nc, datamodel.AllocateLocalIP, datamodel.Publish, datamodel.Pending)
	} else {
		return false, transitionNCStateAndLog(nc, datamodel.AllocateLocalIP, datamodel.Associate, datamodel.Pending)
	}

Cleanup:
	if markProcessingFailed { // todo: this is always true once this is reached, clean once gotos are removed
		nc.Config.Status = datamodel.Failed
		nc.Config.FailureDetail = *failureDetail
	}

	if err = datamodel.NcRepo.PersistNetworkContainerData(nc); err != nil {
		return false, err
	}

	return markProcessingFailed, nil
}

// This will create the pending IP cleanup job for ToBeDeleted Secondary IPs during NC update.
func processPrepareSecondaryIpCleanup(nc *datamodel.NetworkContainerEntity, createdDateTime time.Time) (bool, *api.JsonErr) {
	if config.PubSubProxySettings.EnablePubSubProxy {
		ncKey := datamodel.GetNCKey(nc.Config.NetworkID, nc.Config.NetworkContainerID)
		loggershim.LS.Debugf("processPrepareSecondaryIpCleanup, NCKey %s", ncKey)

		// get reservation
		var subnetName = ""
		if nc.Config.ImplicitReservation {
			subnetName = nc.Config.RequestV2.AllocationRequest.SubnetName
		} else {
			loggershim.LS.Debugf("processPrepareSecondaryIpCleanup: Getting Subnet name from Reservation record, nckey %s, reservationId %s",
				ncKey,
				nc.Config.ReservationID)
			repoReservation, err := datamodel.TryQueryIPReservation(nc.Config.NetworkID, nc.Config.ReservationID)
			if err != nil {
				loggershim.LS.Errorf("processPrepareSecondaryIpCleanup: Failed to query associated Reservation repo, NcKey %s, reservationId %s, err %+v",
					ncKey,
					nc.Config.ReservationID,
					err)
				return false, err
			}

			if repoReservation.RowKey == "" {
				loggershim.LS.Errorf("processPrepareSecondaryIpCleanup: associated Reservation config not found, NcKey %s, reservationId %s, err %+v",
					ncKey,
					nc.Config.ReservationID,
					err)
				return false, err
			}

			subnetName = repoReservation.Configuration.AllocationRequest.SubnetName
		}

		if subnetName == "" {
			return true, api.InternalServerErrorf(api.InvalidSubnetName,
				"ProcessPrepareSecondaryIpCleanup: associated subnetName is empty, NcKey %s, reservationId %s",
				ncKey, nc.Config.ReservationID)
		}

		// We have already updated IpAllocations table with ToBeDeletedOwner for ToBeDeletedIpIds
		// in Reserve state, thus it is safe to request all pending deleting IPs and
		// create or Merge the PendingResourceCleanupJob
		err := ipAllocator.CreateIpCleanupRequest(nc.Config.NetworkID, subnetName, nc.Config.NetworkContainerID)
		if err != nil {
			loggershim.LS.Errorf("processPrepareSecondaryIpCleanup: Failed to create the cleanup request err:  %+v",
				err)
			return false, err
		}

		creationTimeMs := time.Since(createdDateTime).Milliseconds()
		sendNCMetric(aiwrapper.NCProcessTimeMetric, float64(creationTimeMs), aiwrapper.Succeeded, "Container"+datamodel.Program.String(), &nc.Config, nil)
		aiwrapper.LogNCSnapsnot(nc.Config.NetworkContainerID, nc.Config.NetworkID, subnetName, nc.Config.GetAssociationInfo().NodeID)
	}

	// todo: we have to log program completion before transitioning state, since program can be transitioned to
	//   from refresh but we don't want to log completion in those cases
	nc.Config.LogStateCompletion(datamodel.Program)
	return false, transitionNCStateAndLog(nc, datamodel.PrepareSecondaryIpCleanup, datamodel.Program, datamodel.Pending)
}

// validateAndGetExplicitReservationID - If the request has specified ReservationID, validate that reservation and use that reservationID.
// If the request has instead specified ReservationSetID, find unassigned reservation from that set. If there is no unassigned reservation
// left in that reservation set, return error
func validateAndGetExplicitReservationID(
	repoNCConfig *datamodel.NCConfiguration) (string, bool, *api.JsonErr) {
	var (
		reservation            datamodel.ReservationRow
		reservationSetID       string
		networkID              = repoNCConfig.NetworkID
		ncRequest              = repoNCConfig.RequestV1
		requestedReservationID = ncRequest.ReservationID
		err                    *api.JsonErr
	)

	// Check if the provided reservation belongs to a reservation set
	if requestedReservationID != "" {
		reservation, err = datamodel.IpReservationRepo.TryQueryIPReservation(networkID, requestedReservationID)
		if err != nil {
			return "", true, api.InternalServerErrorf(api.StatusInternalServerError,
				"could not retrieve reservation: %s. error: %v", requestedReservationID, err)
		}

		if reservation.Configuration == nil {
			return "", false, api.BadRequestf(api.InvalidReservationID, "could not find reservation for ID: %s", requestedReservationID)
		}

		if reservation.SetKey != "" {
			reservationSetID = datamodel.GetReservationIDFromKey(reservation.SetKey)
		}
	} else {
		reservationSetID = ncRequest.ReservationSetID
	}

	// If the reservation belongs to a set find unassigned reservation from that set. If there is no unassigned reservation
	// left in that reservation set, return error
	if reservationSetID != "" {
		return validateAndGetExplicitReservationIDFromSet(repoNCConfig, reservationSetID)
	}

	// If the reservation doesn't belong to a set, associate the NCID with the reservation if reservation is free
	sessionInfo := map[string]string{datamodel.ReservationRepositoryName: requestedReservationID}
	sessionManager.BeginWriteSession(sessionInfo)
	defer sessionManager.EndWriteSession(sessionInfo)

	ncID := repoNCConfig.NetworkContainerID
	// Return error if the reservation is already associated with a different NC
	if reservation.Configuration.AssociatedNCID != "" && reservation.Configuration.AssociatedNCID != ncID {
		return "", false, api.BadRequestf(api.ReservationInUse, "specified reservation: %s is already assigned to NC: %s",
			requestedReservationID, reservation.Configuration.AssociatedNCID)
	}

	// Associate the NCID if not already associated
	if reservation.Configuration.AssociatedNCID != ncID {
		reservation.Configuration.AssociatedNCID = ncID
		if err = datamodel.IpReservationRepo.PersistIPReservation(reservation); err != nil {
			return "", true, api.InternalServerErrorf(api.StatusInternalServerError,
				"could not persist updated reservation: %+v. error: %v", reservation, err)
		}
		loggershim.LS.Debugf("Associated NCID: %s with reservation: %s", ncID, requestedReservationID)
	}

	return requestedReservationID, true, nil
}

func validateAndGetExplicitReservationIDFromSet(
	repoNCConfig *datamodel.NCConfiguration,
	reservationSetID string) (string, bool, *api.JsonErr) {
	var (
		sessionInfo = make(map[string]string)
		ncID        = repoNCConfig.NetworkContainerID
		networkID   = repoNCConfig.NetworkID
		ncRequest   = repoNCConfig.RequestV1
		err         *api.JsonErr
	)

	loggershim.LS.Debugf("NC request for ID: %s is using reservation set: %s", ncID, reservationSetID)
	sessionInfo[datamodel.ReservationSetRepositoryName] = reservationSetID
	sessionManager.BeginWriteSession(sessionInfo)
	defer sessionManager.EndWriteSession(sessionInfo)

	// Check if the reservation set is in a completed state
	var reservationSet *datamodel.ReservationSet
	reservationSet, err = datamodel.IpReservationSetRepo.TryQueryIPReservationSet(networkID, reservationSetID)
	if err != nil {
		return "", true, api.InternalServerErrorf(api.StatusInternalServerError,
			"could not retrieve reservation set: %s. error: %v", reservationSetID, err)
	}

	if reservationSet == nil {
		return "", false, api.BadRequestf(api.InvalidReservationSetID, "could not find reservation set: %s", reservationSetID)
	}

	// Fail the request if the set is already marked for deletion
	if reservationSet.MarkedForDelete {
		return "", false, api.BadRequestf(api.InvalidReservationRequest,
			"reservation set: %s is marked for deletion. Cannot use the reservations from that set", reservationSetID)
	}

	if reservationSet.Status != datamodel.Completed {
		if reservationSet.Status == datamodel.Failed {
			return "", false, api.BadRequestf(api.ReservationSetProcessingFailed, "reservation set: %s processing in failed state", reservationSetID)
		}

		return "", false, api.BadRequestf(api.ReservationSetProcessingPending, "reservation set: %s processing in pending state", reservationSetID)
	}

	// Handle the case where the request included a reservation ID that belongs to this set
	if ncRequest.ReservationID != "" {
		// Read the reservation again and verify that either it's still free or associated with this NC.
		// This is needed after acquiring the reservationset lock as the reservation may have already been given out
		var reservation datamodel.ReservationRow
		if reservation, err = datamodel.IpReservationRepo.TryQueryIPReservation(networkID, ncRequest.ReservationID); err != nil {
			return "", true, api.InternalServerErrorf(api.StatusInternalServerError, "could not retrieve reservation: %s. error: %v",
				ncRequest.ReservationID, err)
		}

		if reservation.Configuration.AssociatedNCID != "" && reservation.Configuration.AssociatedNCID != ncID {
			return "", false, api.BadRequestf(api.ReservationInUse, "specified reservation: %s is already assigned to NC: %s",
				ncRequest.ReservationID, reservation.Configuration.AssociatedNCID)
		}

		if reservation.Configuration.AssociatedNCID != ncID {
			reservation.Configuration.AssociatedNCID = ncID
			if err = datamodel.IpReservationRepo.PersistIPReservation(reservation); err != nil {
				return "", true, api.InternalServerErrorf(api.StatusInternalServerError,
					"could not persist updated reservation: %+v. error: %v", reservation, err)
			}
			loggershim.LS.Debugf("Associated NCID: %s with reservation: %s from reservation set: %s",
				ncID, reservation.Configuration.ReservationID, reservationSetID)
		}

		return reservation.Configuration.ReservationID, true, nil
	}

	// todo: optimize the logic to get a free reservation from the set to batch processing so that multiple NC requests
	//  for the same reservation set can proceed in parallel without the need for lock.
	//  Not a priority for the initial change to enable this feature.
	// Associate NC with unassigned reservation
	var reservationID string
	reservationID, err = datamodel.IpReservationRepo.AssociateNCWithReservationInSet(networkID, ncID, ncRequest.ReservationSetID)
	if err != nil {
		loggershim.LS.Errorf("NC: %s request failed. error: %v", ncID, err)
		// Non-retryable error if the returned error indicates that there is no available reservation
		if err.Code != api.ReservationNotFound {
			return "", true, err
		}

		return "", false, err
	}

	loggershim.LS.Debugf("Associated NCID: %s with reservation: %s from reservation set: %s", ncID, reservationID, reservationSetID)

	return reservationID, true, nil
}

func processNetworkContainerReserve(
	nc *datamodel.NetworkContainerEntity) (bool, *api.JsonErr) {
	var (
		err                  *api.JsonErr
		failureDetail        *api.JsonErr
		markProcessingFailed bool
		repoReservation      datamodel.ReservationRow
		reservationID        = nc.Config.ReservationID
	)

	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: nc.Config.NetworkID}

	// Acquire Node lock if the ContainerType is WebApps
	if nc.Config.GetAssociationInfo().ContainerType == cns.WebApps {
		sessionInfo[datamodel.NodeRepositoryName] = nc.Config.GetAssociationInfo().NodeID
	}

	if !nc.Config.ImplicitReservation {
		var isErrRetryable bool
		if reservationID, isErrRetryable, err = validateAndGetExplicitReservationID(&nc.Config); err != nil {
			loggershim.LS.Errorf("validateAndGetExplicitReservationID failed for NC: %s error: %+v", nc.Config.NetworkContainerID, err)
			// If the error is retryable, do not remove the request from pending queue
			if isErrRetryable {
				return false, err
			}

			// If the error is non retryable mark the processing as failed
			markProcessingFailed = true
			failureDetail = err
			goto Cleanup
		}

		// Update the reservation ID in network container config
		nc.Config.ReservationID = reservationID
	}

	sessionInfo[datamodel.ReservationRepositoryName] = reservationID

	sessionManager.BeginWriteSession(sessionInfo)
	defer sessionManager.EndWriteSession(sessionInfo)

	repoReservation, err = datamodel.TryQueryIPReservation(nc.Config.NetworkID, reservationID)
	if err != nil {
		return false, err
	}

	if repoReservation.Configuration == nil {
		markProcessingFailed = true
		failureDetail = api.BadRequestf(api.ReservationNotFound, "Failed to find reservation %s associated with network container %s",
			nc.Config.ReservationID, nc.Config.NetworkContainerID)
		goto Cleanup
	}

	if len(nc.Config.RequestV2.ToBeDeletedIPIds) > 0 {
		loggershim.LS.Debugf("processNetworkContainerReserve: Process deleting secondary Ips NetworkId %s, ReservationId %s",
			nc.Config.NetworkID, nc.Config.ReservationID)

		// In case pubsub poxy is enabled, wait for publish job to finish before deleting the ips during scale down
		// Thus set the ToBeDeletedOwner for the deleting ips
		if config.PubSubProxySettings.EnablePubSubProxy {
			// It is okay if the associated ips are empty, it will still get cleaned up by the background cleanup monitor
			var ipsPendingCleanup = make(map[string]string)
			for _, allocID := range nc.Config.RequestV2.ToBeDeletedIPIds {
				if ipAddress, found := repoReservation.Configuration.SecondaryAllocations[allocID]; found {
					ipsPendingCleanup[allocID] = ipAddress
				}
			}

			reservationKey := datamodel.GetReservationKey(nc.Config.NetworkID, nc.Config.ReservationID)
			ncKey := datamodel.GetNCKey(nc.Config.NetworkID, nc.Config.NetworkContainerID)
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
			err := ipAllocator.DeallocateIPs(*repoReservation.Configuration, nc.Config.RequestV2.ToBeDeletedIPIds)
			if err != nil {
				return false, err
			}
		}

		// Now deleting
		for _, allocID := range nc.Config.RequestV2.ToBeDeletedIPIds {
			delete(repoReservation.Configuration.SecondaryAllocations, allocID)
		}
		if err := datamodel.IpReservationRepo.PersistIPReservation(repoReservation); err != nil {
			return false, err
		}
	}

	// If the reservation status is pending, process IP allocation
	if repoReservation.Configuration.Status == datamodel.Pending {
		ipReservationAllocationComplete, err := ipAllocator.ProcessIPReservationAllocation(repoReservation)

		// Return if the IP reservation allocation did not complete. This indicates that
		// the operation ran into retriable error.
		if !ipReservationAllocationComplete {
			return false, err
		}
	}

	if repoReservation.Configuration.Status == datamodel.Completed {
		repoSubnetConfiguration, err := datamodel.TryQuerySubnetConfiguration(
			nc.Config.NetworkID,
			repoReservation.Configuration.AllocationRequest.SubnetName)
		if err != nil {
			return true, err
		}

		if repoSubnetConfiguration == nil {
			markProcessingFailed = true
			failureDetail = api.BadRequestf(api.SubnetNotFound, "Configuration for Subnet %s in Network %v not found.",
				repoReservation.Configuration.AllocationRequest.SubnetName, nc.Config.NetworkID)
			goto Cleanup
		}

		nc.Config.ProgrammedTagsInfo = make(map[string]uint64)

		for _, tag := range repoSubnetConfiguration.Tags {
			tagsDefinitionKey := datamodel.GetTagDefintionKey(tag.Type, tag.Name)
			nc.Config.ProgrammedTagsInfo[tagsDefinitionKey] = 0
		}

		return false, transitionNCStateAndLog(nc, datamodel.Reserve, datamodel.AllocateLocalIP, datamodel.Pending)
	}

	if repoReservation.Configuration.Status == datamodel.Failed {
		markProcessingFailed = true
		failureDetail = &repoReservation.Configuration.FailureDetail
	}

Cleanup:
	if markProcessingFailed {
		nc.Config.Status = datamodel.Failed
		nc.Config.FailureDetail = *failureDetail
	}

	if err = datamodel.NcRepo.PersistNetworkContainerData(nc); err != nil {
		return false, err
	}

	return markProcessingFailed, nil
}

func processNetworkContainerBatchReserve(nc *datamodel.NetworkContainerEntity) (bool, *api.JsonErr) {
	sessionInfo := map[string]string{datamodel.ReservationRepositoryName: nc.Config.ReservationID}

	sessionManager.BeginWriteSession(sessionInfo)
	defer sessionManager.EndWriteSession(sessionInfo)

	reservation, apiErr := datamodel.TryQueryIPReservation(nc.Config.NetworkID, nc.Config.ReservationID)
	if apiErr != nil {
		return false, apiErr
	}

	if reservation.Configuration == nil {
		return markNCAsFailed(nc, *api.NotFoundf(api.ReservationNotFound, "could not find reservation %s for nc %s",
			nc.Config.NetworkContainerID, nc.Config.ReservationID))
	}

	res, err := resDispatcher.Reserve(reservation, nc.Config.RequestV2.ToBeDeletedIPIds)
	if err != nil {
		return false, api.InternalServerError(api.StatusInternalServerError, err.Error())
	}

	switch res.Configuration.Status {
	case datamodel.Failed:
		return markNCAsFailed(nc, res.Configuration.FailureDetail)
	case datamodel.Completed:
		return false, transitionNCStateAndLog(nc, datamodel.Reserve, datamodel.AllocateLocalIP, datamodel.Pending)
	default:
		return false, api.InternalServerError(api.StatusInternalServerError, "unexpected reservation status")
	}
}

func processNetworkContainerUnReserve(nc *datamodel.NetworkContainerEntity) *api.JsonErr {

	// create a MAC cleanup request if needed
	containerType := nc.Config.GetAssociationInfo().ContainerType
	if api.IsBareMetalContainer(containerType) && nc.Config.MACAddress != "" {
		if err := macAllocator.CreateMacCleanupRequest(nc); err != nil {
			return err
		}
	} else {
		logger.Debugf("[MacAllocator] Skipping MAC cleanup. ContainerType:%s, MAC:%s",
			containerType, nc.Config.MACAddress)
	}

	if !nc.Config.ImplicitReservation {
		sessionInfo := map[string]string{
			datamodel.ReservationRepositoryName: nc.Config.ReservationID,
		}
		sessionManager.BeginWriteSession(sessionInfo)
		defer sessionManager.EndWriteSession(sessionInfo)

		if err := datamodel.UpdateNCToReservationAssociation(nc, true); err != nil {
			return err
		}

		return transitionNCStateAndLog(nc, datamodel.UnReserve, datamodel.Purge, datamodel.Pending)
	}

	sessionInfo := map[string]string{
		datamodel.ReservationRepositoryName: nc.Config.ReservationID,
	}

	// Acquire the lock only if DisableReservationMapInSubnetConfig is set to false.
	// Lock is required to update the SubnetConfiguration table which is required only if the flag is set to false
	if !config.DNCSettings.DisableReservationMapInSubnetConfig {
		sessionInfo[datamodel.NetworkRepositoryName] = nc.Config.NetworkID
	}

	// Acquire Node lock only if needed
	reservation, err := datamodel.TryQueryIPReservation(nc.Config.NetworkID, nc.Config.ReservationID)
	if err != nil {
		return err
	}
	if reservation.Configuration != nil && len(reservation.Configuration.AllocationRequest.NodeConstraint) != 0 {
		sessionInfo[datamodel.NodeRepositoryName] = nc.Config.GetAssociationInfo().NodeID
	}

	sessionManager.BeginWriteSession(sessionInfo)
	defer sessionManager.EndWriteSession(sessionInfo)

	if err := ipAllocator.DeleteIPReservation(reservation, true); err != nil {
		return err
	}

	return transitionNCStateAndLog(nc, datamodel.UnReserve, datamodel.Purge, datamodel.Pending)
}

func processNetworkContainerPurge(
	nc *datamodel.NetworkContainerEntity,
	createdDateTime time.Time) (bool, *api.JsonErr) {
	sessionInfo := map[string]string{datamodel.NodeRepositoryName: nc.Config.GetAssociationInfo().NodeID}
	sessionManager.BeginWriteSession(sessionInfo)

	var err *api.JsonErr
	defer func() {
		sessionManager.EndWriteSession(sessionInfo)
		if err != nil {
			logger.Errorf("processNetworkContainerPurge failed for NC: %s with error: %v", nc.Config.NetworkContainerID, err)
		}
	}()

	if nc.Config.ImplicitReservation &&
		nc.Config.ReservationID != "" {
		reservation, err := datamodel.TryQueryIPReservation(nc.Config.NetworkID, nc.Config.ReservationID)
		if err != nil {
			return false, err
		}

		if reservation.Configuration != nil {
			logger.Debugf("[processNetworkContainerPurge] Associated implicit reservation record exists for NC id %s, reservation id %s, config %+v", nc.Config.NetworkContainerID, nc.Config.ReservationID, reservation.Configuration)
			// Reservation record still exists (which was created in Prepare state). AllocatedSubnet
			// must be null in that case. If not, then it is a bug as it must have been deleted in unreserve state
			if reservation.Configuration.AllocatedSubnet != "" {
				msg := fmt.Sprintf("Delete NC Key %s failed as associated Reservation is not deleted. reservation id %s, config %+v",
					datamodel.GetNCKey(nc.Config.NetworkID, nc.Config.NetworkContainerID),
					nc.Config.ReservationID,
					reservation.Configuration)
				return true, &api.JsonErr{
					Code:       api.NetworkContainerStateInvalid,
					Text:       msg,
					HTTPStatus: http.StatusBadRequest}
			}

			if err := datamodel.IpReservationRepo.DeleteReservation(reservation.RowKey); err != nil {
				return false, err
			}
		}
	}

	err = datamodel.PurgeNetworkContainerAssociations(nc, networkcontainer.ShouldUpdateVlan(nc.Config.GetAssociationInfo().ContainerType))
	if err != nil {
		return false, err
	}

	err = datamodel.NcRepo.DeleteNetworkContainer(nc.Config.NetworkID, nc.Config.NetworkContainerID)
	if err != nil {
		return false, err
	}

	deletionTimeMs := time.Since(createdDateTime).Milliseconds()
	sendNCMetric(aiwrapper.NCProcessTimeMetric, float64(deletionTimeMs),
		aiwrapper.Succeeded, "Container"+datamodel.Purge.String(), &nc.Config, nil)

	return true, nil
}

func processNetworkContainerAssociate(
	nc *datamodel.NetworkContainerEntity,
	sessionInfo map[string]string) (bool, *api.JsonErr) {
	sessionInfoRead := map[string]string{
		datamodel.ReservationRepositoryName: nc.Config.ReservationID,
		datamodel.NetworkRepositoryName:     nc.Config.NetworkID,
		datamodel.NodeRepositoryName:        nc.Config.GetAssociationInfo().NodeID,
	}
	sessionManager.BeginReadSession(sessionInfoRead)
	ncGoalState, isRetriableError, ncGoalStateErr := datamodel.GetNCGoalState(&nc.Config)
	sessionManager.EndReadSession(sessionInfoRead)

	if ncGoalStateErr != nil {
		if isRetriableError {
			return false, ncGoalStateErr
		}

		nc.Config.Status = datamodel.Failed
		nc.Config.FailureDetail = *ncGoalStateErr
		return true, datamodel.NcRepo.PersistNetworkContainerData(nc)
	}

	containerTypes.Store(ncGoalState.ContainerType, true)

	var (
		resp cns.CreateNetworkContainerResponse
		body bytes.Buffer
	)

	info, removeRequestFromPendingQueue, err := datamodel.GetNetworkContainerRequest(ncGoalState, nc)
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
		return false, api.BadRequestf(api.AssociationFailed, "could not POST to URL: %s for NC: %s. error: %s",
			createOrUpdateNCURL, ncGoalState.NetworkContainerID, respErr.Error())
	}

	defer w.Body.Close()
	if respErr = utils.DecodeResponse(w, &resp); respErr != nil {
		return false, api.BadRequestf(api.AssociationFailed, "could not associate NC: %s with Node: %s. error: %s",
			ncGoalState.NetworkContainerID, ncGoalState.NodeInterfaceID, respErr.Error())
	}

	logger.Debugf("[CNS][Response] create/update NC %s %+v", ncGoalState.NetworkContainerID, resp)

	// Retrieve the latest NC configuration as the current copy of the configuration
	// could have been updated while we were in the process of calling cns.
	retrievedRepoNCData, ncRetrievalErr := datamodel.NcRepo.TryQueryNetworkContainer(ncGoalState.NetworkID, ncGoalState.NetworkContainerID)
	if ncRetrievalErr != nil {
		return false, api.InternalServerErrorf(api.AssociationFailed, "could not associate NC: %s with Node: %s. error: %s",
			ncGoalState.NetworkContainerID, ncGoalState.NodeInterfaceID, ncRetrievalErr.String())
	}

	if retrievedRepoNCData.RowKey == "" {
		return true, api.BadRequestf(api.AssociationFailed, "could not associate NC: %s with Node: %s as data for NC is found",
			ncGoalState.NetworkContainerID, ncGoalState.NodeInterfaceID)
	}

	// Copy over the delete flag as NC could have been deleted when we were invoking CNS to create NC.
	nc.Config.MarkedForDelete = retrievedRepoNCData.Config.MarkedForDelete

	// If the NC is marked for deletion, copy the state as it could have been changed too
	if nc.Config.MarkedForDelete {
		nc.Config.State = retrievedRepoNCData.Config.State
	}

	// If the NC gets marked for delete while calling CNS, in case of error that request needs to be
	// marked as retriable error to avoid losing it and delete NC getting stuck
	if resp.Response.ReturnCode != 0 && !nc.Config.MarkedForDelete {
		// If the failure is due to Unsupported Orchestrator type, it indicates that the CNS state file is lost
		// and doesn't have the orchestrator type. Try sending the orchestrator type to CNS.
		if resp.Response.ReturnCode == restserver.UnsupportedOrchestratorType {
			logger.Debugf("Calling SetOrchestratorType for node: %s as CNS returned UnsupportedOrchestratorType error",
				nc.Config.GetAssociationInfo().NodeID)
			err := cnsInterface.SetOrchestratorType(nc.Config.GetAssociationInfo().NodeID,
				ncGoalState.NodeInterfaceID, ncGoalState.OrchestratorType)
			// If the operation succeeds or fails due to internal error, mark the request as retriable. This will
			// result in associate / refresh associate to be retried. If SetOrchestratorType succeeded, associate
			// task will succeed in the subsequent attempt. If SetOrchestratorType failed with retriable error,
			// If the operation fails due to error code returned by CNS, continue with marking the error as non-retriable
			if err == nil || err.HTTPStatus == http.StatusInternalServerError {
				if err == nil {
					logger.Debugf("Successfully set the orchestrator type: %s for Node: %s",
						ncGoalState.OrchestratorType, nc.Config.GetAssociationInfo().NodeID)
				} else {
					logger.Errorf("Failed to set the orchestrator type: %s for Node: %s due to error: %v",
						ncGoalState.OrchestratorType, nc.Config.GetAssociationInfo().NodeID, err)
				}
				return false, api.InternalServerErrorf(api.AssociationFailed,
					"Associate NC %v to node %v failed due to invalid orchestrator type set",
					ncGoalState.NetworkContainerID, nc.Config.GetAssociationInfo().NodeID)
			}
		}

		failureDetail := api.BadRequestf(api.AssociationFailed, "could not associate NC: %s with Node: %s due to response: %+v",
			ncGoalState.NetworkContainerID, ncGoalState.NodeInterfaceID, resp.Response)
		nc.Config.Status = datamodel.Failed
		nc.Config.FailureDetail = *failureDetail
		return true, datamodel.NcRepo.PersistNetworkContainerData(nc)
	}

	nc.Config.ProgrammedVnetVersion = ncGoalState.VnetVersion

	if nc.Config.State == datamodel.RefreshAssociation {
		// processing time is not logged in NC when refreshed, since we want to only keep the initial processing time
		return false, transitionNCState(nc, datamodel.Program, datamodel.Pending)
	}

	return false, transitionNCStateAndLog(nc, datamodel.Associate, datamodel.Publish, datamodel.Pending)
}

func processNetworkContainerDisAssociate(nc *datamodel.NetworkContainerEntity, ncLock map[string]string) (bool, *api.JsonErr) {
	if err := cnsInterface.DeleteNC(&nc.Config, ncLock); err != nil {
		// we should only retry internal errors. all else should be marked as failed
		removeFromQueue := err.HTTPStatus != http.StatusInternalServerError
		if removeFromQueue {
			nc.Config.Status = datamodel.Failed
		}

		nc.Config.FailureDetail = *err
		if persistErr := datamodel.NcRepo.PersistNetworkContainerData(nc); persistErr != nil {
			return false, persistErr
		}
		return removeFromQueue, err
	}

	logger.Debugf("[CNS] DisAssociated NC %s from node %s", nc.Config.NetworkContainerID, nc.Config.GetAssociationInfo().NodeID)

	return false, transitionNCStateAndLog(nc, datamodel.DisAssociate, datamodel.UnReserve, datamodel.Pending)
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

	var (
		errCode     = "NoError"
		errHttpCode = "NoError"
	)

	if err != nil {
		errCode = err.Code.String()
		errHttpCode = http.StatusText(err.HTTPStatus)
	}

	metric.CustomDimensions[aiwrapper.NCOperationErrorCodeStr] = errCode
	metric.CustomDimensions[aiwrapper.NCOperationErrorHttpCodeStr] = errHttpCode
	go aiwrapper.SendMetric(metric)

	loggershim.LS.Debugf("[DNCOperationStatus] OperationName: %s, NetworkContainerId: %s, Status: %s, Err:%+v",
		ncState, repoNCConfig.NetworkContainerID, status, err)
}

func publishNetworkContainerWithPubSubProxy(
	ncGoalState *datamodel.NCGoalState,
	nc *datamodel.NetworkContainerEntity,
	sessionInfo map[string]string,
	stateTransitionDateTime time.Time) (bool, *api.JsonErr) {
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

	if api.IsBareMetalContainer(nc.Config.GetAssociationInfo().ContainerType) {
		publishError = pubsubproxy.PublishBaremetalNetworkContainer(ncGoalState, &nc.Config)
	} else {
		publishError = pubsubproxy.PublishNetworkContainer(publishNcRequest)
	}

	// Reacquire the session locks after making call to PubSubProxy
	sessionManager.BeginWriteSession(sessionInfo)

	if publishError != nil {
		logger.Errorf("publishNetworkContainerWithPubSubProxy failed for NC: %d, Error: %v", publishNcRequest.NcID, publishError)
		if publishError.HTTPStatus == http.StatusInternalServerError {
			return false, publishError
		}
	}

	// Retrieve the latest NC configuration as the current copy of the configuration
	// could have been updated while pubsubproxy processed the NC publish.
	retrievedRepoNCData, ncRetrievalErr := datamodel.NcRepo.TryQueryNetworkContainer(ncGoalState.NetworkID, ncGoalState.NetworkContainerID)
	if ncRetrievalErr != nil {
		return false, api.InternalServerErrorf(api.PublishingFailed, "could not publish NC: %s. error: %s",
			ncGoalState.NetworkContainerID, ncRetrievalErr.String())
	}

	if retrievedRepoNCData.RowKey == "" {
		return true, api.BadRequestf(api.PublishingFailed, "could not publish NC: %s as NC data is not found", ncGoalState.NetworkContainerID)
	}

	// Copy over the delete flag as NC could have been deleted while calling CNS / NMAgent for publish.
	nc.Config.MarkedForDelete = retrievedRepoNCData.Config.MarkedForDelete

	// If the NC is marked for deletion, copy the state as it could have been changed too
	if nc.Config.MarkedForDelete {
		nc.Config.State = retrievedRepoNCData.Config.State
	}

	if publishError != nil && publishError.HTTPStatus == http.StatusUnauthorized {
		// If the response code is authentication failure, retry the request for the preset duration.
		// If authentication failure is still seen fail the request
		// This workaround is needed because sometimes there is a latency in sending the new authentication token
		// to pubsub. This workaround hides that till the latency is addressed.
		// If the NC gets marked for delete while calling CNS / NMAgent, in case of error that request needs to be
		// marked as retriable error to avoid losing it and delete NC getting stuck
		if (time.Since(stateTransitionDateTime).Seconds() < unauthorizedErrRetryDurationSec) || nc.Config.MarkedForDelete {
			return false, api.StatusUnauthorizedf(api.ProcessingPending, "Create NC %s pending. Response code: %d, time elapsed: %.0f sec",
				nc.Config.NetworkContainerID, publishError.HTTPStatus, time.Since(stateTransitionDateTime).Seconds())
		}

		// Mark the request as non-retriable if it fails for retry duration
		nc.Config.Status = datamodel.Failed
		failureDetail := api.StatusUnauthorizedf(api.PublishingFailed,
			"Failed to publish NC: %s with status code: %v. Marking as non-retryable", nc.Config.NetworkContainerID, publishError.HTTPStatus)
		nc.Config.FailureDetail = *failureDetail

		return true, datamodel.NcRepo.PersistNetworkContainerData(nc)
	}

	return false, transitionNCStateAndLog(nc, datamodel.Publish, datamodel.PrepareSecondaryIpCleanup, datamodel.Pending)
}

func processNetworkContainerPublish(
	nc *datamodel.NetworkContainerEntity,
	sessionInfo map[string]string,
	stateTransitionDateTime time.Time,
	createdDateTime time.Time) (bool, *api.JsonErr) {
	sessionInfoRead := map[string]string{
		datamodel.ReservationRepositoryName: nc.Config.ReservationID,
		datamodel.NetworkRepositoryName:     nc.Config.NetworkID,
		datamodel.NodeRepositoryName:        nc.Config.GetAssociationInfo().NodeID,
	}
	sessionManager.BeginReadSession(sessionInfoRead)
	ncGoalState, isRetriableError, ncGoalStateErr := datamodel.GetNCGoalState(&nc.Config)
	sessionManager.EndReadSession(sessionInfoRead)

	if ncGoalStateErr != nil {
		if isRetriableError {
			return false, ncGoalStateErr
		}

		nc.Config.Status = datamodel.Failed
		nc.Config.FailureDetail = *ncGoalStateErr
		return true, datamodel.NcRepo.PersistNetworkContainerData(nc)
	}

	if config.PubSubProxySettings.EnablePubSubProxy {
		return publishNetworkContainerWithPubSubProxy(ncGoalState, nc, sessionInfo, stateTransitionDateTime)
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
			nc.Config.Status = datamodel.Failed
			nc.Config.FailureDetail = api.JsonErr{
				Code:       api.PublishingFailed,
				Text:       msg,
				HTTPStatus: http.StatusBadRequest}
			return true, datamodel.NcRepo.PersistNetworkContainerData(nc)
		}

		publishNCURL := fmt.Sprintf(
			config.CNSSettings.ConnectionStrings.PublishNetworkContainerURLFmt,
			ncGoalState.NodeInterfaceID)

		logger.Debugf("[CNS] Publish NC %s, body:%s ", ncGoalState.NetworkContainerID, ncRequest.String())
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
					nc.Config.NetworkContainerID,
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
					nc.Config.NetworkContainerID,
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
				nc.Config.NetworkContainerID,
				resp.StatusCode),
			HTTPStatus: http.StatusInternalServerError}
	}

	var createNetworkContainerResponse nmagentcontracts.CreateNetworkContainerResponse
	decodeRespErr := utils.DecodeResponse(resp, &createNetworkContainerResponse)
	if decodeRespErr != nil {
		msg := fmt.Sprintf("Failed to decode create NC %s response due to error %s.",
			nc.Config.NetworkContainerID,
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
			nc.Config.NetworkContainerID)
		return false, &api.JsonErr{
			Code:       api.ParseResponseCodeFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	// Retrieve the latest NC configuration as the current copy of the configuration
	// could have been updated while we were in the process of calling cns.
	retrievedRepoNCData, ncRetrievalErr := datamodel.NcRepo.TryQueryNetworkContainer(ncGoalState.NetworkID, ncGoalState.NetworkContainerID)
	if ncRetrievalErr != nil {
		return false, api.InternalServerErrorf(api.PublishingFailed, "could not publish NC: %s. error: %s",
			ncGoalState.NetworkContainerID, ncRetrievalErr.String())
	}

	if retrievedRepoNCData.RowKey == "" {
		return true, api.BadRequestf(api.PublishingFailed, "could not publish NC: %s as NC data is not found", ncGoalState.NetworkContainerID)
	}

	// Copy over the delete flag as NC could have been deleted while calling CNS / NMAgent for publish.
	nc.Config.MarkedForDelete = retrievedRepoNCData.Config.MarkedForDelete

	// If the NC is marked for deletion, copy the state as it could have been changed too
	if nc.Config.MarkedForDelete {
		nc.Config.State = retrievedRepoNCData.Config.State
	}

	if putNCResponseCode != http.StatusOK {
		// If the response code is authentication failure, retry the request for the preset duration.
		// If authentication failure is still seen fail the request
		// This workaround is needed because sometimes there is a latency in sending the new authentication token
		// to pubsub. This workaround hides that till the latency is addressed.
		// If the NC gets marked for delete while calling CNS / NMAgent, in case of error that request needs to be
		// marked as retriable error to avoid losing it and delete NC getting stuck
		if putNCResponseCode == http.StatusUnauthorized {
			if (time.Since(stateTransitionDateTime).Seconds() < unauthorizedErrRetryDurationSec) || nc.Config.MarkedForDelete {
				return false, api.StatusUnauthorizedf(api.ProcessingPending, "Create NC %s pending. Response code: %d, time elapsed: %.0f sec",
					nc.Config.NetworkContainerID, putNCResponseCode, time.Since(stateTransitionDateTime).Seconds())
			}

			// Mark the request as non-retryable if it fails for retry duration
			failureDetail := api.StatusUnauthorizedf(api.PublishingFailed,
				"Failed to publish NC: %s with status code: %v. Marking as non-retryable", nc.Config.NetworkContainerID, putNCResponseCode)
			nc.Config.Status = datamodel.Failed
			nc.Config.FailureDetail = *failureDetail

			return true, datamodel.NcRepo.PersistNetworkContainerData(nc)
		}

		// Retry if NMAgent error response code is anything other than http.StatusUnauthorized
		msg := fmt.Sprintf("Failed to publish NC: %s with status code: %v. Marking as retryable.",
			nc.Config.NetworkContainerID, putNCResponseCode)

		return false, &api.JsonErr{
			Code:       api.PublishingFailed,
			Text:       msg,
			HTTPStatus: int(putNCResponseCode)}
	}

	// todo: we have to log program completion before transitioning state, since program can be transitioned to
	//   from refresh but we don't want to log completion in those cases
	nc.Config.LogStateCompletion(datamodel.Program)
	transitionErr := transitionNCStateAndLog(nc, datamodel.Publish, datamodel.Program, datamodel.Pending)

	creationTimeMs := time.Since(createdDateTime).Milliseconds()
	sendNCMetric(aiwrapper.NCProcessTimeMetric, float64(creationTimeMs), aiwrapper.Succeeded, "Container"+datamodel.Program.String(), &nc.Config, nil)
	aiwrapper.LogNCSnapsnot(ncGoalState.NetworkContainerID, ncGoalState.NetworkID, ncGoalState.BaseSubnetName, ncGoalState.AssociatedNodeID)

	return false, transitionErr
}

func unpublishNetworkContainerWithPubSubProxy(
	ncGoalState *datamodel.NCGoalState,
	nc *datamodel.NetworkContainerEntity,
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

	if api.IsBareMetalContainer(nc.Config.GetAssociationInfo().ContainerType) {
		unpublishErr = pubsubproxy.UnpublishBaremetalNetworkContainer(ncGoalState, &nc.Config)
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
						nc.Config.NetworkContainerID,
						unpublishErr.HTTPStatus,
						time.Since(stateTransitionDateTime).Seconds())
					return false, &api.JsonErr{
						Code:       api.ProcessingPending,
						Text:       msg,
						HTTPStatus: http.StatusUnauthorized}
				}

				// Mark the request as non-retriable if it fails for retry duration
				msg := fmt.Sprintf("Failed to unpublish NC: %s with status code: %v. Marking as non retriable.",
					nc.Config.NetworkContainerID, unpublishErr.HTTPStatus)
				nc.Config.Status = datamodel.Failed
				nc.Config.FailureDetail = api.JsonErr{
					Code:       api.UnpublishingFailed,
					Text:       msg,
					HTTPStatus: http.StatusUnauthorized}
				return true, datamodel.NcRepo.PersistNetworkContainerData(nc)
			}

			// Retry if error response code is anything other than http.StatusUnauthorized
			msg := fmt.Sprintf("Failed to unpublish NC: %s with status code: %v. Marking as retriable.",
				nc.Config.NetworkContainerID, unpublishErr.HTTPStatus)

			return false, &api.JsonErr{
				Code:       api.UnpublishingFailed,
				Text:       msg,
				HTTPStatus: unpublishErr.HTTPStatus}
		}
	}

	return false, transitionNCStateAndLog(nc, datamodel.UnPublish, datamodel.UnReserve, datamodel.Pending)
}

func processNetworkContainerUnPublish(
	nc *datamodel.NetworkContainerEntity,
	sessionInfo map[string]string,
	stateTransitionDateTime time.Time) (bool, *api.JsonErr) {
	sessionInfoRead := map[string]string{
		datamodel.ReservationRepositoryName: nc.Config.ReservationID,
		datamodel.NetworkRepositoryName:     nc.Config.NetworkID,
		datamodel.NodeRepositoryName:        nc.Config.GetAssociationInfo().NodeID,
	}
	sessionManager.BeginReadSession(sessionInfoRead)
	ncGoalState, isRetriableError, ncGoalStateErr := datamodel.GetNCGoalState(&nc.Config)
	sessionManager.EndReadSession(sessionInfoRead)

	if ncGoalStateErr != nil {
		if isRetriableError {
			return false, ncGoalStateErr
		}

		nc.Config.Status = datamodel.Failed
		nc.Config.FailureDetail = *ncGoalStateErr
		return true, datamodel.NcRepo.PersistNetworkContainerData(nc)
	}

	if config.PubSubProxySettings.EnablePubSubProxy {
		return unpublishNetworkContainerWithPubSubProxy(ncGoalState, nc, sessionInfo, stateTransitionDateTime)
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
					nc.Config.NetworkContainerID,
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
					nc.Config.NetworkContainerID,
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
				nc.Config.NetworkContainerID,
				respErr.Error()),
			HTTPStatus: http.StatusInternalServerError}
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, &api.JsonErr{
			Code: api.UnpublishingFailed,
			Text: fmt.Sprintf("Failed to unpublish NC %s with error code %d",
				nc.Config.NetworkContainerID,
				resp.StatusCode),
			HTTPStatus: http.StatusInternalServerError}
	}

	var deleteNCResponse nmagentcontracts.DeleteNetworkContainerResponse
	decodeRespErr := utils.DecodeResponse(resp, &deleteNCResponse)
	if decodeRespErr != nil {
		msg := fmt.Sprintf("Failed to decode delete NC %s response due to error %s",
			nc.Config.NetworkContainerID,
			decodeRespErr.Error())
		return false, &api.JsonErr{
			Code:       api.DecodeResponseFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	logger.Debugf("[NMAgent][Response] %s for NC: %s, unpublishViaCNS: %t", deleteNCResponse.String(),
		nc.Config.NetworkContainerID, unpublishViaCNS)

	deleteNCResponseCode, parseErr := strconv.ParseInt(deleteNCResponse.ResponseCode, 10, 64)
	if parseErr != nil {
		msg := fmt.Sprintf(
			"Failed to convert response code %s  to integer when parsing delete NC response for NC %s, Marking this error as retriable.",
			deleteNCResponse.ResponseCode, nc.Config.NetworkContainerID)
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
					nc.Config.NetworkContainerID, deleteNCResponseCode, time.Since(stateTransitionDateTime).Seconds())
				return false, &api.JsonErr{
					Code:       api.ProcessingPending,
					Text:       msg,
					HTTPStatus: http.StatusUnauthorized}
			}

			// Mark the request as non-retriable if it fails for retry duration
			msg := fmt.Sprintf("Failed to unpublish NC: %s with status code: %v. Marking as non retriable.",
				nc.Config.NetworkContainerID, deleteNCResponseCode)
			nc.Config.Status = datamodel.Failed
			nc.Config.FailureDetail = api.JsonErr{
				Code:       api.UnpublishingFailed,
				Text:       msg,
				HTTPStatus: http.StatusUnauthorized}
			return true, datamodel.NcRepo.PersistNetworkContainerData(nc)
		}

		// Retry if NMAgent error response code is anything other than http.StatusUnauthorized
		msg := fmt.Sprintf("Failed to unpublish NC: %s with status code: %v. Marking as retriable.",
			nc.Config.NetworkContainerID, deleteNCResponseCode)

		return false, &api.JsonErr{
			Code:       api.UnpublishingFailed,
			Text:       msg,
			HTTPStatus: int(deleteNCResponseCode)}
	}

	return false, transitionNCStateAndLog(nc, datamodel.UnPublish, datamodel.DisAssociate, datamodel.Pending)
}

// todo: this could be removed, since all it does is add an extra db operation to change pending to completed.
//   states that transition to program should just mark status as completed.
func processNetworkContainerProgram(nc *datamodel.NetworkContainerEntity) (bool, *api.JsonErr) {
	return true, transitionNCState(nc, datamodel.Program, datamodel.Completed)
}

func processReservationSetWithKey(
	reservationSetKey string) (bool, *api.JsonErr) {
	var (
		networkID        string
		reservationSetID string
		parseErr         error
	)

	if networkID, parseErr = datamodel.GetNetworkIDFromReservationSetKey(reservationSetKey); parseErr != nil {
		loggershim.LS.Errorf("could not get networkID from reservation set key: %s. Error: %v", reservationSetKey, parseErr)
		return removeFromPendingQ, nil
	}

	if reservationSetID, parseErr = datamodel.GetSetIDFromReservationSetKey(reservationSetKey); parseErr != nil {
		loggershim.LS.Errorf("could not get reservationSetID from reservation set key: %s. Error: %v", reservationSetKey, parseErr)
		return removeFromPendingQ, nil
	}

	reservationSet, err := datamodel.IpReservationSetRepo.TryQueryIPReservationSet(networkID, reservationSetID)
	if err != nil {
		return dontRemoveFromPendingQ, err
	}

	if reservationSet == nil {
		loggershim.LS.Errorf("could not find reservation set: %s for processing. Removing from pending queue", reservationSetID)
		return removeFromPendingQ, nil
	}

	if reservationSet.MarkedForDelete {
		loggershim.LS.Debugf("reservation set: %s is marked for deletion. Removing from pending queue", reservationSetID)
		return removeFromPendingQ, nil
	}

	existingReservations, err := datamodel.IpReservationRepo.TryQueryAllReservationsInSet(networkID, reservationSetID)
	if err != nil {
		return dontRemoveFromPendingQ, err
	}

	// Reservationset will be processed until there are no pending reservations in the set i.e. reservationset processing
	// will be be completed when all the reservations in the set change their status to either Failed or Completed.
	// The reservationset will be updated at this point with either Failed or Completed status
	var (
		successfulReservations []string
		failedReservations     []string
	)
	for _, reservation := range existingReservations {
		switch reservation.Configuration.Status {
		case datamodel.Failed:
			failedReservations = append(failedReservations, reservation.Configuration.ReservationID)

		case datamodel.Completed:
			successfulReservations = append(successfulReservations, reservation.Configuration.ReservationID)

		default:
			loggershim.LS.Debugf("reservation set: %s processing is in progress", reservationSetID)
			return dontRemoveFromPendingQ, nil
		}
	}

	processedReservationCount := len(successfulReservations) + len(failedReservations)
	if processedReservationCount != len(existingReservations) {
		// This code path should never be taken. Log the error. todo: add the alerting.
		err := fmt.Errorf("mismatch between processed reservation count: %d and total number of reservations: %d",
			processedReservationCount, len(existingReservations))
		loggershim.LS.Errorf(err.Error())
		return dontRemoveFromPendingQ, api.InternalServerErrorf(api.StatusInternalServerError, err.Error())
	}

	reservationSet.SuccessfulReservationCount = len(successfulReservations)

	//todo: add setID in the reservation processing error logs
	if len(failedReservations) > 0 {
		loggershim.LS.Errorf("marking reservation set: %s processing as failed. Failed reservations: [%s]",
			reservationSetID, strings.Join(failedReservations, ","))
		// Update the reservation set status as failed
		reservationSet.Status = datamodel.Failed
		if err = datamodel.IpReservationSetRepo.PersistReservationSet(reservationSet); err != nil {
			return dontRemoveFromPendingQ, err
		}

		return removeFromPendingQ, nil
	}

	loggershim.LS.Debugf("reservation set: %s processing completed successfully. Successful reservations: [%s]",
		reservationSetID, strings.Join(successfulReservations, ","))

	// Update the reservation set status as completed
	reservationSet.Status = datamodel.Completed
	if err = datamodel.IpReservationSetRepo.PersistReservationSet(reservationSet); err != nil {
		return dontRemoveFromPendingQ, err
	}

	loggershim.LS.Debugf("successfully completed reservation set request: %s with total reservation count %d",
		reservationSetID, len(successfulReservations))
	return removeFromPendingQ, nil
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
		repoNCData                    datamodel.NetworkContainerEntity
	)

	defer func() {
		if repoNCData.RowKey != "" {
			if removeRequestFromPendingQueue && repoNCData.Config.Status == datamodel.Failed {
				processingTime := time.Since(createdDateTime).Milliseconds()
				var ncState string
				if datamodel.IsDeleteInProgress(repoNCData.Config.State) {
					ncState = "Container" + datamodel.Purge.String()
					logger.Errorf("NC deletion failed for NC: %s", datamodel.GetNCIdFromKey(networkContainerKey))
				} else {
					ncState = "Container" + datamodel.Program.String()
					logger.Errorf("NC creation failed for NC: %s", datamodel.GetNCIdFromKey(networkContainerKey))
				}
				sendNCMetric(aiwrapper.NCProcessTimeMetric, float64(processingTime), aiwrapper.Failed, ncState, &repoNCData.Config, err)
			}
		}
	}()

	// Continue advancing the NC state to completion while there is no error (err == nil) and
	// indication to remove the request from the pending queue (!removeRequestFromPendingQueue)
	for !removeRequestFromPendingQueue && err == nil {
		repoNCData, err = datamodel.NcRepo.TryQueryNetworkContainerByKey(networkContainerKey)

		if err != nil {
			return false, err
		}

		if repoNCData.RowKey == "" {
			logger.Errorf("Failed to find network container in repository for key: %s", networkContainerKey)
			return true, nil
		}

		// Check if NC has been marked for delete and if so ensure that we are in the correct deletion state.
		if repoNCData.Config.MarkedForDelete && !datamodel.IsDeleteInProgress(repoNCData.Config.State) {
			revertState, err := datamodel.GetRevertState(repoNCData.Config.State)
			if err != nil {
				logger.Errorf("Failed to get the deletion state for NC: %s, current state: %s",
					repoNCData.Config.NetworkContainerID, repoNCData.Config.State)
				return false, err
			}
			repoNCData.Config.State = revertState
		}

		if repoNCData.Config.Status == datamodel.Failed {
			return true, nil
		}

		ncState := repoNCData.Config.State

		switch repoNCData.Config.State {
		case datamodel.Prepare:
			removeRequestFromPendingQueue, err = preparer.PrepareNetworkContainer(&repoNCData)

		case datamodel.Reserve:
			if config.FeatureFlagSettings.EnableBatchAllocationProcessing {
				removeRequestFromPendingQueue, err = processNetworkContainerBatchReserve(&repoNCData)
			} else {
				removeRequestFromPendingQueue, err = processNetworkContainerReserve(&repoNCData)
			}

		case datamodel.AllocateLocalIP:

			// Local IP allocation is not needed for bare-metal containers.
			// In that case change the NC config state to AllocateMAC and return.
			if api.IsBareMetalContainer(repoNCData.Config.GetAssociationInfo().ContainerType) {
				removeRequestFromPendingQueue, err = false, transitionNCStateAndLog(&repoNCData, datamodel.AllocateLocalIP, datamodel.AllocateMAC, datamodel.Pending)
			} else {
				removeRequestFromPendingQueue, err = processAllocateLocalIP(&repoNCData)
			}

		case datamodel.AllocateMAC:
			removeRequestFromPendingQueue, err = macAllocator.ProcessAllocateMAC(&repoNCData)

		case datamodel.Associate:
			fallthrough

		case datamodel.RefreshAssociation:
			removeRequestFromPendingQueue, err = processNetworkContainerAssociate(&repoNCData, sessionInfo)

		case datamodel.Publish:
			removeRequestFromPendingQueue, err = processNetworkContainerPublish(&repoNCData, sessionInfo, stateTransitionDateTime, createdDateTime)

		case datamodel.PrepareSecondaryIpCleanup:
			removeRequestFromPendingQueue, err = processPrepareSecondaryIpCleanup(&repoNCData, createdDateTime)

		case datamodel.Program:
			removeRequestFromPendingQueue, err = processNetworkContainerProgram(&repoNCData)

		case datamodel.UnPublish:
			removeRequestFromPendingQueue, err = processNetworkContainerUnPublish(&repoNCData, sessionInfo, stateTransitionDateTime)

		case datamodel.DisAssociate:
			removeRequestFromPendingQueue, err = processNetworkContainerDisAssociate(&repoNCData, sessionInfo)

		case datamodel.UnReserve:
			removeRequestFromPendingQueue = false // there is no obvious case where un-reserving requires removing the item from the pending queue
			err = processNetworkContainerUnReserve(&repoNCData)

		case datamodel.Purge:
			removeRequestFromPendingQueue, err = processNetworkContainerPurge(&repoNCData, createdDateTime)

		default:
			logger.Errorf("NC: %s processing failed due to invalid state %s", repoNCData.Config.NetworkContainerID, repoNCData.Config.State)
			return true, nil
		}

		var processStatus string
		if !removeRequestFromPendingQueue && err == nil {
			processStatus = aiwrapper.Succeeded
		} else {
			processStatus = aiwrapper.Failed
		}

		// Record the time taken for NC state processing and send the NC process time metric
		recordNetworkContainerProcessingTime(&repoNCData, ncState, processStatus, createdDateTime, stateTransitionDateTime, err)

		if !removeRequestFromPendingQueue && err == nil {
			requestContext.LastRetriedDateTime = time.Time{}
			requestContext.RetryInterval = requestContextRetryInterval
			requestContext.StateTransitionDateTime = time.Now().UTC()

			// todo: return if error?
			if err := datamodel.AddRequestToPendingQueue(requestID, requestContext); err != nil {
				logger.Errorf("could not add %s id %s back to pending queue", requestContext.Type.String(), requestID, err)
			}

			// Update stateTransitionDateTime to be used during the successful state transition
			stateTransitionDateTime = requestContext.StateTransitionDateTime
		}
	}

	return removeRequestFromPendingQueue, err
}

// Record the time taken for NC state processing and send the NC process time metric
func recordNetworkContainerProcessingTime(
	nc *datamodel.NetworkContainerEntity,
	ncState datamodel.NetworkContainerState,
	processStatus string,
	createdDateTime, stateTransitionDateTime time.Time,
	err *api.JsonErr) {
	if ncState == datamodel.Program ||
		ncState == datamodel.Purge {
		// Program and purge times are calculated based on request create time. This provides
		// the total time taken to program (create) and delete the NC.
		utils.RecordTime(createdDateTime, "Container"+ncState.String(), nc.Config.NetworkContainerID)
	} else {
		utils.RecordTime(stateTransitionDateTime, "Container"+ncState.String(), nc.Config.NetworkContainerID)
	}

	// Upload state process time metric for all the states. Program and Purge times are calulcated
	// based on request create time and RefreshAssociation is excluded to avoid flooding the metric
	// data as every NC will periodically go through refreshAssociation
	if ncState != datamodel.Program &&
		ncState != datamodel.Purge &&
		ncState != datamodel.RefreshAssociation {
		processTimeMs := time.Since(stateTransitionDateTime).Milliseconds()
		sendNCMetric(aiwrapper.NCProcessTimeMetric,
			float64(processTimeMs), processStatus, "Container"+ncState.String(), &nc.Config, err)
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

			if !config.TelemetrySettings.DisableTelemetry {
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
			logger.Debugf("Processing request: %+v", rc)
			processRequest(rid, &rc)
			processRequest(rid, &rc)
			// Remove the request from the in-process map
			requestsMap.Delete(rid)
			goroutines <- struct{}{}
		}(requestID, requestContext, inProcessRequestsMap, concurrentGoroutines)
	}
}
