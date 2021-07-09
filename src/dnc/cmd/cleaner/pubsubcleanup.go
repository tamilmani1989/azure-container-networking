package main

import (
	"context"
	"dnc/api"
	"dnc/datamodel"
	"dnc/pubsubproxy"
)

// This will unpublish the pending NC state and then delete the associated ips
func UnpublishNCState(ctx context.Context, repo *partitionRepo) *api.JsonErr {
	// First parse the PendingPublishRequest entries and process the pending unpublish job types
	err := unpublishFromPendingPublishRequests(ctx, repo)
	if err != nil {
		Log(ctx).Errorf("error with pending publish: %v", err)
		return err
	}

	// Now check if there is any pending NCs
	return unpublishFromPendingNetworkContainers(ctx, repo)
}

// This will unpublish the pending NC state and then delete the associated ips
func unpublishFromPendingPublishRequests(ctx context.Context, repo *partitionRepo) *api.JsonErr {
	// First parse the PendingPublishRequest entries and process the pending unpublish job types
	pendingPublishRequests, err := repo.pendingPublishRequestRepo.GetNcPendingPublishRequestContexts()
	if err != nil {
		Log(ctx).Errorf("error GetNcPendingPublishRequestContexts %v", err)
		return err
	}

	for rowKey, requestEntity := range pendingPublishRequests {
		requestContext := requestEntity.Configuration
		if requestContext.Type == datamodel.NetworkContainerUnpublishRequest {
			err = pubsubproxy.ForceUnpublishInAz(requestContext.InfraNetworkID, requestContext.NodeInterfaceIP, requestContext.NetworkContainerID, requestContext.PubsubAzId)
			if err != nil {
				// todo: do we need to retry?
				return err
			}
		}

		Log(ctx).Debugf("Deleting Pending pubsub Job,  rowKey %s, requestContext %+v",
			rowKey,
			requestContext)
		err = repo.pendingPublishRequestRepo.RemoveNcPendingPublishContext(requestContext.NetworkID, requestContext.NetworkContainerID, requestContext.PubsubAzId)
		if err != nil {
			return err
		}
	}

	Log(ctx).Debugf("UnpublishNCState: State is unpublished for pending publish requests now wiping PendingPublishRequest to avoid re-unpublish")

	return repo.pendingPublishRequestRepo.DeleteAllRows()
}

// This will unpublish the pending NC state for all the Pending NetworkContainers in the NC table
func unpublishFromPendingNetworkContainers(ctx context.Context, repo *partitionRepo) *api.JsonErr {
	ncs, err := repo.ncRepo.GetAllNetworkContainers()
	if err != nil {
		Log(ctx).Errorf("error GetAllNetworkContainers: %v", err)
		return err
	}

	for _, nc := range ncs {
		// Get the associated Node for a given NC to fetch InfraVnetId
		nodeID := nc.Config.GetAssociationInfo().NodeID
		repoNodeConfig, err := repo.nodeRepo.TryQueryNodeConfiguration(nodeID)
		if err != nil {
			Log(ctx).Errorf("error TryQueryNodeConfiguration: %v", err)
			return err
		}

		if repoNodeConfig == nil {
			Log(ctx).Errorf("error repoNodeConfig is nil for nc %s", nc.Config.NetworkContainerID)
			return api.BadRequestf(api.NodeNotFound,
				"configuration for node: %s not found. Associated NC: %s", nodeID, nc.Config.NetworkContainerID)
		}

		var nodeInterfaceIp string
		assocInfo := nc.Config.GetAssociationInfo()

		if len(assocInfo.InterfaceID) == 0 {
			nodeInterfaceIp = repoNodeConfig.NodeInfo.IPAddresses[0]
		} else {
			nodeInterfaceIp = assocInfo.InterfaceID
		}

		// todo: retry? Caller will but...
		err = pubsubproxy.ForceUnpublishInAllAzs(repoNodeConfig.NodeInfo.InfrastructureNetwork, nodeInterfaceIp, nc.Config.NetworkContainerID)
		if err != nil {
			Log(ctx).Errorf("error ForceUnpublishInAllAzs %s, %v", err)
			return err
		}
	}

	Log(ctx).Debugf("UnpublishNCState: State is unpublished for NetworkContainers Now wiping NetworkContainer table to avoid re-unpublish")

	return repo.ncRepo.DeleteAllRows()
}
