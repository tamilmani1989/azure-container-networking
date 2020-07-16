package main

import (
	"dnc/datamodel"
	"dnc/resourceupdatehandler/networkupdatehandler"
	"time"
)

func monitorNetworkContainers() {
	monitorTimerChannel := time.NewTicker(time.Millisecond *
		time.Duration(config.DNCSettings.NetworkContainerRefreshInterval)).C
	for {
		select {
		case <-monitorTimerChannel:
			if replicaInstance.IsReplicaPrimary() {
				refreshNodes()
			}
			break
		case <-refreshNCStartProcessing:
			refreshNodes()
			break
		case <-refreshNCStopProcessing:
			break
		}
	}
}

func refreshNodes() {
	nodesMap, err := datamodel.GetAllNodeIDs()
	if err != nil {
		logger.Errorf("Failed to retrieve nodes from repository due to error %s", err.Error())
	}
	for nodeKey := range nodesMap {
		refreshNodeNetworkContainers(nodeKey)
	}
}

func refreshNodeNetworkContainers(nodeKey string) {
	nodeID := datamodel.GetNodeIDFromKey(nodeKey)
	sessionInfo := map[string]string{datamodel.NodeRepositoryName: nodeID}

	sessionManager.BeginReadSession(sessionInfo)
	repoNodeConfiguration, err := datamodel.TryQueryNodeConfigurationByKey(nodeKey)
	sessionManager.EndReadSession(sessionInfo)
	// If the node gets deleted after DNC reads the node list using GetAllNodeIDs,
	// the nodeConfiguration for the deleted node will be nil. The nil check below
	// protects against dnc crash due to nil pointer dereference
	if err != nil || repoNodeConfiguration == nil {
		return
	}

	for networkID, ncMap := range repoNodeConfiguration.DeployedNetworkContainerMap {
		for ncID := range ncMap {
			if err := networkupdatehandler.CreateAssociationTask(nodeID, networkID, ncID); err != nil {
				logger.Errorf("Failed to create NC refresh association task for NC %s due to error %v",
					ncID, err)
			}
		}
	}
}
