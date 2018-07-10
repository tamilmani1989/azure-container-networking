package main

import (
	"dnc/datamodel"
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
		logger.Errorf("Failed to retrieve nodeIDs from repository due to error %s", err.Error())
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
	if err != nil {
		return
	}

	for networkID, ncMap := range repoNodeConfiguration.DeployedNetworkContainerMap {
		for ncID := range ncMap {
			CreateAssociationTask(nodeID, networkID, ncID)
		}
	}
}
