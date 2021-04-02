package networkcontainer

import (
	"dnc/api"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"fmt"
)

// prepopulatedStrategy determines allocations by leveraging indexing from the datastore: if all available records
// for a subnet have already been created but not assigned to reservations, we can run efficient queries to determine
// what ip addresses are available to be allocated.
type prepopulatedStrategy struct {
	allocator          *IPAllocator
	subnetInfo         subnetInfo
	logger             loggershim.LoggerInterface
	pubsubProxyEnabled bool
}

func (t *prepopulatedStrategy) logf(format string, args ...interface{}) {
	t.logger.Debugf("[prepopulated strategy] vnet %s subnet %s - %s", t.subnetInfo.vnetID, t.subnetInfo.subnetName, fmt.Sprintf(format, args...))
}

func (t *prepopulatedStrategy) logErrorf(format string, args ...interface{}) {
	t.logger.Errorf("[prepopulated strategy] vnet %s subnet %s - %s", t.subnetInfo.vnetID, t.subnetInfo.subnetName, fmt.Sprintf(format, args...))
}

// todo: most of this logic is very similar to other strategy and should be in a shared place. refactor
func (t *prepopulatedStrategy) scaleDown(reservations []datamodel.ReservationConfiguration, tbdAllocationIDs map[string][]string) ([]datamodel.ReservationConfiguration, error) {
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

	// get all existing records that need to be cleared
	tbdEntities, err := t.allocator.Allocations.QueryIPAllocationsByAllocationID(t.subnetInfo.vnetID, t.subnetInfo.subnetName, allocIDs)
	if err != nil {
		return nil, err
	}

	if len(tbdEntities) == 0 {
		t.logf("the following allocation ids were not found in allocation table: %+v", allocIDs)
		return reservations, nil
	}

	if t.pubsubProxyEnabled {
		// don't delete the allocations, just mark them with a tbd owner to be handled by cleanup monitor
		var updatedTBDEntities []datamodel.IPAllocationEntity
		for _, tbdEntity := range tbdEntities {
			ncID, ok := allocationIDToNCID[tbdEntity.AllocationID]
			if !ok || ncID == "" {
				t.logErrorf("scaleDown: no nc id found for allocation id %s, reservation key %s", tbdEntity.AllocationID, tbdEntity.IPAllocation)
				continue
			}

			tbdEntity.ToBeDeletedOwner = datamodel.GetNCKey(t.subnetInfo.vnetID, ncID)
			updatedTBDEntities = append(updatedTBDEntities, tbdEntity)
		}

		if err := t.allocator.UpdateAllocations(t.subnetInfo.vnetID, t.subnetInfo.subnetName, updatedTBDEntities); err != nil {
			t.logErrorf("Failed to set ToBeDeletedOwner for associated allocIds, allocId Count %d, err %+v", len(allocIDs), err)
			return nil, err
		}

		return reservations, nil
	}

	// will retain etags
	for i := range tbdEntities {
		tbdEntities[i].IPAllocation = ""
		tbdEntities[i].AllocationID = ""
		tbdEntities[i].ToBeDeletedOwner = ""
	}

	if err := t.allocator.UpdateAllocations(t.subnetInfo.vnetID, t.subnetInfo.subnetName, tbdEntities); err != nil {
		return nil, err
	}

	return reservations, nil
}

// clearAllocations will clear all properties for the given allocation ids
func (t *prepopulatedStrategy) clearAllocations(networkID, subnetName string, allocationIDs []string) *api.JsonErr {
	entities, err := t.allocator.Allocations.QueryIPAllocationsByAllocationID(networkID, subnetName, allocationIDs)
	if err != nil {
		return err
	}

	if len(entities) == 0 {
		t.logf("the following allocation ids were not found in allocation table: %+v", allocationIDs)
		return nil
	}

	// will retain etags
	for i := range entities {
		entities[i].IPAllocation = ""
		entities[i].AllocationID = ""
		entities[i].ToBeDeletedOwner = ""
	}

	return t.allocator.UpdateAllocations(networkID, subnetName, entities)
}

func (t *prepopulatedStrategy) scaleUp(reservations []datamodel.ReservationConfiguration) ([]datamodel.ReservationConfiguration, error) {
	allocResult, err := t.allocateFromPrepopulated(reservations)
	if err != nil {
		return nil, err
	}

	if len(allocResult.entitiesToPersist) > 0 {
		t.logf("claiming %d allocations", len(allocResult.entitiesToPersist))

		if err := t.allocator.UpdateAllocations(t.subnetInfo.vnetID, t.subnetInfo.subnetName, allocResult.entitiesToPersist); err != nil {
			t.logErrorf("could not update allocation entities: %v", err)
			return nil, err
		}
	}

	return append(allocResult.completed, allocResult.failed...), nil
}

type batchEntityResult struct {
	completed         []datamodel.ReservationConfiguration
	failed            []datamodel.ReservationConfiguration
	entitiesToPersist []datamodel.IPAllocationEntity
}

func (t *prepopulatedStrategy) allocateFromPrepopulated(reservations []datamodel.ReservationConfiguration) (batchEntityResult, error) {
	reservationIDs := make([]string, len(reservations))
	for i, reservation := range reservations {
		reservationIDs[i] = reservation.ReservationID
	}

	// fetch only allocations that are associated to input, in order to reconcile reservations with existing allocation state in db
	allocationState, err := t.allocator.getAllocationsForReservationIDs(t.subnetInfo.vnetID, t.subnetInfo.subnetName, reservationIDs)
	if err != nil {
		return batchEntityResult{}, err
	}

	// keep track of which reservations should be persisted as completed.
	// some reservations may be considered completed right after reconciliation,
	// or after all of its requested IPs have been populated.
	var (
		completedReservations []datamodel.ReservationConfiguration
		reservationsToProcess []datamodel.ReservationConfiguration
	)

	unallocatedCount := 0
	for _, res := range reservations {
		if isComplete := reconcileReservation(&res, allocationState); isComplete {
			res.Status = datamodel.Completed
			completedReservations = append(completedReservations, res)
			continue // no need to add to process list
		}

		// count missing allocations and add reconciled state to process list
		if res.AllocatedSubnet == "" {
			unallocatedCount++
		}

		for _, ipAddress := range res.SecondaryAllocations {
			if ipAddress == "" {
				unallocatedCount++
			}
		}

		reservationsToProcess = append(reservationsToProcess, res)
	}

	// query for any allocation records that have not been associated to reservations.
	// returned entity count may be below requested count. this is to make best effort to satisfy as many reservations in the batch.
	freeAllocs, err := t.allocator.Allocations.QueryUnreservedAllocations(t.subnetInfo.vnetID, t.subnetInfo.subnetName, unallocatedCount)
	if err != nil {
		return batchEntityResult{}, err
	}

	stack := allocationStack(freeAllocs)

	// update any unpopulated reservation IP addresses, and keep a list of the corresponding updated allocations.
	// if the reservation can only be processed partially, we consider it failed.
	var (
		updatedAllocations []datamodel.IPAllocationEntity
		failedReservations []datamodel.ReservationConfiguration
	)

	for _, res := range reservationsToProcess {
		reservationKey := datamodel.GetReservationKey(res.NetworkID, res.ReservationID)

		if res.AllocatedSubnet == "" {
			alloc, ok := stack.pop()
			if !ok {
				res.Status = datamodel.Failed
				res.FailureDetail = *api.BadRequest(api.SubnetFull, "not enough free IP addresses to satisfy request")
				failedReservations = append(failedReservations, res)
				continue
			}
			ipAddress := datamodel.DecodeIPAllocationKey(alloc.RowKey)
			res.AllocatedSubnet = ipAddress
			alloc.IPAllocation = reservationKey
			alloc.AllocationID = res.ReservationID
			updatedAllocations = append(updatedAllocations, alloc)
		}

		for allocationID, allocatedIP := range res.SecondaryAllocations {
			if allocatedIP != "" {
				continue
			}
			alloc, ok := stack.pop()
			if !ok {
				res.Status = datamodel.Failed
				res.FailureDetail = *api.BadRequest(api.SubnetFull, "not enough free IP addresses to satisfy request")
				failedReservations = append(failedReservations, res)
				break
			}
			ipAddress := datamodel.DecodeIPAllocationKey(alloc.RowKey)
			res.SecondaryAllocations[allocationID] = ipAddress
			alloc.IPAllocation = reservationKey
			alloc.AllocationID = allocationID
			updatedAllocations = append(updatedAllocations, alloc)
		}

		if res.Status != datamodel.Failed {
			res.Status = datamodel.Completed
			completedReservations = append(completedReservations, res)
		}
	}

	return batchEntityResult{
		completed:         completedReservations,
		failed:            failedReservations,
		entitiesToPersist: updatedAllocations,
	}, nil
}

type allocationStack []datamodel.IPAllocationEntity

func (a *allocationStack) pop() (datamodel.IPAllocationEntity, bool) {
	if len(*a) == 0 {
		return datamodel.IPAllocationEntity{}, false
	}
	tmp := *a
	last := tmp[len(tmp)-1]
	*a = tmp[:len(tmp)-1]
	return last, true
}
