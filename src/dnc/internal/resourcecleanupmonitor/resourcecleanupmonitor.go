package resourcecleanupmonitor

import (
	"context"
	"dnc/api"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/networkcontainer"
	"dnc/pubsubproxy"
	"fmt"
	"sync"
	"time"
)

type ResourceCleanupMonitor struct {
	PendingCleanupRepo *datamodel.PendingResourceCleanupRepo
	Logger             loggershim.LoggerInterface
	IsPrimary          bool
	IpAllocator        *networkcontainer.IPAllocator
}

const maxJobs = 50

func StartCleanupMonitor(monitor *ResourceCleanupMonitor, ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-ctx.Done():
			monitor.Logger.Debugf("Cleanup monitor shutdown")
			return
		case <-ticker.C:
			if monitor.IsPrimary {
				if err := monitor.pendingResourceCleanup(); err != nil {
					monitor.Logger.Errorf("ProcessCleanupResources failed with err: %v", err)
				}
			}
		}
	}
}

// pendingResourceCleanup checks which resources are pending cleanup in the PendingResourceCleanup table,
// and compares with the NcPendingPublish table to make sure there are no pending pubsubproxy jobs
// If there are no pending pubsub proxy jobs, the resources in the PendingResourceCleanup job are cleaned up
func (m *ResourceCleanupMonitor) pendingResourceCleanup() *api.JsonErr {
	pendingResourcesCleanup, err := m.PendingCleanupRepo.GetAllPendingResourceCleanups()
	if err != nil {
		return api.InternalServerError(api.StatusInternalServerError, fmt.Sprintf("Failed to get all resources pending cleanup with err: %v", err))
	}

	// limit the max number of jobs running at one time
	sem := make(chan struct{}, maxJobs)
	var wg sync.WaitGroup

	for _, cleanupResourceRow := range pendingResourcesCleanup {
		sem <- struct{}{}
		wg.Add(1)
		go func(entity datamodel.PendingResourceCleanupEntity, wg *sync.WaitGroup) {
			defer wg.Done()
			if err := m.processCleanupResource(entity); err != nil {
				m.Logger.Errorf("%s Failed to process ResourceCleanup with err %+v", logPrefix(entity), err)
			}
			<-sem
		}(cleanupResourceRow, &wg)
	}

	wg.Wait()
	return nil
}

func (m *ResourceCleanupMonitor) processCleanupResource(cleanupResourceRow datamodel.PendingResourceCleanupEntity) *api.JsonErr {
	if cleanupResourceRow.CleanupContext.Type == datamodel.IPCleanup {
		m.Logger.Debugf("%s Starting Processing ResourceCleanup", logPrefix(cleanupResourceRow))

		// only fetch pending jobs that belong to a specific NC
		networkid, err := datamodel.GetNetworkIDFromPendingResourceCleanupKey(cleanupResourceRow.RowKey)
		if err != nil {
			return err
		}

		ncid, err := datamodel.GetNetworkContainerIDFromPendingResourceCleanupKey(cleanupResourceRow.RowKey)
		if err != nil {
			return err
		}

		ncPendingPublish, err := pubsubproxy.GetPendingNCJobContexts(networkid, ncid)
		if err != nil && err.Code != api.NcPendingPublishEntryNotFound {
			return err
		}

		// check if there's a scheduled pubsub job matching the same NCID
		if len(ncPendingPublish) == 0 {
			ncKey := datamodel.GetNCKey(networkid, ncid)
			err := m.removeIPAllocations(ncKey, cleanupResourceRow)
			if err != nil {
				return api.InternalServerError(api.StatusInternalServerError, fmt.Sprintf("Failed to remove IP Allocations for NC ID %v with error: %v", cleanupResourceRow, err))
			}

			err = m.PendingCleanupRepo.DeleteCleanupRequest(cleanupResourceRow.RowKey, cleanupResourceRow.Etag)
			if err != nil {
				return api.InternalServerError(api.StatusInternalServerError, fmt.Sprintf("Failed to delete PendingResourceCleanup object with error: %v", err))
			}
		}
	}

	return nil
}

func (m *ResourceCleanupMonitor) removeIPAllocations(ncKey string, resourceCleanupEntity datamodel.PendingResourceCleanupEntity) *api.JsonErr {
	m.Logger.Debugf("%s Attempting to remove IP Allocations: [%+v]", logPrefix(resourceCleanupEntity), resourceCleanupEntity.CleanupContext.ToBeDeletedIpAllocations)
	for ipAllocationsKey, cleanupAllocations := range resourceCleanupEntity.CleanupContext.ToBeDeletedIpAllocations {
		subnetName, err := datamodel.GetSubnetFromIPAllocationsPartitionKey(ipAllocationsKey)
		if err != nil {
			return err
		}
		networkID, err := datamodel.GetNetworkIdFromIPAllocationsPartitionKey(ipAllocationsKey)
		if err != nil {
			return err
		}

		// get ip allocations with ToBeDeletedOwner matching the ncid
		allocationEntities, err := m.IpAllocator.GetIPAllocationsWithToBeDeletedOwner(networkID, subnetName, ncKey)
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
				entitiesToBeCleanedUp = append(entitiesToBeCleanedUp, entity)
			} else {
				m.Logger.Errorf(fmt.Sprintf("%s Allocation [%s] not found in IPAllocations table with NCKey [%s]", logPrefix(resourceCleanupEntity), cleanupAllocation, ncKey))
			}
		}

		// safe to delete allocations
		m.Logger.Debugf("%s Verified IP Allocations with correct ToBeDeletedOwner ID's set [%+v]", logPrefix(resourceCleanupEntity), entitiesToBeCleanedUp)
		err = m.IpAllocator.DeleteAllocationsWithEtag(entitiesToBeCleanedUp)
		if err != nil {
			return err
		}
	}

	return nil
}

func logPrefix(resource datamodel.PendingResourceCleanupEntity) string {
	return fmt.Sprintf("[Cleanup Resource: %s, Etag: %s]:", resource.RowKey, resource.Etag)
}
