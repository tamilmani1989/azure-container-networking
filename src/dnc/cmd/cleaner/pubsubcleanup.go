package main

import (
	"dnc/api"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/pubsubproxy"
)

// This will unpublish the pending NC state and then delete the associated ips
func UnpublishNCState(repo *partitionRepo) *api.JsonErr {
	// First parse the PendingPublishRequest entries and process the pending unpublish job types
	err := unpublishFromPendingPublishRequests(repo)
	if err != nil {
		loggershim.LS.Errorf("error with pending publish %s, %v", repo.partitionKey, err)
		return err
	}

	// Now check if there is any pending NCs
	return unpublishFromPendingNetworkContainers(repo)
}

// This will unpublish the pending NC state and then delete the associated ips
func unpublishFromPendingPublishRequests(repo *partitionRepo) *api.JsonErr {
	// First parse the PendingPublishRequest entries and process the pending unpublish job types
	pendingPublishRequests, err := repo.pendingPublishRequestRepo.GetNcPendingPublishRequestContexts()
	if err != nil {
		loggershim.LS.Errorf("error GetNcPendingPublishRequestContexts %s, %v", repo.partitionKey, err)
		return err
	}

	for rowKey, requestContext := range pendingPublishRequests {
		if requestContext.Type == datamodel.NetworkContainerUnpublishRequest {
			err = pubsubproxy.ForceUnpublishInAz(requestContext.InfraNetworkID, requestContext.NodeInterfaceIP, requestContext.NetworkContainerID, requestContext.PubsubAzId)
			if err != nil {
				// todo: do we need to retry?
				return err
			}
		}

		loggershim.LS.Debugf("Deleting Pending pubsub Job, partitionKey %s, rowKey %s, requestContext %+v",
			repo.partitionKey,
			rowKey,
			requestContext)
		err = repo.pendingPublishRequestRepo.RemoveNcPendingPublishContext(requestContext.NetworkID, requestContext.NetworkContainerID, requestContext.PubsubAzId)
		if err != nil {
			return err
		}
	}

	loggershim.LS.Debugf("UnpublishNCState: State is unpublished for pending publish requests for PartitionKey %s, "+
		"now wiping PendingPublishRequest to avoid re-unpublish", repo.partitionKey)

	return repo.pendingPublishRequestRepo.DeleteAllRows()
}

// This will unpublish the pending NC state for all the Pending NetworkContainers in the NC table
func unpublishFromPendingNetworkContainers(repo *partitionRepo) *api.JsonErr {
	networkContainersConfig, err := repo.ncRepo.GetAllNetworkContainers()
	if err != nil {
		loggershim.LS.Errorf("error GetAllNetworkContainers %s, %v", repo.partitionKey, err)
		return err
	}

	for _, repoNCConfig := range networkContainersConfig {
		// Get the associated Node for a given NC to fetch InfraVnetId
		nodeID := repoNCConfig.GetAssociationInfo().NodeID
		repoNodeConfig, err := repo.nodeRepo.TryQueryNodeConfiguration(nodeID)
		if err != nil {
			loggershim.LS.Errorf("error TryQueryNodeConfiguration %s, %v", repo.partitionKey, err)
			return err
		}

		if repoNodeConfig == nil {
			loggershim.LS.Errorf("error repoNodeConfig is nil for nc %s, partitionKey: %s", repoNCConfig.NetworkContainerID, repo.partitionKey)
			return api.BadRequestf(api.NodeNotFound,
				"configuration for node: %s not found. Associated NC: %s", nodeID, repoNCConfig.NetworkContainerID)
		}

		var nodeInterfaceIp string
		assocInfo := repoNCConfig.GetAssociationInfo()

		if len(assocInfo.InterfaceID) == 0 {
			nodeInterfaceIp = repoNodeConfig.NodeInfo.IPAddresses[0]
		} else {
			nodeInterfaceIp = assocInfo.InterfaceID
		}

		// todo: retry? Caller will but...
		err = pubsubproxy.ForceUnpublishInAllAzs(repoNodeConfig.NodeInfo.InfrastructureNetwork, nodeInterfaceIp, repoNCConfig.NetworkContainerID)
		if err != nil {
			loggershim.LS.Errorf("error ForceUnpublishInAllAzs %s, %v", repo.partitionKey, err)
			return err
		}
	}

	loggershim.LS.Debugf("UnpublishNCState: State is unpublished for NetworkContainers in partition: %s, "+
		"Now wiping NetworkContainer table to avoid re-unpublish", repo.partitionKey)

	return repo.ncRepo.DeleteAllRows()
}
