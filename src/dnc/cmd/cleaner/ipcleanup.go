package main

import (
	"context"
	"dnc/api"
	"dnc/datamodel"
)

// This will delete the allocated ips created for this DNC partition
func DeleteAllocatedIpsForPartition(ctx context.Context, repo *partitionRepo) *api.JsonErr {
	Log(ctx).Debugf("DeleteAllocatedIpsForPartition: Delete pending ips from PendingResourceCleanup")
	// First run the cleanupMonitor to delete the ips associated to ToBeDeleted NCs
	err := deleteIpsFromPendingResourceCleanup(ctx, repo)
	if err != nil {
		return err
	}

	Log(ctx).Debugf("DeleteAllocatedIpsForPartition: Delete pending ips from Reservations")

	return deleteIpsFromReservationTable(ctx, repo)
}

// Delete the allocated ips associated with pendingResourceCleanup on this partition
func deleteIpsFromPendingResourceCleanup(ctx context.Context, repo *partitionRepo) *api.JsonErr {
	pendingResourcesCleanup, err := repo.pendingResourceCleanupRepo.GetAllPendingResourceCleanups()
	if err != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError, "Failed to get all resources pending cleanup with err: %v", err)
	}

	if len(pendingResourcesCleanup) == 0 {
		Log(ctx).Debugf("Nothing to clean in PendingResourceCleanup")
		return nil
	}

	for _, entity := range pendingResourcesCleanup {
		// only fetch pending jobs that belong to a specific NC
		networkid, err := datamodel.GetNetworkIDFromPendingResourceCleanupKey(entity.RowKey)
		if err != nil {
			return err
		}

		ncid, err := datamodel.GetNetworkContainerIDFromPendingResourceCleanupKey(entity.RowKey)
		if err != nil {
			return err
		}

		ncKey := datamodel.GetNCKey(networkid, ncid)
		err = removeIPAllocations(ctx, repo, ncKey, entity)
		if err != nil {
			return api.InternalServerErrorf(api.StatusInternalServerError,
				"deleteIpsFromPendingResourceCleanup: Failed to remove IP Allocations for "+
					"entity %s with error: %v", entity.RowKey, err)
		}
	}

	// Wipe PendingResourceCleanuptable
	return repo.pendingResourceCleanupRepo.DeleteAllRows()
}

// delete the allocated ips for a given PendingResourceCleanup resource
func removeIPAllocations(ctx context.Context, repo *partitionRepo, ncKey string, resourceCleanupEntity datamodel.PendingResourceCleanupEntity) *api.JsonErr {
	for ipAllocationsPartitionKey, cleanupAllocations := range resourceCleanupEntity.CleanupContext.ToBeDeletedIpAllocations {
		Log(ctx).Debugf("RemoveIPAllocations: Attempting to remove IP Allocations for PartitionKey %s and TobeDeletedOwner %s",
			ipAllocationsPartitionKey, ncKey)

		// get ip allocations with ToBeDeletedOwner matching the ncid
		allocationEntities, err := repo.ipAllocationRepo.QueryIPAllocationsByToBeDeletedOwner(ipAllocationsPartitionKey, ncKey)
		if err != nil {
			return err
		}

		// construct a map for faster queries based on RowKey (which is an IPAddress)
		ipAllocationsMap := make(map[string]datamodel.IPAllocationEntity, len(allocationEntities))
		for _, entity := range allocationEntities {
			ipAllocationsMap[entity.RowKey] = entity
		}

		// check to make sure that all IP's to be cleaned up have the matching ToBeDeletedOwner from the cleanup resource
		entitiesToBeCleanedUp := []datamodel.IPAllocationEntity{}
		for _, cleanupAllocation := range cleanupAllocations {
			if entity, exists := ipAllocationsMap[cleanupAllocation]; exists {
				// keep a slice of entities with their etags that exist with the ToBeDeletedOwner set as the pending cleanup NCID
				Log(ctx).Debugf("RemoveIPAllocations: Add IPAllocation for delete, PartitionKey %s, entity %v", ipAllocationsPartitionKey, entity)
				entitiesToBeCleanedUp = append(entitiesToBeCleanedUp, entity)
			} else {
				Log(ctx).Debugf("RemoveIPAllocations: Ignore delete, IPAllocation %s not found in IPAllocations table with PartitionKey %s and ToBeDeletedOwner %s", cleanupAllocation, ipAllocationsPartitionKey, ncKey)
			}
		}

		k, parseErr := datamodel.ParseIPAllocationPartitionKey(ipAllocationsPartitionKey)
		if parseErr != nil {
			return api.InternalServerError(api.StatusInternalServerError, parseErr.Error())
		}
		err = deleteIPAllocations(ctx, repo, k.Network, k.Subnet, entitiesToBeCleanedUp)
		if err != nil {
			return err
		}
	}

	return nil
}

// This will delete the allocated ips associated with Reservations on this partition
func deleteIpsFromReservationTable(ctx context.Context, repo *partitionRepo) *api.JsonErr {
	reservationsRow, err := repo.reservationRepo.TryQueryAllReservations()
	if err != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError, "Failed to get all resources pending cleanup with err: %v", err)
	}

	if len(reservationsRow) == 0 {
		Log(ctx).Debugf("RemoveIPAllocations: Nothing to clean in reservationsConfigs for partition")
		return nil
	}

	for _, row := range reservationsRow {
		// only fetch pending jobs that belong to a specific NC
		// fetch in IpAllocations for this ReservationId
		ipAllocationEntities, err := repo.ipAllocationRepo.QueryIPAllocationsByReservationKey(row.Configuration.NetworkID, row.Configuration.AllocationRequest.SubnetName, row.RowKey)
		if err != nil {
			return err
		}

		err = deleteIPAllocations(ctx, repo, row.Configuration.NetworkID, row.Configuration.AllocationRequest.SubnetName, ipAllocationEntities)
		if err != nil {
			return err
		}
	}

	// Wipe PendingResourceCleanuptable
	return repo.reservationRepo.DeleteAllRows()
}

// Batch delete all the ipAllocations for a given partition key (Vnet + Subnet)
func deleteIPAllocations(ctx context.Context, repo *partitionRepo, networkId string, subnetName string, entitiesToBeCleanedUp []datamodel.IPAllocationEntity) *api.JsonErr {
	if len(entitiesToBeCleanedUp) == 0 {
		Log(ctx).Debugf("DeleteAllocations: no allocations to delete for a given NetworkId %s, Subnet %s", networkId, subnetName)
		return nil
	}

	rowKeys := make([]string, len(entitiesToBeCleanedUp))
	for i, entity := range entitiesToBeCleanedUp {
		rowKeys[i] = entity.RowKey
	}

	//todo: This Batch logic should be inside ipallocationrepo
	var batches [][]string
	for len(rowKeys) > repo.MaxBatchSize {
		rowKeys, batches = rowKeys[repo.MaxBatchSize:], append(batches, rowKeys[0:repo.MaxBatchSize])
	}

	batches = append(batches, rowKeys)

	for i := 0; i < len(batches); i++ {
		if err := repo.ipAllocationRepo.DeleteIPAllocations(networkId, subnetName, batches[i]); err != nil {
			return err
		}
	}
	return nil
}
