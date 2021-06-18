package main

import (
	"dnc/api"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"strings"
	"time"
)

func processNodeSetCleanupRequest(nodeSetName string) (bool, *api.JsonErr) {
	logger := loggershim.WithPrefixf(logger, "[node set cleanup] node set %s", nodeSetName)

	logger.Debugf("processing started")
	ncs, err := datamodel.NcRepo.QueryNetworkContainersInNodeSet(nodeSetName)
	if err != nil {
		return dontRemoveFromPendingQ, err
	}

	logger.Debugf("%d associated ncs", len(ncs))

	var unprocessedNCs []string
	for _, nc := range ncs {
		// todo: this is not very efficient, consider a semaphore
		_, err := startNetworkContainerDeletion(nc)
		if err != nil {
			logger.Errorf("could not start deletion for nc %s: %v", nc.Config.NetworkContainerID, err)
			unprocessedNCs = append(unprocessedNCs, nc.Config.NetworkContainerID)
		}
	}

	if len(unprocessedNCs) > 0 {
		return dontRemoveFromPendingQ, api.InternalServerErrorf(api.StatusInternalServerError, "could not start nc deletion for ncs: %s", strings.Join(unprocessedNCs, ","))
	}

	// at this point, all ncs associated to the nodeset will be in deletion asynchronously (or already deleted)
	for {
		ncs, err := datamodel.NcRepo.QueryNetworkContainersInNodeSet(nodeSetName)
		if err != nil {
			return dontRemoveFromPendingQ, err
		}

		if len(ncs) == 0 {
			logger.Debugf("no associated network containers remaining")
			break
		}

		// todo: there is a chance more are being added since nc creation locks on nodes, not node sets.
		//  until that race condition is fixed, check on every iteration for any that could have been added
		for _, nc := range ncs {
			if !nc.Config.MarkedForDelete {
				logger.Debugf("nc %s on node %s added during node set deletion. queueing deletion")
				_, err := startNetworkContainerDeletion(nc)
				if err != nil {
					return dontRemoveFromPendingQ, err
				}
			}
		}

		logger.Debugf("%d network containers remaining", len(ncs))
		time.Sleep(time.Second * 5) // todo: configurable
	}

	logger.Debugf("all network containers removed")
	// all should be deleted now, safe to remove all nodes in node set
	nodes, err := datamodel.NodeConfigRepo.TryQueryAllNodeConfigurations(nodeSetName)
	if err != nil {
		return dontRemoveFromPendingQ, err
	}

	logger.Debugf("%d associated nodes to be deleted", len(nodes))

	var undeletedNodes []string
	for _, node := range nodes {
		nodeLock := map[string]string{datamodel.NodeRepositoryName: node.NodeID}
		sessionManager.BeginWriteSession(nodeLock)

		if err := datamodel.NodeConfigRepo.DeleteNode(node.NodeID); err != nil {
			logger.Errorf("could not delete node %s: %v", node.NodeID, err)
			undeletedNodes = append(undeletedNodes, node.NodeID)
		}

		sessionManager.EndWriteSession(nodeLock)
	}

	if len(undeletedNodes) > 0 {
		return dontRemoveFromPendingQ, api.InternalServerErrorf(api.StatusInternalServerError, "could not delete nodes: %s", strings.Join(undeletedNodes, ","))
	}

	logger.Debugf("all nodes removed")

	if err := datamodel.NodeSetsRepo.DeleteNodeSet(nodeSetName); err != nil {
		return dontRemoveFromPendingQ, api.InternalServerErrorf(api.StatusInternalServerError, "could not delete node set: %v", err)
	}

	logger.Debugf("nodeset deleted")

	return removeFromPendingQ, nil
}

func startNetworkContainerDeletion(nc datamodel.NetworkContainerEntity) (datamodel.NetworkContainerEntity, *api.JsonErr) {
	ncLock := map[string]string{datamodel.NCRepositoryName: nc.Config.NetworkContainerID}
	sessionManager.BeginWriteSession(ncLock)
	defer sessionManager.EndWriteSession(ncLock)

	requestContext := datamodel.RequestContext{
		Type:      datamodel.NetworkContainerRequest,
		NetworkID: nc.Config.NetworkID,
		NodeID:    nc.Config.GetAssociationInfo().NodeID,
	}

	// even if the nc is marked for deletion and status is pending, we need to ensure a corresponding queue item
	if nc.Config.MarkedForDelete && nc.Config.Status == datamodel.Pending {
		if err := datamodel.EnsureNCPendingWorkItem(nc.Config.NetworkID, nc.Config.NetworkContainerID, nc.Config.GetAssociationInfo().NodeID); err != nil {
			return datamodel.NetworkContainerEntity{}, err
		}

		return nc, nil
	}

	if !datamodel.IsDeleteInProgress(nc.Config.State) {
		revertState, err := datamodel.GetRevertState(nc.Config.State)
		if err != nil {
			return datamodel.NetworkContainerEntity{}, err // todo: if the nc is in an invalid state, should we just try to delete it?
		}
		nc.Config.State = revertState
	}

	nc.Config.MarkedForDelete = true
	nc.Config.Status = datamodel.Pending
	nc.Config.FailureDetail = api.JsonErr{}

	if err := datamodel.NcRepo.PersistNetworkContainerData(&nc); err != nil {
		return datamodel.NetworkContainerEntity{}, err
	}

	if err := datamodel.UpsertPendingQueueItem(nc.RowKey, requestContext); err != nil {
		return datamodel.NetworkContainerEntity{}, err
	}

	return nc, nil
}
