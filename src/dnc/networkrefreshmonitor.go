package main

import (
	"dnc/api"
	"dnc/datamodel"
	"dnc/nmagentcontracts"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var tagSubnetMap = make(map[string]map[string]string)

var processedNetworks = make(map[string]bool)
var tagsToProcess = make(map[string]datamodel.TagDefinition)
var tagArtifactsLock sync.RWMutex

func monitorNetworkUpdates() {
	monitorTimerChannel := time.NewTicker(time.Millisecond * 1000).C
	for {
		select {
		case <-monitorTimerChannel:
			if replicaInstance.IsReplicaPrimary() {
				refreshNetworks()
			}
			break
		case <-networkMonitorStartProcessing:
			refreshNetworks()
			logger.Log("Received signal to process pending work\n")
			break
		case <-networkMonitorStopProcessing:
			logger.Log("Received signal to exit\n")
			break
		}
	}
}

func monitorTagUpdates() {
	monitorTimerChannel := time.NewTicker(time.Millisecond * 1000).C
	for {
		select {
		case <-monitorTimerChannel:
			if replicaInstance.IsReplicaPrimary() {
				refreshTagDefintions()
			}
			break
		case <-tagMonitorStartProcessing:
			refreshTagDefintions()
			logger.Log("Received signal to process tag updates\n")
			break
		case <-tagMonitorStopProcessing:
			logger.Log("Received signal to exit\n")
			break
		}
	}
}

func refreshNetworks() {
	networksMap, err := datamodel.GetAllNetworkIDs()
	if err != nil {
		logger.Log("Failed to retrieve networks from repository due to error %s", err.Error())
	}
	for networkKey := range networksMap {
		refreshNetworkConfiguration(networkKey)
	}
}

func InitializeTagArtifacts() {
	tagSubnetMap = make(map[string]map[string]string)
	processedNetworks = make(map[string]bool)
	tagsToProcess = make(map[string]datamodel.TagDefinition)
}

func PopulateTagsToProcess() *api.JsonErr {
	sessionInfo := map[string]string{datamodel.TagRepositoryName: datamodel.TagRepositoryName}
	sessionLocked := false
	sessionManager.BeginReadSession(sessionInfo)
	sessionLocked = true
	defer func() {
		if sessionLocked {
			sessionManager.EndReadSession(sessionInfo)
		}
	}()
	repoTagsDefinitionMap, err := datamodel.GetAllTagDefinitions()
	if err != nil {
		logger.Log("Retrieval of tag defintitions failed with error:%s.\n",
			err.Error())
		return &api.JsonErr{
			Code:       api.RepositoryTagRetrievalFailed,
			Text:       fmt.Sprintf("Failed to retriev tags from repository due to error %s", err.Error()),
			HTTPStatus: http.StatusInternalServerError}
	}
	sessionManager.EndReadSession(sessionInfo)
	sessionLocked = false

	tagArtifactsLock.Lock()
	defer tagArtifactsLock.Unlock()
	for tagDefintitionKey, tagDefintiion := range repoTagsDefinitionMap {
		tagsToProcess[tagDefintitionKey] = tagDefintiion
	}

	return nil
}

func RegisterAndGetTagDefinition(tag nmagentcontracts.Tag) (string, *api.JsonErr) {
	tagDefinitionKey := datamodel.GetTagDefintionKey(tag.Type, tag.Name)
	readLockAcquired := false
	tagArtifactsLock.RLock()
	readLockAcquired = true

	defer func() {
		if readLockAcquired {
			tagArtifactsLock.RUnlock()
		}
	}()
	currentRepoTagDefinition, found := tagsToProcess[tagDefinitionKey]
	if found && currentRepoTagDefinition.Version > 0 {
		return currentRepoTagDefinition.Value, nil
	}

	tagArtifactsLock.RUnlock()
	readLockAcquired = false
	tagDefinition := datamodel.TagDefinition{
		Type:    tag.Type,
		Name:    tag.Name,
		Version: 0}
	AddTagToProcess(tagDefinition)

	return "", &api.JsonErr{
		Text:       fmt.Sprintf("Resolving definition for tag %s is pending", tag.String()),
		Code:       api.TagResolutionPending,
		HTTPStatus: http.StatusProcessing}
}

func AddTagToProcess(tagDefinition datamodel.TagDefinition) {
	tagDefinitionKey := datamodel.GetTagDefintionKey(tagDefinition.Type, tagDefinition.Name)
	tagArtifactsLock.Lock()
	defer tagArtifactsLock.Unlock()
	currentRepoTagDefinition, found := tagsToProcess[tagDefinitionKey]
	if !found || currentRepoTagDefinition.Version < tagDefinition.Version {
		tagsToProcess[tagDefinitionKey] = tagDefinition
	}
}

func CheckAndAddNetwork(repoNetworkConfiguration *datamodel.NetworkConfiguration, refresh bool) *api.JsonErr {
	logger.Log("Taking artifacts lock")
	tagArtifactsLock.Lock()
	defer func() {
		logger.Log("Exisiting CheckAndAddNetwork")
		tagArtifactsLock.Unlock()
	}()
	logger.Log("Acquried artifacts lock")
	found := processedNetworks[repoNetworkConfiguration.NetworkID]
	if found && !refresh {
		return nil
	}

	// Retrieve the current subnets from the repository.
	for subnetName := range repoNetworkConfiguration.Subnets {
		subnetConfiguration, subnetCfgErr := datamodel.QuerySubnetConfiguration(repoNetworkConfiguration.NetworkID, subnetName)
		if subnetCfgErr != nil {
			logger.Log(
				"Failed to retrieve subnet configuration for subnet %s in network %s due to error %s\n",
				subnetName,
				repoNetworkConfiguration.NetworkID,
				subnetCfgErr)
			return subnetCfgErr
		}

		if len(subnetConfiguration.Tags) > 0 {
			for _, subnetMap := range tagSubnetMap {
				delete(subnetMap, subnetName)
			}

			for _, tag := range subnetConfiguration.Tags {
				tagKey := datamodel.GetTagDefintionKey(tag.Type, tag.Name)
				subnetMap, found := tagSubnetMap[tagKey]
				if !found {
					subnetMap = make(map[string]string)
					tagSubnetMap[tagKey] = subnetMap
				}
				subnetMap[subnetName] = subnetConfiguration.NetworkID
			}
		}
	}
	processedNetworks[repoNetworkConfiguration.NetworkID] = true
	return nil
}

func refreshNetworkConfiguration(networkKey string) {
	logger.Log("Inside refreshNetworkConfiguration for network Id:%s", networkKey)
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
		logger.Log("Retrieval of configuration for network %s failed with error:%s.\n",
			networkID,
			err.String())
		return
	} else if repoNetworkConfiguration == nil {
		logger.Log("Configuration for Network %s not found.\n", networkID)
		return
	}

	if repoNetworkConfiguration.NetworkState != datamodel.Join ||
		repoNetworkConfiguration.Status != datamodel.Completed {
		logger.Log("Skipping refreshing of network %s as it is in state:%s %s.\n",
			networkID,
			repoNetworkConfiguration.NetworkState,
			repoNetworkConfiguration.Status)
		return
	}

	sessionManager.EndReadSession(sessionInfo)
	networkSessionLockedForRead = false

	_, failureDetail = postRequestToAgentForNetworkJoin(networkID)
	logger.Log("Post network config")
	if failureDetail != nil {
		logger.Log("Post request Network %s to Host Agent failed with error %s.\n",
			networkID,
			failureDetail.String())

		return
	}

	logger.Log("Get network config\n")
	_, failureDetail, networkResponse = getNetworkConfigurationFromAgent(networkID)
	if failureDetail != nil {
		logger.Log("Request to get configuration for network %s failed with error %s.\n",
			networkID,
			failureDetail.String())
		return
	}

	networkVersion, parseNetworkVersionErr := strconv.ParseUint(networkResponse.VnetVersion, 10, 64)
	if parseNetworkVersionErr != nil {
		logger.Log("Failed to parse network version:%s for network:%s failed with error %s.\n",
			networkResponse.VnetVersion,
			networkID,
			parseNetworkVersionErr.Error())
		return
	}

	logger.Log("Lock session for write\n")
	networkSessionLockedForWrite := false
	sessionManager.BeginWriteSession(sessionInfo)
	networkSessionLockedForWrite = true
	defer func() {
		logger.Log("networkSessionLockedForWrite:%v", networkSessionLockedForWrite)
		if networkSessionLockedForWrite {
			sessionManager.EndWriteSession(sessionInfo)
		}
	}()

	repoNetworkConfiguration, err = datamodel.TryQueryNetworkConfigurationByKey(networkKey)
	if err != nil {
		logger.Log("Retrieval of configuration for network %s failed with error:%s.\n",
			networkID,
			err.String())
		return
	} else if repoNetworkConfiguration == nil {
		logger.Log("Configuration for Network %s not found.\n", networkID)
		return
	}

	if repoNetworkConfiguration.NetworkState != datamodel.Join ||
		repoNetworkConfiguration.Status != datamodel.Completed {
		logger.Log("Skipping refreshing of network %s as it is in state:%s %s.\n",
			networkID,
			repoNetworkConfiguration.NetworkState,
			repoNetworkConfiguration.Status)
		return
	}

	if repoNetworkConfiguration.Version == networkVersion {
		logger.Log("CheckAndAddNetwork\n")
		CheckAndAddNetwork(repoNetworkConfiguration, false)
		return
	}

	_, failureDetail, subnetsToAdd, subnetsToRemove, subnetsToUpdate :=
		validateAndRetrieveNetworkConfigurationUpdate(
			repoNetworkConfiguration,
			networkResponse)

	if failureDetail != nil {
		logger.Log("Validation of response %v for network %s failed with error %s.\n",
			networkResponse,
			networkID,
			failureDetail.String())
		return
	}

	for subnetName := range subnetsToRemove {
		logger.Log("Deleting subnet %s in network %s failed with error %s.\n",
			subnetName,
			networkID)
		failureDetail = datamodel.DeleteSubnetConfiguration(networkID, subnetName)
		if failureDetail != nil {
			logger.Log("Deletion of subnet %s in network %s failed with error %s.\n",
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
			logger.Log("Failed to retrieve network containers in subnet %s Network %s due to error %s",
				subnetName,
				networkID,
				err.String())
		}
	}

	logger.Log("Updating vnetconfig with current version %d Phase 1 commit Response:%v\n",
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
		logger.Log("Persisting configuration of network %s failed with error %s. \n",
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
			logger.Log("Persisting configuration for subnet %s in network %s failed with error %s. \n",
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
			logger.Log("Persisting updated configuration for subnet %s in network %s failed with error %s. \n",
				subnetName,
				networkID,
				failureDetail.String())
			return
		}

		failureDetail = UpdateAssociationStateForNCsInSubnet(networkID, subnetKey)
		if failureDetail != nil {
			logger.Log("Failed to update association info for network containers in subnet %s in network %s. Error %s. \n",
				subnetName,
				networkID,
				failureDetail.String())
			return
		}
	}

	logger.Log("Updating vnetconfig with received version %d Phase 2 commit Response:%s\n",
		repoNetworkConfiguration.Version,
		networkResponse.String())

	repoNetworkConfiguration.Version = networkVersion

	CheckAndAddNetwork(repoNetworkConfiguration, true)
	failureDetail = datamodel.PersistNetworkConfiguration(repoNetworkConfiguration)
	if failureDetail != nil {
		logger.Log("Persisting configuration of network %s failed with error %s. \n",
			networkID,
			failureDetail.String())
	}

	logger.Log("Successfully refreshed configuration for network %s", networkID)
}

// Retrieve network configuration from host agent.
func getSystemTagDefinitionFromAgent(tagName, addressType string) (bool, *api.JsonErr, *nmagentcontracts.TagResponse) {
	getURL := fmt.Sprintf(
		config.HostAgentSettings.ConnectionStrings.GetSystemTagDefinitionURLFmt,
		config.HostAgentSettings.WireServerIP,
		tagName,
		addressType)
	getResponse, getRespErr := http.Get(getURL)
	if getRespErr != nil {
		msg := fmt.Sprintf(
			"Retrieval of system tag definition for tag [%s] failed with error [%s]. Marking this error as retriable.",
			tagName,
			getRespErr.Error())
		return true, &api.JsonErr{
			Code:       api.AgentTagDefinitionRetrievalFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil
	}

	var tagResponse nmagentcontracts.TagResponse
	decodeRespErr := decodeResponse(getResponse, &tagResponse)
	if decodeRespErr != nil {
		msg := fmt.Sprintf("Failed to decode tag response for tag %s due to error %s. "+
			"Marking this error as retriable.",
			tagName,
			decodeRespErr.Error())
		logger.Log("%s\n", msg)

		return true, &api.JsonErr{
			Code:       api.DeocodeResponseFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil
	}

	logger.Log("Received response for tag %s Defintion:%s\n", tagName, tagResponse.String())
	getTagResponseCode, parseErr := strconv.ParseInt(tagResponse.ResponseCode, 10, 64)
	if parseErr != nil {
		msg := fmt.Sprintf(
			"Failed to convert response code %s  to integer when parsing tag definition response for tag %s. "+
				"Marking this error as retriable.",
			tagResponse.ResponseCode,
			tagName)
		logger.Log("%s\n", msg)

		return true, &api.JsonErr{
			Code:       api.ParseResponseCodeFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil
	}

	if getTagResponseCode == http.StatusProcessing {
		msg := fmt.Sprintf(
			"GetTagDefinition for tag %s is in pending state", tagName)
		logger.Log("%s\n", msg)

		return true, &api.JsonErr{
			Code:       api.ProcessingPending,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil
	}

	if getTagResponseCode == http.StatusUnauthorized {
		msg := fmt.Sprintf("Get tag definition for tag %s is unauthorized", tagName)
		logger.Log("%s\n", msg)

		return false, &api.JsonErr{
			Code:       api.GetTagDefinitionFailed,
			Text:       msg,
			HTTPStatus: http.StatusForbidden}, nil
	}

	if getTagResponseCode != http.StatusOK {
		msg := fmt.Sprintf("Failed to retrieve tag defintition for tag %s due to error %d", tagName, getTagResponseCode)
		return false, &api.JsonErr{
			Code:       api.GetTagDefinitionFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil
	}

	logger.Log("Retrieved tag definition %s\n", tagResponse.String())
	return false, nil, &tagResponse
}

func RetrieveAndPersistTagDefinition(tagType, tagName string) (bool, *api.JsonErr) {
	isRetriableError, failureDetail, tagResponse := getSystemTagDefinitionFromAgent(tagName, "IPv4")
	if failureDetail != nil {
		logger.Log("Request to get definition for tag %s failed with error %s.\n",
			tagName,
			failureDetail.String())
		return isRetriableError, failureDetail
	}

	return true, datamodel.CreateOrUpdateTagDefintion(tagType, tagName, tagResponse.Value)
}

func refreshTagDefintions() {
	tagSessionInfo := map[string]string{datamodel.TagRepositoryName: datamodel.TagRepositoryName}
	tagsToUpdate := make(map[string]nmagentcontracts.TagResponse)

	err := PopulateTagsToProcess()
	if err != nil {
		logger.Log("Failed to populate tags to process due to error %s", err.String())
	}

	for _, repoTagDefinition := range tagsToProcess {
		_, failureDetail, tagResponse := getSystemTagDefinitionFromAgent(repoTagDefinition.Name, "IPv4")
		if failureDetail != nil {
			logger.Log("Request to get definition for tag %s failed with error %s.\n",
				repoTagDefinition.Name,
				failureDetail.String())
			continue
		}

		if len(repoTagDefinition.Value) == 0 || repoTagDefinition.Value != tagResponse.Value {
			tagDefinitionKey := datamodel.GetTagDefintionKey(repoTagDefinition.Type, repoTagDefinition.Name)
			tagsToUpdate[tagDefinitionKey] = *tagResponse
		}
	}

	tagSessionOpenedForWrite := false
	sessionManager.BeginWriteSession(tagSessionInfo)
	tagSessionOpenedForWrite = true
	defer func() {
		if tagSessionOpenedForWrite {
			sessionManager.EndWriteSession(tagSessionInfo)
		}
	}()
	for tagKey, tagResponse := range tagsToUpdate {
		repoTagDefinition := tagsToProcess[tagKey]
		repoTagDefinition.Value = tagResponse.Value
		repoTagDefinition.Version++
		AddTagToProcess(repoTagDefinition)

		err := datamodel.CreateOrUpdateTagDefintion(
			repoTagDefinition.Type,
			tagResponse.Name,
			tagResponse.Value)
		if err != nil {
			logger.Log("Failed to update tag definition for tag Response:%s due to error %s",
				tagResponse.String(),
				err.String())
		} else {
			logger.Log("Successfully updated tag %s", tagResponse.Name)
		}
	}

	sessionManager.EndWriteSession(tagSessionInfo)
	tagSessionOpenedForWrite = false
	UpdateSubnetsForTag(tagsToUpdate)
}

func UpdateSubnetsForTag(tagsToUpdate map[string]nmagentcontracts.TagResponse) {
	tagArtifactsLockAcquired := false
	tagArtifactsLock.RLock()
	tagArtifactsLockAcquired = true
	defer func() {
		if tagArtifactsLockAcquired {
			tagArtifactsLock.RUnlock()
		}
	}()
	subnetsToUpdate := make(map[string]string)
	for tagKey, _ := range tagsToUpdate {
		subnetMap, found := tagSubnetMap[tagKey]
		if !found {
			continue
		}

		for subnetKey, networkID := range subnetMap {
			subnetsToUpdate[subnetKey] = networkID
		}
	}
	tagArtifactsLock.RUnlock()
	tagArtifactsLockAcquired = false
	for subnetKey, networkID := range subnetsToUpdate {
		UpdateAssociationStateForNCsInSubnetNetworkTransacted(networkID, subnetKey)
	}
}

func UpdateAssociationStateForNCsInSubnetNetworkTransacted(networkID, subnetKey string) *api.JsonErr {
	logger.Log("Inside UpdateAssociationStateForNCsInSubnetNetworkTransacted")
	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	sessionManager.BeginWriteSession(sessionInfo)
	defer sessionManager.EndWriteSession(sessionInfo)
	return UpdateAssociationStateForNCsInSubnet(networkID, subnetKey)
}

func CreateAssociationTask(networkContainerInfo *datamodel.NetworkContainerInfo) *api.JsonErr {
	logger.Log("Inside check association")
	repoNetworkContainerConfiguration, err := datamodel.TryQueryNetworkContainer(
		networkContainerInfo.NetworkID,
		networkContainerInfo.NetworkContainerID)

	if err != nil {
		logger.Log("Failed to retrieve configuration for network container %s due to error %s",
			networkContainerInfo.NetworkContainerID,
			err.String())
		return err
	}

	if repoNetworkContainerConfiguration == nil {
		logger.Log("Configuration for network container %s not found.",
			networkContainerInfo.NetworkContainerID)
		return nil
	}

	if repoNetworkContainerConfiguration.MarkedForDelete {
		return nil
	}

	if repoNetworkContainerConfiguration.Status == datamodel.Failed ||
		repoNetworkContainerConfiguration.Status == datamodel.Denied {
		return nil
	}

	createTask := false
	if repoNetworkContainerConfiguration.Status == datamodel.Pending &&
		(repoNetworkContainerConfiguration.State == datamodel.Publish ||
			repoNetworkContainerConfiguration.State == datamodel.Program) {
		repoNetworkContainerConfiguration.State = datamodel.Associate
		createTask = true
	}

	if repoNetworkContainerConfiguration.State == datamodel.Program &&
		repoNetworkContainerConfiguration.Status == datamodel.Completed {
		repoNetworkContainerConfiguration.State = datamodel.RefreshAssociation
		createTask = true
	}

	if createTask {
		repoNetworkContainerConfiguration.CNSVersion++
		return datamodel.PersistNCConfiguration(repoNetworkContainerConfiguration, true)
	}

	return nil
}

func UpdateAssociationStateForNCsInSubnet(networkID string, subnetKey string) *api.JsonErr {
	repoSubnetConfiguration, err := datamodel.TryQuerySubnetConfigurationByKey(subnetKey)
	if err != nil {
		logger.Log("Retrieval of subnet configuration for subnet key %s failed due to error %s", subnetKey, err.String())
		return err
	}
	if repoSubnetConfiguration == nil {
		return nil
	}

	networkContainersInfo, err := datamodel.GetNetworkContainersInSubnet(repoSubnetConfiguration, false)
	if err != nil {
		logger.Log("Retrieval of network containers in subnet configuration %s failed due to error %s",
			repoSubnetConfiguration.String(),
			err.String())
		return err
	}

	for _, networkContainerInfo := range networkContainersInfo {
		err := CreateAssociationTask(networkContainerInfo)
		if err != nil {
			return err
		}
	}

	return nil
}
