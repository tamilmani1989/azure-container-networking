package main

import (
	"dnc/allocationhelper"
	"dnc/api"
	"dnc/datamodel"
	"dnc/utils"
	"fmt"
	"net"
	"net/http"
)

const reservedIPCount = 4

func processIPReservationAllocation(networkID, reservationKey string) (bool, *api.JsonErr) {
	var markAllocationAsFailed bool
	var failureDetail *api.JsonErr
	var allocatedIP *net.IPNet
	var ipSubnet *net.IPNet
	var allocationsInSubnet, reservedAddresses map[string]*net.IPNet
	var nodeID string
	addressesToIgnore := make(map[string]*net.IPNet)

	repoReservationRequest, err := datamodel.TryQueryIPReservationByKey(reservationKey)
	if err != nil {
		return false, err
	}

	if repoReservationRequest == nil {
		return false, nil
	}

	logger.Debugf("Processing Reservation Request for NetworkID %s\n", repoReservationRequest.NetworkID)
	subnetConfiguration, err := datamodel.TryQuerySubnetConfiguration(
		repoReservationRequest.NetworkID, repoReservationRequest.AllocationRequest.SubnetName)

	if err != nil {
		return false, err
	}

	if subnetConfiguration == nil {
		markAllocationAsFailed = true
		failureDetail = &api.JsonErr{
			Code:       api.SubnetNotFound,
			HTTPStatus: http.StatusBadRequest,
			Text:       fmt.Sprintf("Failed to find subnet for allocation request %s", repoReservationRequest.AllocatedSubnet)}
		goto Cleanup
	}
	if len(repoReservationRequest.AllocatedSubnet) != 0 {
		logger.Debugf(
			"Reservation request %s is already processed and has an allocated IP Address:%s.\n",
			repoReservationRequest.AllocationRequest.String(),
			repoReservationRequest.AllocatedSubnet)
		return true, nil
	}

	logger.Debugf("Found NetworkID: %s, ReservationID: %s, IPConstraint: %s\n", repoReservationRequest.NetworkID, repoReservationRequest.ReservationID, repoReservationRequest.AllocationRequest.IPConstraint)
	allocationsInSubnet, err = getAllocatedAddressesInSubnet(
		repoReservationRequest.NetworkID,
		subnetConfiguration)
	if err != nil {
		markAllocationAsFailed = true
		failureDetail = err
		goto Cleanup
	}

	ipSubnet, _ = utils.ParseIPNet(subnetConfiguration.IPSubnet)
	reservedAddresses, _ = getReservedAddressesInSubnet(ipSubnet)

	for reservedSubnetString, reservedIPSubnet := range reservedAddresses {
		addressesToIgnore[reservedSubnetString] = reservedIPSubnet
	}

	if len(repoReservationRequest.AssociatedNCID) > 0 {
		networkContainerConfiguration, err := datamodel.TryQueryNetworkContainer(
			repoReservationRequest.NetworkID,
			repoReservationRequest.AssociatedNCID)
		if err != nil {
			return false, err
		}

		if networkContainerConfiguration == nil {
			markAllocationAsFailed = true
			failureDetail = &api.JsonErr{
				Code:       api.NetworkContainerNotFound,
				HTTPStatus: http.StatusBadRequest,
				Text:       fmt.Sprintf("Failed to find configuration for network container %s", repoReservationRequest.AssociatedNCID)}
			goto Cleanup
		}

		nodeID = networkContainerConfiguration.GetAssociationInfo().NodeID
	} else if len(repoReservationRequest.AllocationRequest.NodeConstraint) != 0 {
		nodeID = repoReservationRequest.AllocationRequest.NodeConstraint
	}

	if len(nodeID) > 0 {
		allocatedAddressesOnNode, _ := getAllocatedAddressesOnNode(nodeID)
		for allocatedNodeSubnetString, allocatedNodeSubnet := range allocatedAddressesOnNode {
			addressesToIgnore[allocatedNodeSubnetString] = allocatedNodeSubnet
		}
	}

	for allocatedSubnetString, allocatedIPSubnet := range allocationsInSubnet {
		addressesToIgnore[allocatedSubnetString] = allocatedIPSubnet
	}

	if repoReservationRequest.AllocationRequest.IPConstraint != "" {
		var ipSubnetConstraint *net.IPNet
		ipSubnetConstraint, err = allocationhelper.GetIPSubnetForIPAllocationConstraint(
			repoReservationRequest.AllocationRequest.IPConstraint)

		if err != nil {
			markAllocationAsFailed = true
			failureDetail = err
			goto Cleanup
		} else {
			var result bool
			result, err = allocationhelper.IsRequestedRangeAllocatable(ipSubnet, addressesToIgnore, ipSubnetConstraint)
			if result {
				allocatedIP = ipSubnetConstraint
				logger.Debugf("Assigned IP: %s\n", allocatedIP)
			} else {
				// logger.Errorf(
				// 	"Failed to process request %s due to error %s\n",
				// 	repoReservationRequest.AllocationRequest.String(),
				// 	err.String())
				markAllocationAsFailed = true
				failureDetail = err
				goto Cleanup
			}
		}
	} else {
		var allocationErr error
		allocatedIP, allocationErr = allocationhelper.AllocateSubnetFromNetworkSpace(ipSubnet, addressesToIgnore, 32)
		if allocationErr != nil {
			markAllocationAsFailed = true
			failureDetail = &api.JsonErr{
				Code:       api.AllocationFailed,
				HTTPStatus: http.StatusBadRequest,
				Text: fmt.Sprintf(
					"Failed to allocate address for request %s. Error:[%s]",
					repoReservationRequest.AllocationRequest.String(),
					allocationErr.Error())}
			goto Cleanup
		}

		logger.Debugf("Allocated IP: %s\n", allocatedIP)
	}

	if allocatedIP != nil {
		repoReservationRequest.Status = datamodel.Completed
		repoReservationRequest.AllocatedSubnet = allocatedIP.String()

		if len(repoReservationRequest.AllocationRequest.NodeConstraint) > 0 {
			err = datamodel.UpdateNodeConstrainedReservationSetByNodeID(
				repoReservationRequest.AllocationRequest.NodeConstraint,
				repoReservationRequest.NetworkID,
				repoReservationRequest.ReservationID,
				false)
			if err != nil {
				logger.Errorf(
					"Failed to associate reservation request %s to node %s due to error %s\n",
					repoReservationRequest.String(),
					repoReservationRequest.AllocationRequest.NodeConstraint,
					err.String())
				return false, err
			}
		}
		datamodel.UpdateSubnetReservationSet(
			repoReservationRequest.NetworkID,
			repoReservationRequest.ReservationID,
			repoReservationRequest.AllocatedSubnet,
			false,
			subnetConfiguration)
		if err != nil {
			logger.Errorf(
				"Failed to persist updating reservation request %s in subnet configuration due to error %s\n",
				repoReservationRequest.String(),
				err.String())
			return false, err
		}

		err = datamodel.PutIPReservationConfiguration(repoReservationRequest)
		if err != nil {
			logger.Errorf(
				"Failed to mark reservation request %s as succeded due to error %s\n",
				repoReservationRequest.String(),
				err.String())
			return false, err
		}
	} else {
		logger.Debugf(
			"Invalid state: Process reservation for request %s completed without allocated IPn\n",
			repoReservationRequest.AllocationRequest.String())
	}

Cleanup:

	if markAllocationAsFailed {
		repoReservationRequest.Status = datamodel.Failed
		repoReservationRequest.FailureDetail = *failureDetail
		logger.Errorf(
			"Failed to process reservation request :{%s} due to error :{%s}\n",
			repoReservationRequest.String(),
			failureDetail.String())
		err := datamodel.PutIPReservationConfiguration(repoReservationRequest)
		if err != nil {
			logger.Errorf("Failed to perists reservation request %s as failed\n", repoReservationRequest.String())
			return false, err
		}
	}

	return true, nil
}

func getReservedAddressesInSubnet(ipSubnet *net.IPNet) (map[string]*net.IPNet, error) {
	reservedIPs := make(map[string]*net.IPNet)

	startingIP := utils.IP2Int(ipSubnet.IP)
	for i := 0; i < reservedIPCount; i++ {
		reservedIPString := utils.Int2IP(startingIP+uint32(i)).String() + utils.PrefixDelimiter + utils.IPV4DefaultPrefix
		reservedIPSubnet, _ := utils.ParseIPNet(reservedIPString)
		reservedIPs[reservedIPString] = reservedIPSubnet
	}
	var broadCastIP net.IP
	broadCastIP = allocationhelper.GetLastAdressInSubnet(ipSubnet)
	logger.Debugf("Broadcast IP: %s\n", broadCastIP)
	broadcastIPString := broadCastIP.String() + utils.PrefixDelimiter + utils.IPV4DefaultPrefix
	broadcastIPSubnet, _ := utils.ParseIPNet(broadcastIPString)
	reservedIPs[broadcastIPString] = broadcastIPSubnet
	return reservedIPs, nil
}

func getAllocatedAddressesInSubnet(networkID string, subnetConfiguration *datamodel.SubnetConfiguration) (map[string]*net.IPNet, *api.JsonErr) {
	allocatedAddresses := make(map[string]*net.IPNet)
	for reservationID := range subnetConfiguration.Reservations {
		logger.Debugf("Processing virtual network: %s\n", subnetConfiguration.NetworkID)
		reservationConfiguration, err := datamodel.QueryIPReservation(subnetConfiguration.NetworkID, reservationID)
		if err != nil {
			logger.Errorf("Failed to retrieve reservation configuration %s Error: %s\n", reservationID, err.String())
			return nil, err
		}

		if reservationConfiguration.AllocatedSubnet == "" {
			continue
		}

		allocatedIPSubnet, _ := utils.ParseIPNet(reservationConfiguration.AllocatedSubnet)
		allocatedAddresses[reservationConfiguration.AllocatedSubnet] = allocatedIPSubnet
	}

	return allocatedAddresses, nil
}

func getAllocatedAddressesOnNode(nodeID string) (map[string]*net.IPNet, *api.JsonErr) {
	allocatedSubnets := make(map[string]*net.IPNet)
	nodeConfiguration, err := datamodel.QueryNodeConfiguration(nodeID)

	if err != nil {
		return nil, err
	}

	if nodeConfiguration.ConstrainedReservations != nil {
		for reservationID, networkID := range nodeConfiguration.ConstrainedReservations {
			logger.Debugf("Processing virtual network %s\n", networkID)
			reservationConfiguration, err := datamodel.TryQueryIPReservation(networkID, reservationID)
			if err != nil {
				return nil, err
			}

			if reservationConfiguration == nil {
				logger.Debugf(
					"Configuration for reservation %s in network %s not found.\n",
					networkID,
					reservationID)
				continue
			}

			if reservationConfiguration.AllocatedSubnet != "" {
				allocatedIPSubnet, _ := utils.ParseIPNet(reservationConfiguration.AllocatedSubnet)
				allocatedSubnets[reservationConfiguration.AllocatedSubnet] = allocatedIPSubnet
			}
		}
	}
	if nodeConfiguration.DeployedNetworkContainerMap != nil {
		for networkID, ncMap := range nodeConfiguration.DeployedNetworkContainerMap {
			for ncKey, reservationID := range ncMap {
				logger.Debugf("Processing virtual network %s NC: %s ReservationID: %s\n", networkID, ncKey, reservationID)
				reservationConfiguration, err := datamodel.TryQueryIPReservation(networkID, reservationID)
				if err != nil {
					return nil, err
				}
				if reservationConfiguration == nil {
					logger.Debugf(
						"Configuration for reservation %s in network %s not found.\n",
						networkID,
						reservationID)
					continue
				}
				if reservationConfiguration.AllocatedSubnet != "" {
					allocatedIPSubnet, _ := utils.ParseIPNet(reservationConfiguration.AllocatedSubnet)
					allocatedSubnets[reservationConfiguration.AllocatedSubnet] = allocatedIPSubnet
				}
			}
		}
	}

	for ipIndex := range nodeConfiguration.NodeInfo.IPAddresses {
		nodeIPSubnetString := nodeConfiguration.NodeInfo.IPAddresses[ipIndex] + utils.PrefixDelimiter + utils.IPV4DefaultPrefix
		nodeIPSubnet, _ := utils.ParseIPNet(nodeIPSubnetString)
		allocatedSubnets[nodeIPSubnetString] = nodeIPSubnet
	}

	for subnetString := range allocatedSubnets {
		logger.Debugf("%s\n", subnetString)
	}

	return allocatedSubnets, nil
}
