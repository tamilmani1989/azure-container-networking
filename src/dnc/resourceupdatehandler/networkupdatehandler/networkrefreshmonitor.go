package networkupdatehandler

import (
	"bytes"
	"dnc/aiwrapper"
	"dnc/api"
	"dnc/configuration"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/nmagentcontracts"
	"dnc/replica"
	"dnc/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-container-networking/aitelemetry"
)

var (
	networkRefreshMonitorData *api.RefreshMonitorStatus
	config                    *configuration.Configuration
	logger                    *loggershim.LoggerShim
	replicaInstance           *replica.Replica
	sessionManager            *utils.SessionManager

	// HTTP client object to be reused wherever needed
	httpClient *http.Client

	// SleepIntervalBetweenBatches
	sleepIntervalBetweenNetworkBatches time.Duration

	// SleepIntervalBetweenNetworkRefresh
	sleepIntervalBetweenNetworkRefresh time.Duration

	NetworkMonitorStartProcessing = make(chan bool)
	NetworkMonitorStopProcessing  = make(chan bool)
)

func Init(
	cfg *configuration.Configuration,
	replicaInst *replica.Replica,
	sessionMgr *utils.SessionManager,
	httpC *http.Client,
	loggershim *loggershim.LoggerShim) {
	config = cfg
	replicaInstance = replicaInst
	logger = loggershim
	sessionManager = sessionMgr
	httpClient = httpC
	sleepIntervalBetweenNetworkBatches = time.Duration(config.DNCSettings.SleepIntervalBetweenNetworkBatches) * time.Millisecond
	sleepIntervalBetweenNetworkRefresh = time.Duration(config.DNCSettings.SleepIntervalBetweenNetworkRefresh) * time.Millisecond
}

func MonitorNetworkUpdates() {
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
		case <-NetworkMonitorStartProcessing:
			logger.Debugf("[MonitorNetworkUpdates]:Refreshing networks as replica became primary")
			refreshNetworks()
			break
		case <-NetworkMonitorStopProcessing:
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
		areCallsThrottled, success := RefreshNetworkConfiguration(networkKey, false)
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
			case <-NetworkMonitorStopProcessing:
				logger.Debugf("Received stop processing signal while in the process of refreshing networks")
				return
			}
		} else {
			time.Sleep(sleepIntervalBetweenNetworkRefresh)
		}
	}
}

func RefreshNetworkConfiguration(
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

	// Additonal check for NC state of RefreshAssociation is needed because
	// it may happen that NC state changes to refreshAssociation from Program but
	// DNC fails to add pending request in the queue due to DNC crash or upgrading
	// the DNC during deployment
	if (repoNetworkContainerConfiguration.State == datamodel.Program ||
		repoNetworkContainerConfiguration.State == datamodel.RefreshAssociation) &&
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

func UpdateNetworkConfiguration(
	networkID string,
	repoNetworkConfiguration *datamodel.NetworkConfiguration,
	networkResponse *nmagentcontracts.VirtualNetworkResponse) (bool, *api.JsonErr) {
	var (
		failureDetail            *api.JsonErr
		isRetriableError         bool
		subnetsToAdd             map[string]nmagentcontracts.Subnet
		subnetsToRemove          map[string]bool
		subnetsInUpdatedResponse map[string]nmagentcontracts.Subnet
		ipSubnetMap              map[string]string
		vnetVersion              uint64
		dnsServers               []string
	)

	isRetriableError, failureDetail, subnetsToAdd, subnetsToRemove, _ =
		validateAndRetrieveNetworkConfigurationUpdate(
			repoNetworkConfiguration,
			networkResponse)

	if failureDetail != nil {
		if isRetriableError {
			return false, failureDetail
		}
		goto Cleanup
	}

	for subnetName := range subnetsToRemove {
		failureDetail = datamodel.DeleteSubnetConfiguration(networkID, subnetName)
		if failureDetail != nil {
			goto Cleanup
		}
	}

	ipSubnetMap = make(map[string]string)
	subnetsInUpdatedResponse = make(map[string]nmagentcontracts.Subnet)
	for _, subnet := range networkResponse.Subnets {
		subnetsInUpdatedResponse[subnet.SubnetName] = subnet
		ipSubnetMap[subnet.SubnetName] = datamodel.GetSubnetKey(networkID, subnet.SubnetName)
	}

	// Commiting virtual network config in two phases.
	// First phase, update only the configuration and the subnet
	// list and the state is still pending join.
	// This helps in driving cleanup as we have handle to all the subnets
	// and we can handle any failures that happens when persisting subnet
	// configurations.
	if len(networkResponse.DNSServers) == 0 {
		dnsServers = make([]string, 0)
	} else {
		dnsServers = strings.Split(networkResponse.DNSServers, ",")
	}

	vnetVersion, _ = strconv.ParseUint(networkResponse.VnetVersion, 10, 64)

	// Update the existing NetworkConfiguration with updated data
	repoNetworkConfiguration.Version = vnetVersion
	repoNetworkConfiguration.CNetSpace = networkResponse.CNetSpace
	repoNetworkConfiguration.VNetSpace = networkResponse.VNetSpace
	repoNetworkConfiguration.Subnets = ipSubnetMap
	repoNetworkConfiguration.DNSServers = dnsServers

	failureDetail = datamodel.PersistNetworkConfiguration(repoNetworkConfiguration)
	if failureDetail != nil {
		goto Cleanup
	}

	for subnetName := range subnetsToAdd {
		updatedSubnetCfg := subnetsInUpdatedResponse[subnetName]
		var tags []nmagentcontracts.Tag

		if updatedSubnetCfg.Tags == nil || len(updatedSubnetCfg.Tags) == 0 {
			tags = make([]nmagentcontracts.Tag, 0)
		} else {
			tags = updatedSubnetCfg.Tags
		}

		subnetConfig := datamodel.SubnetConfiguration{
			NetworkID:         networkID,
			SubnetName:        subnetName,
			IPSubnet:          updatedSubnetCfg.AddressPrefix,
			IsDelegatedSubnet: true,
			Gateway:           updatedSubnetCfg.DefaultGateway,
			Tags:              tags,
			UpdateStatus:      datamodel.SubnetUpdateCompleted}
		failureDetail = datamodel.PutSubnetConfiguration(&subnetConfig)
		if failureDetail != nil {
			goto Cleanup
		}
	}

	// Mark the join operation as completed.
	repoNetworkConfiguration.NetworkState = datamodel.Join
	repoNetworkConfiguration.Status = datamodel.Completed
	failureDetail = datamodel.PersistNetworkConfiguration(repoNetworkConfiguration)

Cleanup:
	// If it is an unknown failure then we return with marking the
	// operation has failed so we can retry later.
	if failureDetail != nil {
		if failureDetail.Code == api.StatusInternalServerError {
			return false, failureDetail
		}

		// Non retriable errors, mark the operation as failed.
		repoNetworkConfiguration.Status = datamodel.Failed
		repoNetworkConfiguration.FailureDetail = *failureDetail
		if err := datamodel.PersistNetworkConfiguration(repoNetworkConfiguration); err != nil {
			return false, err
		}
	}

	return true, nil
}

func ProcessVirtualNetworkJoin(
	networkID string,
	repoNetworkConfiguration *datamodel.NetworkConfiguration) (bool, *api.JsonErr) {
	var (
		failureDetail    *api.JsonErr
		isRetriableError bool
		networkResponse  *nmagentcontracts.VirtualNetworkResponse
	)

	isRetriableError, failureDetail, _ = postRequestToAgentForNetworkJoin(networkID)
	if failureDetail == nil {
		isRetriableError, failureDetail, networkResponse, _ = getNetworkConfigurationFromAgent(networkID)
	}

	if failureDetail != nil {
		// If it is an unknown failure then we return with marking the
		// operation has failed so we can retry later.
		if isRetriableError || (failureDetail.Code == api.StatusInternalServerError) {
			return false, failureDetail
		}

		// Non retriable errors, mark the operation as failed.
		repoNetworkConfiguration.Status = datamodel.Failed
		repoNetworkConfiguration.FailureDetail = *failureDetail
		if err := datamodel.PersistNetworkConfiguration(repoNetworkConfiguration); err != nil {
			return false, err
		}

		return true, nil
	}

	return UpdateNetworkConfiguration(networkID, repoNetworkConfiguration, networkResponse)
}

func validateAndRetrieveNetworkConfigurationUpdate(
	repoNetworkConfig *datamodel.NetworkConfiguration,
	networkConfigResponse *nmagentcontracts.VirtualNetworkResponse) (
	bool,
	*api.JsonErr,
	map[string]nmagentcontracts.Subnet,
	map[string]bool, map[string]*datamodel.SubnetConfiguration) {
	currentSubnets := make(map[string]*datamodel.SubnetConfiguration)

	var currentDNSServers []string
	var updatedDNSServers []string
	if len(networkConfigResponse.DNSServers) != 0 {
		updatedDNSServers = strings.Split(networkConfigResponse.DNSServers, ",")
	} else {
		updatedDNSServers = make([]string, 0)
	}

	if len(repoNetworkConfig.DNSServers) != 0 {
		currentDNSServers = repoNetworkConfig.DNSServers
	} else {
		currentDNSServers = make([]string, 0)
	}

	hasDNSServersChanged := !utils.AreSetsEqual(currentDNSServers, updatedDNSServers)

	var currentCnetSpaces []string
	var updatedCnetSpaces []string
	if len(networkConfigResponse.CNetSpace) != 0 {
		updatedCnetSpaces = strings.Split(networkConfigResponse.CNetSpace, ",")
	} else {
		updatedCnetSpaces = make([]string, 0)
	}

	if len(repoNetworkConfig.CNetSpace) != 0 {
		currentCnetSpaces = strings.Split(repoNetworkConfig.CNetSpace, ",")
	} else {
		currentCnetSpaces = make([]string, 0)
	}

	hasCnetSpacesChanged := !utils.AreSetsEqual(currentCnetSpaces, updatedCnetSpaces)

	// Retrieve the current subnets from the repository.
	for subnetName := range repoNetworkConfig.Subnets {
		subnetConfiguration, subnetCfgErr := datamodel.TryQuerySubnetConfiguration(repoNetworkConfig.NetworkID, subnetName)
		if subnetCfgErr != nil {
			return (subnetCfgErr.Code == api.StatusInternalServerError), subnetCfgErr, nil, nil, nil
		}

		if subnetConfiguration != nil {
			currentSubnets[subnetName] = subnetConfiguration
		}
	}

	subnetsToRemove := make(map[string]bool)
	subnetsToAdd := make(map[string]nmagentcontracts.Subnet)
	subnetsToUpdate := make(map[string]*datamodel.SubnetConfiguration)
	subnetNamesInResponse := make(map[string]bool)

	// Iterate through all the subnets in the response.
	for _, subnet := range networkConfigResponse.Subnets {
		subnetNamesInResponse[subnet.SubnetName] = true
		repoSubnetConfig, exist := currentSubnets[subnet.SubnetName]

		if exist && repoSubnetConfig != nil {
			// If the subnet exist in the repository and there are no allocations
			// (i.e) no reservations exist, then mark the subnet for update.
			if len(repoSubnetConfig.Reservations) == 0 {
				subnetsToUpdate[subnet.SubnetName] = repoSubnetConfig
				continue
			}

			// If reservations exists, then changing the IPSubnet is not allowed.
			// This should never happen.
			updatedSubnet, err := utils.ParseIPNet(subnet.AddressPrefix)
			if err != nil {
				msg := fmt.Sprintf("Failed to parse IPSubnet %s from received config %s",
					subnet.AddressPrefix,
					networkConfigResponse.String())
				return false, &api.JsonErr{
					Code:       api.NetworkJoinFailed,
					Text:       msg,
					HTTPStatus: http.StatusInternalServerError}, nil, nil, nil
			}

			currentSubnet, err := utils.ParseIPNet(repoSubnetConfig.IPSubnet)
			if err != nil {
				msg := fmt.Sprintf("Failed to parse IPSubnet %s from repo config %s.",
					subnet.AddressPrefix,
					repoSubnetConfig.String())
				return false, &api.JsonErr{
					Code:       api.NetworkJoinFailed,
					Text:       msg,
					HTTPStatus: http.StatusInternalServerError}, nil, nil, nil
			}

			if !utils.AreIPSubnetsEqual(currentSubnet, updatedSubnet) {
				msg := fmt.Sprintf(
					"Update of subnet %s is not allowed as there are allocations. Current subnet:%s, Updated subnet:%s.",
					repoSubnetConfig.SubnetName,
					currentSubnet.String(),
					updatedSubnet.String())
				return false, &api.JsonErr{
					Code:       api.NetworkJoinFailed,
					Text:       msg,
					HTTPStatus: http.StatusInternalServerError}, nil, nil, nil
			}

			if hasDNSServersChanged || hasCnetSpacesChanged {
				subnetsToUpdate[subnet.SubnetName] = repoSubnetConfig
			}
		} else {
			// This is a new subnet and mark it to be added to the repository.
			subnetsToAdd[subnet.SubnetName] = subnet
		}
	}

	// Cleanup all the subnets that are no longer present in the received configuration.
	for subnetName, repoSubnetConfig := range currentSubnets {
		_, exists := subnetNamesInResponse[subnetName]
		// If a subnet has been deleted outside of DNC,
		// delete it only if there are no resources in them
		if !exists {
			if len(repoSubnetConfig.Reservations) > 0 {
				logger.Errorf("Error: Leaked NC's detected: Subnet %s in network %s is removed but there are allocations.",
					repoSubnetConfig.SubnetName,
					repoSubnetConfig.NetworkID)

				// Send the metric
				sendLeakedNCMetric(repoSubnetConfig.NetworkID, repoSubnetConfig.SubnetName, len(repoSubnetConfig.Reservations))
			}

			subnetsToRemove[subnetName] = true
		}
	}

	return false, nil, subnetsToAdd, subnetsToRemove, subnetsToUpdate
}

// Retrieve network configuration from host agent.
func getNetworkConfigurationFromAgent(networkID string) (bool, *api.JsonErr, *nmagentcontracts.VirtualNetworkResponse, bool) {
	getURL := fmt.Sprintf(
		config.HostAgentSettings.ConnectionStrings.GetNetworkConfigurationURLFmt,
		config.HostAgentSettings.WireServerIP,
		networkID)

	getResponse, getRespErr := httpClient.Get(getURL)
	if getRespErr != nil {
		msg := fmt.Sprintf(
			"Retrieval of network configuration for network [%s] failed with error [%s]. Marking this error as retriable.",
			networkID,
			getRespErr.Error())
		return true, &api.JsonErr{
			Code:       api.NetworkConfigurationRetrievalFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil, false
	}

	defer getResponse.Body.Close()

	if getResponse.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Get network configuration call for network %s failed with status[%d].",
			networkID,
			getResponse.StatusCode)
		return true, &api.JsonErr{
			Code:       api.NetworkJoinFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil, getResponse.StatusCode == http.StatusForbidden
	}

	var networkConfigurationResponse nmagentcontracts.VirtualNetworkResponse
	decodeRespErr := utils.DecodeResponse(getResponse, &networkConfigurationResponse)
	if decodeRespErr != nil {
		msg := fmt.Sprintf("Failed to decode get virtual network configuration response for network %s due to error %s. "+
			"Marking this error as retriable.",
			networkID,
			decodeRespErr.Error())
		return true, &api.JsonErr{
			Code:       api.DecodeResponseFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil, false
	}

	logger.Debugf("[NMAgent][Response] for network %s configuration:%s", networkID, networkConfigurationResponse.String())
	getNetworkResponseCode, parseErr := strconv.ParseInt(networkConfigurationResponse.ResponseCode, 10, 64)
	if parseErr != nil {
		msg := fmt.Sprintf(
			"Failed to convert response code %s  to integer when parsing network configuration response for network %s. "+
				"Marking this error as retriable.",
			networkConfigurationResponse.ResponseCode,
			networkID)
		return true, &api.JsonErr{
			Code:       api.ParseResponseCodeFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil, false
	}

	if getNetworkResponseCode == http.StatusProcessing {
		msg := fmt.Sprintf(
			"GetNetworkConfiguration for network %s is pending", networkID)
		return true, &api.JsonErr{
			Code:       api.ProcessingPending,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil, false
	}

	if getNetworkResponseCode == http.StatusUnauthorized {
		msg := fmt.Sprintf("Join for network %s is unauthorized", networkID)
		return false, &api.JsonErr{
			Code:       api.NetworkJoinFailed,
			Text:       msg,
			HTTPStatus: http.StatusUnauthorized}, nil, false
	}

	if getNetworkResponseCode != http.StatusOK {
		msg := fmt.Sprintf("Failed to retrieve network due to error %d", getNetworkResponseCode)
		return true, &api.JsonErr{
			Code:       api.NetworkJoinFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil, false
	}

	return false, nil, &networkConfigurationResponse, false
}

// Post request to host nmagent to express the intent for joining a network.
// Returns IsRetriable, optional error if any, isThrottled
func postRequestToAgentForNetworkJoin(networkID string) (bool, *api.JsonErr, bool) {
	joinURL := fmt.Sprintf(
		config.HostAgentSettings.ConnectionStrings.JoinNetworkURLFmt,
		config.HostAgentSettings.WireServerIP,
		networkID)

	loggershim.LS.Debugf("[NMAgent] joining network [%s]", networkID)
	// Empty body is required as wireserver cannot handle a post without the body.
	var body bytes.Buffer
	json.NewEncoder(&body).Encode("")
	joinResponse, joinRespErr := httpClient.Post(joinURL, "application/json", &body)

	return DecodeNetworkJoinResponse(networkID, joinResponse, joinRespErr)
}

// Decode the response received for join network request.
// Returns IsRetirbale, optional error if any, isThrottled
func DecodeNetworkJoinResponse(
	networkID string,
	joinResponse *http.Response,
	joinErr error) (bool, *api.JsonErr, bool) {
	if joinErr != nil {
		msg := fmt.Sprintf(
			"Join for network [%s] failed with reponse error [%s]. Marking this error as retriable",
			networkID,
			joinErr.Error())
		return true, &api.JsonErr{
			Code:       api.JoinDenied,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, false
	}

	defer joinResponse.Body.Close()

	if joinResponse.StatusCode == http.StatusProcessing {
		msg := fmt.Sprintf(
			"Join for network [%s] is pending. Marking this error as retriable",
			networkID)
		return true, &api.JsonErr{
			Code:       api.JoinPending,
			Text:       msg,
			HTTPStatus: http.StatusProcessing}, false
	}

	if joinResponse.StatusCode == http.StatusUnauthorized {
		msg := fmt.Sprintf("Join for network [%s] is unauthorized. Marking this error as non retriable.", networkID)
		return false, &api.JsonErr{
			Code:       api.NetworkJoinFailed,
			Text:       msg,
			HTTPStatus: http.StatusUnauthorized}, false
	}

	if joinResponse.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Join for network [%s] failed with status [%d]. Marking this error as retriable.",
			networkID,
			joinResponse.StatusCode)
		return true, &api.JsonErr{
			Code:       api.NetworkJoinFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, joinResponse.StatusCode == http.StatusForbidden
	}

	return false, nil, false
}

func sendLeakedNCMetric(networkID, subnetName string, ncCount int) {
	metric := aitelemetry.Metric{
		Name:             aiwrapper.LeakedNCEventMetric,
		Value:            float64(ncCount),
		CustomDimensions: make(map[string]string),
	}

	metric.CustomDimensions[aiwrapper.NetworkIDStr] = networkID
	metric.CustomDimensions[aiwrapper.SubNetStr] = subnetName
	go aiwrapper.SendMetric(metric)
}
