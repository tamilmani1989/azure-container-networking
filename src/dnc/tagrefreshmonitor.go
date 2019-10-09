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

var (
	tagsCache                 = make(map[string]datamodel.TagDefinition)
	tagsCacheLock             sync.RWMutex
	tagsCacheRehydrated       = false
	defaultTagRefreshInterval = 30000
)

func monitorTagUpdates() {
	if config.DNCSettings.TagRefreshInterval == 0 {
		config.DNCSettings.TagRefreshInterval = defaultTagRefreshInterval
	}

	monitorTimerChannel := time.NewTicker(time.Millisecond *
		time.Duration(config.DNCSettings.TagRefreshInterval)).C
	for {
		select {
		case <-monitorTimerChannel:
			if replicaInstance.IsReplicaPrimary() {
				refreshTagDefinitions()
			}
			break
		case <-tagMonitorStartProcessing:
			refreshTagDefinitions()
			break
		case <-tagMonitorStopProcessing:
			break
		}
	}
}

func refreshTagDefinitions() {
	if err := hydrateTagsCache(); err != nil {
		logger.Errorf("%+v", err)
	}
	// Resolve the tags by calling in to Agent.
	for _, repoTagDefinition := range tagsCache {
		if _, err := RefreshTag(nmagentcontracts.Tag{Name: repoTagDefinition.Name, Type: repoTagDefinition.Type}, repoTagDefinition); err != nil {
			logger.Errorf("%+v", err)
		}
	}
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

	return RefreshTag(tag, tagDefinition)
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
	getResponse, getRespErr := httpClient.Get(getURL)
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

		return true, &api.JsonErr{
			Code:       api.DecodeResponseFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil
	}

	getTagResponseCode, parseErr := strconv.ParseInt(tagResponse.ResponseCode, 10, 64)
	if parseErr != nil {
		msg := fmt.Sprintf(
			"Failed to convert response code %s  to integer when parsing tag definition response for tag %s. "+
				"Marking this error as retriable.",
			tagResponse.ResponseCode,
			tagName)

		return true, &api.JsonErr{
			Code:       api.ParseResponseCodeFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}, nil
	}

	if getTagResponseCode == http.StatusProcessing {
		msg := fmt.Sprintf(
			"GetTagDefinition for tag %s is in pending state", tagName)

		return true, &api.JsonErr{
			Code:       api.TagResolutionPending,
			Text:       msg,
			HTTPStatus: http.StatusProcessing}, nil
	}

	if getTagResponseCode == http.StatusUnauthorized {
		msg := fmt.Sprintf("Get tag definition for tag %s is unauthorized", tagName)

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

	return false, nil, &tagResponse
}

// RefreshTag queries NMAgent for tag definitions and returns either the retrieved definition
// or an empty definition with JsonErr recorded.
func RefreshTag(tag nmagentcontracts.Tag, tagDefinition datamodel.TagDefinition) (datamodel.TagDefinition, *api.JsonErr) {
	_, err, tagResponse := getSystemTagDefinitionFromAgent(
		tag.Name,
		nmagentcontracts.IPv4)
	if err != nil {
		return datamodel.TagDefinition{}, err
	}

	// Cache the tag and the value if it was never resolved before (or)
	// the values has changed.
	if !utils.AreSetsEqual(tagDefinition.Values, tagResponse.Values) {
		tagDefinition.Values = tagResponse.Values
		tagDefinition.Version++
		// Update the tag in memory cache with the resolved tag values.
		updateTagsCacheWithDefinition(tagDefinition)

		// Update repository tag definition
		tagSessionInfo := map[string]string{datamodel.TagRepositoryName: datamodel.TagRepositoryName}
		sessionManager.BeginWriteSession(tagSessionInfo)
		err = datamodel.CreateOrUpdateTagDefintion(
			tagDefinition.Type,
			tagDefinition.Name,
			tagResponse.Values)
		sessionManager.EndWriteSession(tagSessionInfo)
	}

	return tagDefinition, err
}
