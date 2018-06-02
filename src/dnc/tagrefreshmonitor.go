package main

import (
	"dnc/api"
	"dnc/datamodel"
	"dnc/nmagentcontracts"
	"dnc/utils"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var tagsCache = make(map[string]datamodel.TagDefinition)
var tagsCacheLock sync.RWMutex
var tagsCacheRehydrated = false

func monitorTagUpdates() {
	monitorTimerChannel := time.NewTicker(time.Millisecond *
		time.Duration(config.DNCSettings.NetworkRefreshInterval)).C
	for {
		select {
		case <-monitorTimerChannel:
			if replicaInstance.IsReplicaPrimary() {
				refreshTagDefintions()
			}
			break
		case <-tagMonitorStartProcessing:
			refreshTagDefintions()
			logger.Debugf("Received signal to process tag updates\n")
			break
		case <-tagMonitorStopProcessing:
			logger.Debugf("Received signal to exit\n")
			break
		}
	}
}

func refreshTagDefintions() {
	tagSessionInfo := map[string]string{datamodel.TagRepositoryName: datamodel.TagRepositoryName}
	tagsToUpdate := make(map[string]nmagentcontracts.TagResponse)

	hydrateTagsCache()

	// Resolve the tags by calling in to Agent.
	for _, repoTagDefinition := range tagsCache {
		_, failureDetail, tagResponse := getSystemTagDefinitionFromAgent(
			repoTagDefinition.Name,
			nmagentcontracts.IPv4)

		// Log and move on to resolving the next tag after logging an error.
		if failureDetail != nil {
			logger.Errorf("Request to get definition for tag %s failed with error %s.\n",
				repoTagDefinition.Name,
				failureDetail.String())
			continue
		}

		// Cache the tag and the value if it was never resolved before (or)
		// the values has changed.
		if len(repoTagDefinition.Values) == 0 ||
			!utils.AreSetsEqual(repoTagDefinition.Values, tagResponse.Values) {
			tagDefinitionKey := datamodel.GetTagDefintionKey(repoTagDefinition.Type, repoTagDefinition.Name)
			tagsToUpdate[tagDefinitionKey] = *tagResponse
		}
	}

	// Open the tag repsository and upate the resolved tags.
	sessionManager.BeginWriteSession(tagSessionInfo)

	for tagKey, tagResponse := range tagsToUpdate {
		repoTagDefinition := tagsCache[tagKey]
		repoTagDefinition.Values = tagResponse.Values
		repoTagDefinition.Version++

		// Update the tag in memory cache with the resolved tag values.
		updateTagsCacheWithDefinition(repoTagDefinition)

		err := datamodel.CreateOrUpdateTagDefintion(
			repoTagDefinition.Type,
			repoTagDefinition.Name,
			tagResponse.Values)
		if err != nil {
			logger.Errorf("Failed to update tag definition for tag Response:%s due to error %s",
				tagResponse.String(),
				err.String())
		} else {
			logger.Debugf("Successfully updated tag %s", tagResponse.Name)
		}
	}

	sessionManager.EndWriteSession(tagSessionInfo)
}

// RegisterAndGetTagDefinition - Register and get tag definition.
func RegisterAndGetTagDefinition(tag nmagentcontracts.Tag) (datamodel.TagDefinition, *api.JsonErr) {
	tagDefinitionKey := datamodel.GetTagDefintionKey(tag.Type, tag.Name)

	tagsCacheLock.RLock()
	currentRepoTagDefinition, found := tagsCache[tagDefinitionKey]
	if found && currentRepoTagDefinition.Version > 0 {
		tagsCacheLock.RUnlock()
		return currentRepoTagDefinition, nil
	}
	tagsCacheLock.RUnlock()

	tagDefinition := datamodel.TagDefinition{
		Type:    tag.Type,
		Name:    tag.Name,
		Version: 0}
	updateTagsCacheWithDefinition(tagDefinition)

	return datamodel.TagDefinition{}, &api.JsonErr{
		Text:       fmt.Sprintf("Resolving definition for tag %s is pending", tag.String()),
		Code:       api.TagResolutionPending,
		HTTPStatus: http.StatusProcessing}
}

func hydrateTagsCache() *api.JsonErr {
	tagsCacheLock.RLock()
	if tagsCacheRehydrated {
		tagsCacheLock.RUnlock()
		return nil
	}
	tagsCacheLock.RUnlock()

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
		logger.Errorf("Retrieval of tag defintitions failed with error:%s.\n",
			err.Error())
		return &api.JsonErr{
			Code:       api.RepositoryTagRetrievalFailed,
			Text:       fmt.Sprintf("Failed to retrieve tags from repository due to error %s", err.Error()),
			HTTPStatus: http.StatusInternalServerError}
	}
	sessionManager.EndReadSession(sessionInfo)
	sessionLocked = false

	tagsCacheLock.Lock()
	defer tagsCacheLock.Unlock()
	for tagDefintitionKey, tagDefintiion := range repoTagsDefinitionMap {
		tagsCache[tagDefintitionKey] = tagDefintiion
	}

	tagsCacheRehydrated = true

	return nil
}

func registerTagForProcessing(tag nmagentcontracts.Tag) {
	tagDefinitionKey := datamodel.GetTagDefintionKey(tag.Type, tag.Name)

	tagsCacheLock.RLock()
	_, found := tagsCache[tagDefinitionKey]
	tagsCacheLock.RUnlock()

	if found {
		return
	}

	tagDefinition := datamodel.TagDefinition{
		Type:    tag.Type,
		Name:    tag.Name,
		Version: 0}
	updateTagsCacheWithDefinition(tagDefinition)
}

func updateTagsCacheWithDefinition(tagDefinition datamodel.TagDefinition) {
	tagDefinitionKey := datamodel.GetTagDefintionKey(tagDefinition.Type, tagDefinition.Name)
	tagsCacheLock.Lock()
	defer tagsCacheLock.Unlock()
	currentRepoTagDefinition, found := tagsCache[tagDefinitionKey]
	if !found || currentRepoTagDefinition.Version < tagDefinition.Version {
		tagsCache[tagDefinitionKey] = tagDefinition
	}
}

func checkAndAddTagsCache(networkConfiguration *nmagentcontracts.VirtualNetworkResponse) {
	for _, subnetConfiguration := range networkConfiguration.Subnets {
		for _, tag := range subnetConfiguration.Tags {
			registerTagForProcessing(tag)
		}
	}
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

	defer getResponse.Body.Close()

	var tagResponse nmagentcontracts.TagResponse
	decodeRespErr := utils.DecodeResponse(getResponse, &tagResponse)
	if decodeRespErr != nil {
		msg := fmt.Sprintf("Failed to decode tag response for tag %s due to error %s. "+
			"Marking this error as retriable.",
			tagName,
			decodeRespErr.Error())
		logger.Errorf("%s\n", msg)

		return true, &api.JsonErr{
			Code:       api.DeocodeResponseFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil
	}

	logger.Debugf("Received response for tag %s Defintion:%s\n", tagName, tagResponse.String())
	getTagResponseCode, parseErr := strconv.ParseInt(tagResponse.ResponseCode, 10, 64)
	if parseErr != nil {
		msg := fmt.Sprintf(
			"Failed to convert response code %s  to integer when parsing tag definition response for tag %s. "+
				"Marking this error as retriable.",
			tagResponse.ResponseCode,
			tagName)
		logger.Errorf("%s\n", msg)

		return true, &api.JsonErr{
			Code:       api.ParseResponseCodeFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil
	}

	if getTagResponseCode == http.StatusProcessing {
		msg := fmt.Sprintf(
			"GetTagDefinition for tag %s is in pending state", tagName)
		logger.Debugf("%s\n", msg)

		return true, &api.JsonErr{
			Code:       api.ProcessingPending,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil
	}

	if getTagResponseCode == http.StatusUnauthorized {
		msg := fmt.Sprintf("Get tag definition for tag %s is unauthorized", tagName)
		logger.Debugf("%s\n", msg)

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

	logger.Debugf("Retrieved tag definition %s\n", tagResponse.String())
	return false, nil, &tagResponse
}

func retrieveAndPersistTagDefinition(tagType, tagName string) (bool, *api.JsonErr) {
	isRetriableError, failureDetail, tagResponse := getSystemTagDefinitionFromAgent(tagName, nmagentcontracts.IPv4)
	if failureDetail != nil {
		logger.Errorf("Request to get definition for tag %s failed with error %s.\n",
			tagName,
			failureDetail.String())
		return isRetriableError, failureDetail
	}

	return true, datamodel.CreateOrUpdateTagDefintion(tagType, tagName, tagResponse.Values)
}
