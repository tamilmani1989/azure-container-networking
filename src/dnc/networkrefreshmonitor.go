package main

import (
	"dnc/api"
	"dnc/datamodel"
	"dnc/nmagentcontracts"
	"strconv"
	"strings"
	"time"
)

var networkRefreshMonitorData *api.RefreshMonitorStatus

func monitorNetworkUpdates() {
	networkRefreshMonitorData = &api.RefreshMonitorStatus{}
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
			logger.Debugf("[MonitorNetworkUpdates]:Refreshing networks as replica became primary")
			refreshNetworks()
			break
		case <-networkMonitorStopProcessing:
			logger.Debugf("[MonitorNetworkUpdates]:Received stop processing of refresh networks")
			break
		}
	}
}

func refreshNetworks() {
	networksMap, err := datamodel.GetAllNetworkIDs()
	if err != nil {
		logger.Errorf("Failed to retrieve networks from repository due to error %s", err.Error())
		return
	}

	startTime := time.Now().UTC()
	networkRefreshMonitorData.StartTime = startTime.String()
	networkRefreshMonitorData.ProcessedItems = 0
	networkRefreshMonitorData.FailedItems = 0
	networkRefreshMonitorData.TotalItems = len(networksMap)
	networkRefreshMonitorData.EndTime = time.Time{}.String()
	networkRefreshMonitorData.IsThrottled = false
	networkRefreshMonitorData.IsRefreshInProgress = true
	networkRefreshMonitorData.TimeToRefresh = ""
	defer func() {
		networkRefreshMonitorData.TimeToRefresh = time.Since(startTime).String()
		networkRefreshMonitorData.EndTime = time.Now().UTC().String()
		networkRefreshMonitorData.IsRefreshInProgress = false
		logger.Debugf("Vnets Count:%d, Time to refresh %v", len(networksMap), time.Since(startTime))
	}()

	processedItemCountInBatch := 0
	// refresh network configurations
	for networkKey := range networksMap {
		areCallsThrottled, success := refreshNetworkConfiguration(networkKey, false)
		networkRefreshMonitorData.ProcessedItems++
		if !success {
			networkRefreshMonitorData.FailedItems++
		}

		processedItemCountInBatch++
		if areCallsThrottled || processedItemCountInBatch > config.DNCSettings.NetworkRefreshBatchSize {
			logger.Debugf("Backingoff. Throttled:{%v}, ProcessedItemCountInBatch:{%v}, RefreshBatchSize:{%v}",
				areCallsThrottled,
				networkRefreshMonitorData.ProcessedItems,
				config.DNCSettings.NetworkRefreshBatchSize)
			networkRefreshMonitorData.IsThrottled = areCallsThrottled
			processedItemCountInBatch = 0
			select {
			case <-time.After(sleepIntervalBetweenNetworkBatches):
				// sleep and process next batch
			case <-networkMonitorStopProcessing:
				logger.Debugf("Received stop processing signal while in the process of refreshing networks")
				return
			}
		} else {
			time.Sleep(sleepIntervalBetweenNetworkRefresh)
		}
	}
}

func refreshNetworkConfiguration(
	networkKey string,
	sessionLockAcquired bool) (bool, bool) {
	var isThrottled bool
	networkID := datamodel.GetNetworkIdFromKey(networkKey)
	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	networkSessionLockedForRead := false

	if !sessionLockAcquired {
		sessionManager.BeginReadSession(sessionInfo)
		networkSessionLockedForRead = true

		defer func() {
			if networkSessionLockedForRead {
				sessionManager.EndReadSession(sessionInfo)
			}
		}()
	}

	var failureDetail *api.JsonErr
	var networkResponse *nmagentcontracts.VirtualNetworkResponse
	repoNetworkConfiguration, err := datamodel.TryQueryNetworkConfigurationByKey(networkKey)
	if err != nil {
		logger.Errorf("[RefreshNetworkConfiguration] Failed to retrieve network %s with error: %s",
			networkID,
			err.String())
		return false, false
	} else if repoNetworkConfiguration == nil {
		logger.Debugf("[RefreshNetworkConfiguration] Configuration for Network %s not found.", networkID)
		return false, false
	}

	if repoNetworkConfiguration.NetworkState != datamodel.Join ||
		(repoNetworkConfiguration.Status != datamodel.Completed &&
			repoNetworkConfiguration.Status != datamodel.Refreshing) {
		logger.Debugf("[RefreshNetworkConfiguration] Skipping network %s as it is in state:%s %s.",
			networkID,
			repoNetworkConfiguration.NetworkState,
			repoNetworkConfiguration.Status)
		return false, false
	}

	if networkSessionLockedForRead {
		sessionManager.EndReadSession(sessionInfo)
		networkSessionLockedForRead = false
	}

	// Try to retrieve the network configuration fron NMAgent.
	_, failureDetail, networkResponse, isThrottled = getNetworkConfigurationFromAgent(networkID)
	if failureDetail != nil {
		if failureDetail.Code != api.ProcessingPending {
			logger.Errorf("[RefreshNetworkConfiguration] Request to get network %s failed with error %s.",
				networkID,
				failureDetail.String())
			return isThrottled, false
		}

		// If response has status processing, indicates the network is not available with
		// NMagent (or) it is under processing. Trigger a join and bail out.
		_, failureDetail, isThrottled = postRequestToAgentForNetworkJoin(networkID)
		if failureDetail != nil {
			logger.Errorf("%s", failureDetail.String())
		}

		return isThrottled, false
	}

	if networkResponse == nil {
		logger.Debugf("Empty response for network %s", networkID)
		return false, false
	}

	networkVersion, parseNetworkVersionErr := strconv.ParseUint(networkResponse.VnetVersion, 10, 64)
	if parseNetworkVersionErr != nil {
		logger.Errorf("Failed to parse network version %s for network %s due to error %s",
			networkResponse.VnetVersion,
			networkID,
			parseNetworkVersionErr.Error())
		return false, false
	}

	if !sessionLockAcquired {
		sessionManager.BeginWriteSession(sessionInfo)
		defer func() {
			sessionManager.EndWriteSession(sessionInfo)
		}()
	}

	checkAndAddTagsCache(networkResponse)
	if repoNetworkConfiguration.Version == networkVersion &&
		repoNetworkConfiguration.Status != datamodel.Refreshing {
		return false, true
	}

	_, failureDetail, subnetsToAdd, subnetsToRemove, subnetsToUpdate :=
		validateAndRetrieveNetworkConfigurationUpdate(
			repoNetworkConfiguration,
			networkResponse)

	if failureDetail != nil {
		logger.Errorf("Validation of response %+v for network %s failed with error %s.",
			networkResponse,
			networkID,
			failureDetail.String())
		return false, false
	}

	ipSubnetMap := make(map[string]string)
	subnetsInUpdatedResponse := make(map[string]nmagentcontracts.Subnet)
	for _, subnet := range networkResponse.Subnets {
		subnetsInUpdatedResponse[subnet.SubnetName] = subnet
		ipSubnetMap[subnet.SubnetName] =
			datamodel.GetSubnetKey(networkID, subnet.SubnetName)
	}

	for subnetName := range subnetsToRemove {
		subnetConfig, _ := datamodel.TryQuerySubnetConfiguration(networkID, subnetName)
		if subnetConfig.UpdateStatus == datamodel.SubnetUpdateCompleted {
			failureDetail = datamodel.DeleteSubnetConfiguration(networkID, subnetName)
			if failureDetail != nil {
				logger.Errorf("%s", failureDetail.String())
				return false, false
			}
		} else if subnetConfig.UpdateStatus == datamodel.SubnetUpdatePending {
			// Check if the subnet to remove has UpdateStatus set to pending. If yes, do not remove it but set the
			// UpdateStatus as SubnetUpdateFailed
			ipSubnetMap[subnetName] = datamodel.GetSubnetKey(networkID, subnetName)
			logger.Debugf("Subnet %s in network %s has the update status as pending. Updating the status as UpdateFailed",
				subnetName, networkID)
			subnetConfig.UpdateStatus = datamodel.SubnetUpdateFailed
			subnetConfig.Joined = false
			failureDetail = datamodel.PutSubnetConfiguration(subnetConfig)
			if failureDetail != nil {
				logger.Errorf("%s", failureDetail.String())
				return false, false
			}
		} else if subnetConfig.UpdateStatus == datamodel.SubnetUpdateFailed {
			ipSubnetMap[subnetName] = datamodel.GetSubnetKey(networkID, subnetName)
		}
	}

	subnetNetworkContainersMap := make(map[string][]*datamodel.NetworkContainerInfo)
	for subnetName, currentRepoSubnetCfg := range subnetsToUpdate {
		subnetNetworkContainersMap[subnetName], err = datamodel.GetNetworkContainersInSubnet(currentRepoSubnetCfg, false)
		if err != nil {
			logger.Errorf("%s", err.String())
		}
	}

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
		logger.Errorf("%s.", failureDetail.String())
		return false, false
	}

	for subnetName := range subnetsToAdd {
		updatedSubnetCfg := subnetsInUpdatedResponse[subnetName]
		subnetConfig := datamodel.SubnetConfiguration{
			NetworkID:         networkID,
			SubnetName:        subnetName,
			IPSubnet:          updatedSubnetCfg.AddressPrefix,
			IsDelegatedSubnet: true,
			Gateway:           updatedSubnetCfg.DefaultGateway,
			UpdateStatus:      datamodel.SubnetUpdateCompleted}
		failureDetail = datamodel.PutSubnetConfiguration(&subnetConfig)
		if failureDetail != nil {
			logger.Errorf("%s", failureDetail.String())
			return false, false
		}
	}

	for subnetName, currentRepoSubnetCfg := range subnetsToUpdate {
		subnetKey := datamodel.GetSubnetKey(networkID, subnetName)
		ipSubnetMap[subnetName] = subnetKey
		updatedSubnetCfg := subnetsInUpdatedResponse[subnetName]

		currentRepoSubnetCfg.Tags = updatedSubnetCfg.Tags
		currentRepoSubnetCfg.IPSubnet = updatedSubnetCfg.AddressPrefix
		currentRepoSubnetCfg.Gateway = updatedSubnetCfg.DefaultGateway
		currentRepoSubnetCfg.IsDelegatedSubnet = true
		currentRepoSubnetCfg.UpdateStatus = datamodel.SubnetUpdateCompleted
		datamodel.PutSubnetConfiguration(currentRepoSubnetCfg)
		if failureDetail != nil {
			logger.Errorf("%s", failureDetail.String())
			return false, false
		}
	}

	repoNetworkConfiguration.Version = networkVersion
	repoNetworkConfiguration.Status = datamodel.Completed
	failureDetail = datamodel.PersistNetworkConfiguration(repoNetworkConfiguration)
	if failureDetail != nil {
		logger.Errorf("%s", failureDetail.String())
		return false, false
	}

	return false, true
}

// CreateAssociationTask - Create network container association task
func CreateAssociationTask(nodeID, networkID, networkContainerID string) *api.JsonErr {
	sessionInfo := map[string]string{datamodel.NCRepositoryName: networkContainerID}
	sessionManager.BeginWriteSession(sessionInfo)
	defer sessionManager.EndWriteSession(sessionInfo)

	repoNetworkContainerConfiguration, err := datamodel.TryQueryNetworkContainer(
		networkID,
		networkContainerID)
	if err != nil {
		logger.Errorf("Failed to retrieve configuration for NC %s due to error %s",
			networkContainerID,
			err.String())
		return err
	}

	if repoNetworkContainerConfiguration == nil {
		logger.Debugf("Configuration for NC %s not found.",
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
		return datamodel.PersistNCConfiguration(repoNetworkContainerConfiguration, datamodel.AddToPendingQ)
	}

	return nil
}

// GetNetworkRefreshMonitorStatus - Helper to get the refresh monitor status
func GetNetworkRefreshMonitorStatus() api.RefreshMonitorStatus {
	return *networkRefreshMonitorData
}
