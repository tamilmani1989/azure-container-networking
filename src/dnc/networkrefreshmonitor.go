package main

import (
	"dnc/api"
	"dnc/datamodel"
	"dnc/nmagentcontracts"
	"strconv"
	"strings"
	"time"
)

func monitorNetworkUpdates() {
	monitorTimerChannel := time.NewTicker(
		time.Millisecond * time.Duration(config.DNCSettings.NetworkRefreshInterval)).C
	for {
		select {
		case <-monitorTimerChannel:
			if replicaInstance.IsReplicaPrimary() {
				refreshNetworks()
			}
			break
		case <-networkMonitorStartProcessing:
			refreshNetworks()
			break
		case <-networkMonitorStopProcessing:
			break
		}
	}
}

func refreshNetworks() {
	networksMap, err := datamodel.GetAllNetworkIDs()
	if err != nil {
		logger.Errorf("Failed to retrieve networks from repository due to error %s", err.Error())
	}

	// refresh network configurations
	for networkKey := range networksMap {
		refreshNetworkConfiguration(networkKey)
	}
}

func refreshNetworkConfiguration(networkKey string) {
	networkID := datamodel.GetNetworkIdFromKey(networkKey)
	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	networkSessionLockedForRead := false
	sessionManager.BeginReadSession(sessionInfo)
	networkSessionLockedForRead = true
	defer func() {
		if networkSessionLockedForRead {
			sessionManager.EndReadSession(sessionInfo)
		}
	}()

	var failureDetail *api.JsonErr
	var networkResponse *nmagentcontracts.VirtualNetworkResponse

	repoNetworkConfiguration, err := datamodel.TryQueryNetworkConfigurationByKey(networkKey)
	if err != nil {
		logger.Errorf("Retrieval of configuration for network %s failed with error:%s.\n",
			networkID,
			err.String())
		return
	} else if repoNetworkConfiguration == nil {
		logger.Debugf("Configuration for Network %s not found.\n", networkID)
		return
	}

	if repoNetworkConfiguration.NetworkState != datamodel.Join ||
		repoNetworkConfiguration.Status != datamodel.Completed {
		logger.Debugf("Skipping refreshing of network %s as it is in state:%s %s.\n",
			networkID,
			repoNetworkConfiguration.NetworkState,
			repoNetworkConfiguration.Status)
		return
	}

	sessionManager.EndReadSession(sessionInfo)
	networkSessionLockedForRead = false

	_, failureDetail = postRequestToAgentForNetworkJoin(networkID)
	if failureDetail != nil {
		logger.Errorf("Post request Network %s to Host Agent failed with error %s.\n",
			networkID,
			failureDetail.String())

		return
	}

	_, failureDetail, networkResponse = getNetworkConfigurationFromAgent(networkID)
	if failureDetail != nil {
		logger.Errorf("Request to get configuration for network %s failed with error %s.\n",
			networkID,
			failureDetail.String())
		return
	}

	networkVersion, parseNetworkVersionErr := strconv.ParseUint(networkResponse.VnetVersion, 10, 64)
	if parseNetworkVersionErr != nil {
		logger.Errorf("Failed to parse network version:%s for network:%s failed with error %s.\n",
			networkResponse.VnetVersion,
			networkID,
			parseNetworkVersionErr.Error())
		return
	}

	networkSessionLockedForWrite := false
	sessionManager.BeginWriteSession(sessionInfo)
	networkSessionLockedForWrite = true
	defer func() {
		if networkSessionLockedForWrite {
			sessionManager.EndWriteSession(sessionInfo)
		}
	}()

	checkAndAddTagsCache(networkResponse)
	if repoNetworkConfiguration.Version == networkVersion {
		return
	}

	_, failureDetail, subnetsToAdd, subnetsToRemove, subnetsToUpdate :=
		validateAndRetrieveNetworkConfigurationUpdate(
			repoNetworkConfiguration,
			networkResponse)

	if failureDetail != nil {
		logger.Errorf("Validation of response %v for network %s failed with error %s.\n",
			networkResponse,
			networkID,
			failureDetail.String())
		return
	}

	for subnetName := range subnetsToRemove {
		failureDetail = datamodel.DeleteSubnetConfiguration(networkID, subnetName)
		if failureDetail != nil {
			logger.Errorf("Deletion of subnet %s in network %s failed with error %s.\n",
				subnetName,
				networkID,
				failureDetail.String())
			return
		}
	}

	ipSubnetMap := make(map[string]string)
	subnetsInUpdatedResponse := make(map[string]nmagentcontracts.Subnet)
	for _, subnet := range networkResponse.Subnets {
		subnetsInUpdatedResponse[subnet.SubnetName] = subnet
		ipSubnetMap[subnet.SubnetName] =
			datamodel.GetSubnetKey(networkID, subnet.SubnetName)
	}

	subnetNetworkContainersMap := make(map[string][]*datamodel.NetworkContainerInfo)
	for subnetName, currentRepoSubnetCfg := range subnetsToUpdate {
		subnetNetworkContainersMap[subnetName], err = datamodel.GetNetworkContainersInSubnet(currentRepoSubnetCfg, false)
		if err != nil {
			logger.Errorf("Failed to retrieve network containers in subnet %s Network %s due to error %s",
				subnetName,
				networkID,
				err.String())
		}
	}

	logger.Debugf("Updating vnetconfig with current version %d Phase 1 commit Response:%v\n",
		repoNetworkConfiguration.Version,
		networkResponse)
	var dnsServers []string
	if len(networkResponse.DNSServers) == 0 {
		dnsServers = make([]string, 0)
	} else {
		dnsServers = strings.Split(networkResponse.DNSServers, ",")
	}

	repoNetworkConfiguration.CNetSpace = networkResponse.CNetSpace
	repoNetworkConfiguration.VNetSpace = networkResponse.VNetSpace
	repoNetworkConfiguration.Subnets = ipSubnetMap
	repoNetworkConfiguration.DNSServers = dnsServers

	failureDetail = datamodel.PersistNetworkConfiguration(repoNetworkConfiguration)
	if failureDetail != nil {
		logger.Errorf("Persisting configuration of network %s failed with error %s. \n",
			networkID,
			failureDetail.String())
	}

	for subnetName := range subnetsToAdd {
		updatedSubnetCfg := subnetsInUpdatedResponse[subnetName]
		subnetConfig := datamodel.SubnetConfiguration{
			NetworkID:         networkID,
			SubnetName:        subnetName,
			IPSubnet:          updatedSubnetCfg.AddressPrefix,
			IsDelegatedSubnet: true,
			Gateway:           updatedSubnetCfg.DefaultGateway}
		failureDetail = datamodel.PutSubnetConfiguration(&subnetConfig)
		if failureDetail != nil {
			logger.Errorf("Persisting configuration for subnet %s in network %s failed with error %s. \n",
				subnetName,
				networkID,
				failureDetail.String())
			return
		}
	}

	for subnetName, currentRepoSubnetCfg := range subnetsToUpdate {
		subnetKey := datamodel.GetSubnetKey(networkID, subnetName)
		ipSubnetMap[subnetName] = subnetKey
		updatedSubnetCfg := subnetsInUpdatedResponse[subnetName]
		currentRepoSubnetCfg.Tags = updatedSubnetCfg.Tags
		datamodel.PutSubnetConfiguration(currentRepoSubnetCfg)
		if failureDetail != nil {
			logger.Errorf("Persisting updated configuration for subnet %s in network %s failed with error %s. \n",
				subnetName,
				networkID,
				failureDetail.String())
			return
		}
	}

	logger.Debugf("Updating vnetconfig with received version %d Phase 2 commit Response:%s\n",
		repoNetworkConfiguration.Version,
		networkResponse.String())

	repoNetworkConfiguration.Version = networkVersion

	failureDetail = datamodel.PersistNetworkConfiguration(repoNetworkConfiguration)
	if failureDetail != nil {
		logger.Errorf("Persisting configuration of network %s failed with error %s. \n",
			networkID,
			failureDetail.String())
		return
	}

	logger.Debugf("Successfully refreshed configuration for network %s", networkID)
}

func CreateAssociationTask(nodeID, networkID, networkContainerID string) *api.JsonErr {
	sessionInfo := map[string]string{
		datamodel.NodeRepositoryName:    nodeID,
		datamodel.NetworkRepositoryName: networkID}
	sessionManager.BeginWriteSession(sessionInfo)
	defer sessionManager.EndWriteSession(sessionInfo)
	repoNetworkContainerConfiguration, err := datamodel.TryQueryNetworkContainer(
		networkID,
		networkContainerID)

	if err != nil {
		logger.Errorf("Failed to retrieve configuration for network container %s due to error %s",
			networkContainerID,
			err.String())
		return err
	}

	if repoNetworkContainerConfiguration == nil {
		logger.Debugf("Configuration for network container %s not found.",
			networkContainerID)
		return nil
	}

	if repoNetworkContainerConfiguration.MarkedForDelete {
		return nil
	}

	if repoNetworkContainerConfiguration.Status == datamodel.Failed ||
		repoNetworkContainerConfiguration.Status == datamodel.Denied {
		return nil
	}
	if repoNetworkContainerConfiguration.State == datamodel.Program &&
		repoNetworkContainerConfiguration.Status == datamodel.Completed {
		repoNetworkContainerConfiguration.State = datamodel.RefreshAssociation
		return datamodel.PersistNCConfiguration(repoNetworkContainerConfiguration, true)
	}

	return nil
}
