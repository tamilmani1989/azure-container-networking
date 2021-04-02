package networkcontainer

import (
	"dnc/allocationhelper"
	"dnc/api"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/utils"
	"fmt"
	"net"
	"time"
)

// scanSubnetStrategy determines allocations by reading all currently existing allocation records and determining
// what is currently free or in the process of deletion in order to scale up and down.
type scanSubnetStrategy struct {
	allocator          *IPAllocator
	subnetInfo         subnetInfo
	logger             loggershim.LoggerInterface
	pubsubProxyEnabled bool
}

type batchAllocationResult struct {
	completed []datamodel.ReservationConfiguration
	failed    []datamodel.ReservationConfiguration
	rowBatch  map[string]map[string]interface{}
}

func (r *batchAllocationResult) reservationComplete(res datamodel.ReservationConfiguration) {
	res.Status = datamodel.Completed
	r.completed = append(r.completed, res)
}

func (r *batchAllocationResult) reservationFailed(res datamodel.ReservationConfiguration, err api.JsonErr) {
	res.Status = datamodel.Failed
	res.FailureDetail = err
	r.failed = append(r.failed, res)
}

func (s *scanSubnetStrategy) logf(format string, args ...interface{}) {
	s.logger.Debugf("[scan subnet strategy] vnet %s subnet %s - %s", s.subnetInfo.vnetID, s.subnetInfo.subnetName, fmt.Sprintf(format, args...))
}

func (s *scanSubnetStrategy) logErrorf(format string, args ...interface{}) {
	s.logger.Errorf("[scan subnet strategy] vnet %s subnet %s - %s", s.subnetInfo.vnetID, s.subnetInfo.subnetName, fmt.Sprintf(format, args...))
}

// Delete or unlink the associated secondary ips and return the updated reservation config. Caller will own persisting the updated reservation configs.
// tbdAllocationIDs is a map of reservation ID to a list of allocations IDs that should be removed from the reservation's stored configuration.
func (s *scanSubnetStrategy) scaleDown(reservations []datamodel.ReservationConfiguration, tbdAllocationIDs map[string][]string) ([]datamodel.ReservationConfiguration, error) {
	var allocIDs []string
	allocationIDToNCID := make(map[string]string)
	for i := 0; i < len(reservations); i++ {
		tbdIDs := tbdAllocationIDs[reservations[i].ReservationID]

		for _, allocID := range tbdIDs {
			allocIDs = append(allocIDs, allocID)
			allocationIDToNCID[allocID] = reservations[i].AssociatedNCID
			delete(reservations[i].SecondaryAllocations, allocID)
		}
	}

	// get all existing records that need to be deleted/updated
	tbdEntities, err := s.allocator.Allocations.QueryIPAllocationsByAllocationID(s.subnetInfo.vnetID, s.subnetInfo.subnetName, allocIDs)
	if err != nil {
		return nil, err
	}

	if len(tbdEntities) == 0 {
		s.logf("the following allocation ids were not found in allocation table: %+v", allocIDs)
		return reservations, nil
	}

	if s.pubsubProxyEnabled {
		// don't delete the allocations, just mark them with a tbd owner to be handled by cleanup monitor
		var updatedTBDEntities []datamodel.IPAllocationEntity
		for _, tbdEntity := range tbdEntities {
			ncID, ok := allocationIDToNCID[tbdEntity.AllocationID]
			if !ok || ncID == "" {
				s.logErrorf("scaleDown: no nc id found for allocation id %s, reservation key %s", tbdEntity.AllocationID, tbdEntity.IPAllocation)
				continue
			}

			tbdEntity.ToBeDeletedOwner = datamodel.GetNCKey(s.subnetInfo.vnetID, ncID)
			updatedTBDEntities = append(updatedTBDEntities, tbdEntity)
		}

		if err := s.allocator.UpdateAllocations(s.subnetInfo.vnetID, s.subnetInfo.subnetName, updatedTBDEntities); err != nil {
			s.logErrorf("Failed to set ToBeDeletedOwner for associated allocIds, allocId Count %d, err %+v", len(allocIDs), err)
			return nil, err
		}

		return reservations, nil
	}

	rowKeys := make([]string, len(tbdEntities))
	for i, entity := range tbdEntities {
		rowKeys[i] = entity.RowKey
	}

	if err := s.allocator.DeleteAllocations(s.subnetInfo.vnetID, s.subnetInfo.subnetName, rowKeys); err != nil {
		return nil, err
	}

	return reservations, nil
}

func (s *scanSubnetStrategy) scaleUp(reservations []datamodel.ReservationConfiguration) ([]datamodel.ReservationConfiguration, error) {
	allocResult, err := s.allocate(reservations)
	if err != nil {
		s.logErrorf("could not get allocations from subnet table: %v", err)
		return nil, err
	}

	if len(allocResult.rowBatch) > 0 {
		s.logf("persisting %d allocations", len(allocResult.rowBatch))
		if err := s.allocator.saveAllocationBatch(s.subnetInfo.vnetID, s.subnetInfo.subnetName, allocResult.rowBatch); err != nil {
			s.logErrorf("could not save allocation batch: %v", err)
			return nil, err
		}
	}

	return append(allocResult.completed, allocResult.failed...), nil
}

func (s *scanSubnetStrategy) allocate(reservations []datamodel.ReservationConfiguration) (batchAllocationResult, error) {
	allocReadTime := time.Now()

	s.logf("getting allocation state")
	allocationState, err := s.allocator.getAllocationsFromSubnetTable(s.subnetInfo.vnetID, s.subnetInfo.subnetName)
	if err != nil {
		return batchAllocationResult{}, err
	}

	s.logf("fetched %d allocation rows in %s", len(allocationState), time.Since(allocReadTime).String())

	var result batchAllocationResult
	result.rowBatch = make(map[string]map[string]interface{})

	for i := 0; i < len(reservations); i++ {
		res := reservations[i]
		isComplete := reconcileReservation(&res, allocationState)

		if isComplete {
			// if complete, add to completed list and remove from reservations that still need to be processed
			result.reservationComplete(res)
			reservations = append(reservations[:i], reservations[i+1:]...)
			i--
			continue
		}

		// otherwise, add the reconciled state to process list
		reservations[i] = res
	}

	addressesInUse, addrErr := getAllAddressesInUse(s.subnetInfo.baseSubnet, allocationState)
	if addrErr != nil {
		return batchAllocationResult{}, addrErr
	}

	s.logf("%d addresses in use in %s", len(addressesInUse), s.subnetInfo.baseSubnet.String())

	var ipsInUse []net.IP
	for _, ipAddress := range addressesInUse {
		ipsInUse = append(ipsInUse, ipAddress)
	}

	s.logf("determining allocations")
	for _, reservation := range reservations {
		var primaryIP net.IP
		if reservation.AllocatedSubnet != "" {
			primIP, _, err := net.ParseCIDR(reservation.AllocatedSubnet)
			if err != nil {
				result.reservationFailed(reservation, *api.InternalServerError(api.StatusInternalServerError, err.Error()))
				continue
			}
			primaryIP = primIP
		} else {
			ips, err := findAllocatableIPs(s.subnetInfo.baseSubnet, ipsInUse, 1)
			if err != nil {
				result.reservationFailed(reservation, *api.InternalServerError(api.StatusInternalServerError, err.Error()))
				continue
			}
			primaryIP = ips[0]
		}

		if _, ok := addressesInUse[primaryIP.String()]; !ok {
			ipsInUse = append(ipsInUse, primaryIP)
			addressesInUse[primaryIP.String()] = primaryIP
		}

		reservationKey := datamodel.GetReservationKey(reservation.NetworkID, reservation.ReservationID)

		if reservation.AllocatedSubnet == "" {
			reservation.AllocatedSubnet = primaryIP.String() + "/32"

			rowKey := datamodel.EncodeIPAllocationKey(reservation.AllocatedSubnet)
			result.rowBatch[rowKey] = map[string]interface{}{datamodel.IPAllocationProperty: reservationKey, datamodel.AllocationIDProperty: reservation.ReservationID}
		}

		unallocated := make(map[string]struct{})
		for allocationID, ipAddress := range reservation.SecondaryAllocations {
			if ipAddress == "" {
				unallocated[allocationID] = struct{}{}
			}
		}

		if len(unallocated) == 0 {
			result.reservationComplete(reservation)
			continue
		}

		availableIPs, err := findAllocatableIPs(s.subnetInfo.baseSubnet, ipsInUse, len(unallocated))
		if err != nil {
			result.reservationFailed(reservation, *api.InternalServerError(api.StatusInternalServerError, err.Error()))
			continue
		}

		i := 0
		for allocationID := range unallocated {
			encodedSubnet := availableIPs[i].String() + utils.PrefixDelimiter + utils.IPV4DefaultPrefix

			rowKey := datamodel.EncodeIPAllocationKey(encodedSubnet)
			result.rowBatch[rowKey] = map[string]interface{}{datamodel.IPAllocationProperty: reservationKey, datamodel.AllocationIDProperty: allocationID}

			if _, ok := addressesInUse[availableIPs[i].String()]; !ok {
				ipsInUse = append(ipsInUse, availableIPs[i])
				addressesInUse[encodedSubnet] = availableIPs[i]
			}
			i++
			reservation.SecondaryAllocations[allocationID] = encodedSubnet
		}

		result.reservationComplete(reservation)
	}

	return result, nil
}

// getAllAddressesInUse returns all IP addresses in use in a base subnet, including azure reserved addresses.
// results are indexed in "x.x.x.x" format.
func getAllAddressesInUse(baseSubnet *net.IPNet, allocationState map[string]*net.IPNet) (map[string]net.IP, error) {
	addressesInUse := make(map[string]net.IP)
	for _, subnet := range allocationState {
		addressesInUse[subnet.IP.String()] = subnet.IP
	}

	azureReserved, err := allocationhelper.GetReservedAddressesInSubnet(baseSubnet)
	if err != nil {
		return nil, err
	}

	for _, reservedIPSubnet := range azureReserved {
		addressesInUse[reservedIPSubnet.IP.String()] = reservedIPSubnet.IP
	}

	return addressesInUse, nil
}
