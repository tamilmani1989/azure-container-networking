package main

import (
	"dnc/allocationhelper"
	"dnc/api"
	"dnc/configuration"
	"dnc/datamodel"
	"dnc/utils"
	"fmt"
	"net"
	"net/http"

	"github.com/Azure/azure-container-networking/cns"
)

const reservedIPCount = 4

func processIPReservationAllocationWithKey(reservationKey string) (bool, *api.JsonErr) {
	repoReservation, err := datamodel.TryQueryIPReservationByKey(reservationKey)
	if err != nil {
		return false, err
	}

	return processIPReservationAllocation(repoReservation)
}

func processIPReservationAllocation(request *datamodel.ReservationConfiguration) (bool, *api.JsonErr) {
	// if the parent reservation entry doesn't exist then it must have
	// been deleted via separate delete reservation/container request,
	// or if the subnet has already been allocated
	if request == nil || request.AllocatedSubnet != "" {
		return true, nil // don't add new pending work queue item, error
	}

	subnetConfig, err := datamodel.TryQuerySubnetConfiguration(request.NetworkID, request.AllocationRequest.SubnetName)
	if err != nil {
		return false, err
	}
	if subnetConfig == nil {
		err := &api.JsonErr{
			Code:       api.SubnetNotFound,
			HTTPStatus: http.StatusBadRequest,
			Text:       fmt.Sprintf("Failed to find subnet for allocation request %s", request.AllocatedSubnet),
		}
		return cleanupFailedAllocation(request, err)
	}

	reservationKey := datamodel.GetReservationKey(request.NetworkID, request.ReservationID)
	addressesToIgnore, allocatedIP, err := getAllocationsInSubnet(request.NetworkID, request.AllocationRequest.SubnetName, reservationKey, subnetConfig)
	if err != nil {
		return cleanupFailedAllocation(request, err)
	}
	// an ip had already been allocated for this reservation, so we can just skip to the completing steps
	if allocatedIP != nil {
		return completeAllocation(request, allocatedIP, subnetConfig)
	}

	baseSubnet, _ := utils.ParseIPNet(subnetConfig.IPSubnet)
	newIP, err := getAllocatableIP(request, baseSubnet, addressesToIgnore)
	if err != nil {
		return cleanupFailedAllocation(request, err)
	}
	if newIP == nil {
		err := &api.JsonErr{
			Code:       api.AllocationFailed,
			HTTPStatus: http.StatusBadRequest,
			Text:       fmt.Sprintf("Failed to allocate address for request %s. Subnet is full", request.AllocationRequest.String()),
		}
		return cleanupFailedAllocation(request, err)
	}

	if err := addIPAllocation(request.NetworkID, request.AllocationRequest.SubnetName, reservationKey, newIP); err != nil {
		return false, err
	}

	return completeAllocation(request, newIP, subnetConfig)
}

func getAllocatableIP(request *datamodel.ReservationConfiguration, baseSubnet *net.IPNet, addressesToIgnore map[string]*net.IPNet) (*net.IPNet, *api.JsonErr) {
	reservedAddresses := GetReservedAddressesInSubnet(baseSubnet)
	for reservedSubnetString, reservedIPSubnet := range reservedAddresses {
		addressesToIgnore[reservedSubnetString] = reservedIPSubnet
	}

	if request.AssociatedNCID != "" {
		ncConfig, err := datamodel.TryQueryNetworkContainer(request.NetworkID, request.AssociatedNCID)
		if err != nil {
			return nil, err
		}
		if ncConfig == nil {
			return nil, &api.JsonErr{
				Code:       api.NetworkContainerNotFound,
				HTTPStatus: http.StatusBadRequest,
				Text:       fmt.Sprintf("Failed to find configuration for network container %s", request.AssociatedNCID),
			}
		}

		nodeID := ncConfig.GetAssociationInfo().NodeID
		// This needs to be fixed for scenarios where we are using the reserve IP
		// API to dump allocations on a node and orchestrator is WebApps.
		if nodeID != "" && ncConfig.GetAssociationInfo().ContainerType == cns.WebApps {
			allocatedAddressesOnNode, err := getAllocatedAddressesOnNode(nodeID)
			if err != nil {
				return nil, err
			}
			for allocatedNodeSubnetString, allocatedNodeSubnet := range allocatedAddressesOnNode {
				addressesToIgnore[allocatedNodeSubnetString] = allocatedNodeSubnet
			}
		}
	}

	// Process reservations with IP constraint requesting from specific IP.
	if request.AllocationRequest.IPConstraint != "" {
		ipSubnet, err := allocationhelper.GetIPSubnetForIPAllocationConstraint(request.AllocationRequest.IPConstraint)
		if err != nil {
			return nil, err
		}
		_, err = allocationhelper.IsRequestedRangeAllocatable(baseSubnet, addressesToIgnore, ipSubnet)
		if err != nil {
			return nil, err
		}

		return ipSubnet, nil
	}

	ipSubnet, allocErr := allocationhelper.AllocateSubnetFromNetworkSpace(baseSubnet, addressesToIgnore, 32)
	if allocErr != nil {
		return nil, &api.JsonErr{
			Code:       api.AllocationFailed,
			HTTPStatus: http.StatusBadRequest,
			Text:       fmt.Sprintf("Failed to allocate address for request %s. Error:[%s]", request.AllocationRequest.String(), allocErr.Error()),
		}
	}
	return ipSubnet, nil
}

func completeAllocation(request *datamodel.ReservationConfiguration, allocatedIP *net.IPNet, subnetConfig *datamodel.SubnetConfiguration) (bool, *api.JsonErr) {
	request.Status = datamodel.Completed
	request.AllocatedSubnet = allocatedIP.String()

	if request.AllocationRequest.NodeConstraint != "" {
		err := datamodel.UpdateNodeConstrainedReservationSetByNodeID(request.AllocationRequest.NodeConstraint, request.NetworkID, request.ReservationID, false)
		if err != nil {
			err.Text = fmt.Sprintf("Failed to associate reservation request %s to node %s due to error %s", request.String(), request.AllocationRequest.NodeConstraint, err.String())
			return false, err
		}
	}

	if err := datamodel.UpdateSubnetReservationSet(request.NetworkID, request.ReservationID, request.AllocatedSubnet, false, subnetConfig); err != nil {
		err.Text = fmt.Sprintf("Failed to persist updating reservation request %s in subnet configuration due to error %s", request.String(), err.String())
		if err.Code == api.PropertyValueTooLargeStorageError {
			err := &api.JsonErr{
				Code:       api.FailedToUpdateSubnetWithReservationInfo,
				HTTPStatus: http.StatusBadRequest,
				Text:       err.Text}
			return cleanupFailedAllocation(request, err)
		}

		return false, err
	}

	if err := datamodel.PutIPReservationConfiguration(request); err != nil {
		err.Text = fmt.Sprintf("Failed to mark reservation request %s as succeded due to error %s", request.String(), err.String())
		return false, err
	}

	return true, nil
}

func cleanupFailedAllocation(req *datamodel.ReservationConfiguration, err *api.JsonErr) (bool, *api.JsonErr) {
	req.Status = datamodel.Failed
	req.FailureDetail = *err
	logger.Errorf("Failed to process reservation request {%s} due to error {%s}", req.String(), err.String())
	if err := datamodel.PutIPReservationConfiguration(req); err != nil {
		return false, err
	}
	return true, nil
}

func GetReservedAddressesInSubnet(ipSubnet *net.IPNet) map[string]*net.IPNet {
	reservedIPs := make(map[string]*net.IPNet)

	startingIP := utils.IP2Int(ipSubnet.IP)
	for i := 0; i < reservedIPCount; i++ {
		reservedIPString := utils.Int2IP(startingIP+uint32(i)).String() + utils.PrefixDelimiter + utils.IPV4DefaultPrefix
		reservedIPSubnet, _ := utils.ParseIPNet(reservedIPString)
		reservedIPs[reservedIPString] = reservedIPSubnet
	}
	var broadCastIP net.IP
	broadCastIP = allocationhelper.GetLastAdressInSubnet(ipSubnet)
	broadcastIPString := broadCastIP.String() + utils.PrefixDelimiter + utils.IPV4DefaultPrefix
	broadcastIPSubnet, _ := utils.ParseIPNet(broadcastIPString)
	reservedIPs[broadcastIPString] = broadcastIPSubnet
	return reservedIPs
}

func getAllocatedAddressesInSubnet(subnetConfiguration *datamodel.SubnetConfiguration) (map[string]*net.IPNet, *api.JsonErr) {
	allocatedAddresses := make(map[string]*net.IPNet)
	for reservationID := range subnetConfiguration.Reservations {
		reservationConfiguration, err := datamodel.QueryIPReservation(subnetConfiguration.NetworkID, reservationID)
		if err != nil {
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
			reservationConfiguration, err := datamodel.TryQueryIPReservation(networkID, reservationID)
			if err != nil {
				return nil, err
			}

			if reservationConfiguration == nil {
				logger.Debugf(
					"Configuration for reservation %s in network %s not found.",
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
			for _, reservationID := range ncMap {
				reservationConfiguration, err := datamodel.TryQueryIPReservation(networkID, reservationID)
				if err != nil {
					return nil, err
				}
				if reservationConfiguration == nil {
					logger.Debugf(
						"Configuration for reservation %s in network %s not found.",
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

	return allocatedSubnets, nil
}

func addIPAllocation(networkID, subnetName, reservationKey string, allocatedIP *net.IPNet) *api.JsonErr {
	if datamodel.IsFeatureSupported(configuration.SharedSubnetDelegation) {
		err := datamodel.AddIPAllocation(networkID, subnetName, datamodel.EncodeIPAllocationKey(allocatedIP.String()), reservationKey)
		if err != nil {
			return err
		}
	}

	return nil
}

func getAllocationsInSubnet(networkID, subnetName, reservationKey string, subnetConfiguration *datamodel.SubnetConfiguration) (
	map[string]*net.IPNet, *net.IPNet, *api.JsonErr) {
	var err *api.JsonErr
	var allocatedIP *net.IPNet
	allocationsInSubnet := make(map[string]*net.IPNet)

	if datamodel.IsFeatureSupported(configuration.SharedSubnetDelegation) {
		var hasMoreRows = true
		var context interface{}
		var ipAllocationsInSubnet map[string]interface{}
		for hasMoreRows {
			ipAllocationsInSubnet, hasMoreRows, context, err = datamodel.TryQueryAllIPAllocations(networkID, subnetName, context)
			if err != nil {
				return nil, nil, err
			}

			for savedIPAddress, savedReservationKey := range ipAllocationsInSubnet {
				ipAddressString := datamodel.DecodeIPAllocationKey(savedIPAddress)
				allocationsInSubnet[ipAddressString], _ = utils.ParseIPNet(ipAddressString)

				if reservationKey == savedReservationKey {
					allocatedIP, _ = allocationsInSubnet[ipAddressString]
				}
			}
		}
	} else {
		allocationsInSubnet, err = getAllocatedAddressesInSubnet(subnetConfiguration)
	}

	return allocationsInSubnet, allocatedIP, err
}
