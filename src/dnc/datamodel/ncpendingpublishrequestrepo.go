package datamodel

import (
	"dnc/api"
	"dnc/datastoreshim"
	"encoding/json"
	"net/http"
)

type NcPendingPublishRequestRepo struct {
	dataStore datastoreshim.DataStoreShim
}

func NewNcPendingPublishRequestRepo(dataStore datastoreshim.DataStoreShim) *NcPendingPublishRequestRepo {
	return &NcPendingPublishRequestRepo{
		dataStore: dataStore,
	}
}

// RemoveNcPendingPublishContext - Remove pending NC publish / unpublish context from the table.
func (p *NcPendingPublishRequestRepo) RemoveNcPendingPublishContext(
	networkID, networkContainerID string,
	pubsubAzId string) *api.JsonErr {
	key := GetNcPendingPublishContextKey(networkID, networkContainerID, pubsubAzId)
	if err := p.dataStore.Delete(NcPendingPublishContextTableName, key); err != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError,
			"could not delete pending pubsub request: %s. Error: %v", key, err)
	}

	return nil
}

// TryGetPendingCleanupRequest - Query a network container by key.
func (p *NcPendingPublishRequestRepo) TryGetNcPendingPublish(requestKey string) (*NcPendingPublishRequestContext, *api.JsonErr) {
	entity, status, err := p.dataStore.Query(NcPendingPublishContextTableName, requestKey)
	if err != nil && status != http.StatusNotFound {
		return nil, api.InternalServerErrorf(api.StatusInternalServerError,
			"Failed to retrieve pending pubsub request for key: %s. error: %s", requestKey, err.Error())
	}

	if entity == nil && status == http.StatusNotFound {
		return nil, nil
	}

	serializedContextData, ok := entity.Properties[NcPendingPublishContextTag].(string)
	if !ok {
		return nil, api.InternalServerErrorf(api.StatusInternalServerError,
			"Failed to retrieve entity context data from %s", entity.Properties[NcPendingPublishContextTag])
	}

	var pendingPublishContext NcPendingPublishRequestContext
	if err := json.Unmarshal([]byte(serializedContextData), &pendingPublishContext); err != nil {
		return nil, api.InternalServerErrorf(api.JsonBodyUnMarshalFailed,
			"Failed to unmarshal pending pubsub request for key: %s. error: %v", requestKey, err)
	}

	return &pendingPublishContext, nil
}

// GetNcPendingPublishRequestContexts - Retrieve all NC publish / unpublish pending request contexts
func (p *NcPendingPublishRequestRepo) GetNcPendingPublishRequestContexts() (map[string]NcPendingPublishRequestContext, *api.JsonErr) {
	var (
		context           interface{}
		requestsToProcess = make(map[string]NcPendingPublishRequestContext)
	)

	for {
		entities, hasMoreData, cx, status, err := p.dataStore.QueryAllRows(NcPendingPublishContextTableName, context)
		if err != nil {
			if status == http.StatusNotFound {
				return nil, nil
			}

			return nil, api.InternalServerErrorf(api.StatusInternalServerError, err.Error())
		}

		for _, entity := range entities {
			serializedRequestContext := entity.Properties[NcPendingPublishContextTag].(string)
			var requestContext NcPendingPublishRequestContext
			if err := json.Unmarshal([]byte(serializedRequestContext), &requestContext); err != nil {
				return nil, api.InternalServerErrorf(api.JsonBodyUnMarshalFailed,
					"Failed to unmarshal pending pubsub request: %s. error: %v", serializedRequestContext, err)
			}
			requestsToProcess[entity.RowKey] = requestContext
		}

		if !hasMoreData {
			break
		}

		context = cx
	}

	return requestsToProcess, nil
}

// CreateOrUpdateNcPendingPublishRequest - Creates or updates NC publish / unpublish context from the table.
func (p *NcPendingPublishRequestRepo) CreateOrUpdateNcPendingPublishRequest(
	requestContext *NcPendingPublishRequestContext) *api.JsonErr {
	return p.persistNcPendingPublishContext(requestContext)
}

func (p *NcPendingPublishRequestRepo) persistNcPendingPublishContext(
	context *NcPendingPublishRequestContext) *api.JsonErr {
	var (
		jsonSerializedBytes []byte
		err                 error
	)

	jsonSerializedBytes, err = json.Marshal(context)
	if err != nil {
		return api.InternalServerErrorf(api.JsonMarshallingFailed,
			"could not serialize pending pubsub request. NC: %s. error: %v", context.NetworkContainerID, err)
	}

	key := GetNcPendingPublishContextKey(context.NetworkID, context.NetworkContainerID, context.PubsubAzId)
	err = p.dataStore.InsertOrUpdate(NcPendingPublishContextTableName, key,
		TableToColumnMap[NcPendingPublishContextTableName], string(jsonSerializedBytes[:]))
	if err != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError,
			"could not persist pending pubsub request. key: %s error: %v", key, err)
	}

	return nil
}

func (p *NcPendingPublishRequestRepo) QueryNcPendingPublishContext(key string) (*NcPendingPublishRequestContext, *api.JsonErr) {
	entity, status, err := p.dataStore.Query(NcPendingPublishContextTableName, key)
	if err != nil && status != http.StatusNotFound {
		return nil, api.InternalServerErrorf(api.StatusInternalServerError,
			"could not retrieve pending pubsub request. key: %s error: %v", key, err)
	}

	if entity == nil {
		return nil, nil
	}

	serializedContextData := entity.Properties[NcPendingPublishContextTag].(string)
	var pendingPublishContext NcPendingPublishRequestContext
	if err := json.Unmarshal([]byte(serializedContextData), &pendingPublishContext); err != nil {
		return nil, api.InternalServerErrorf(api.JsonBodyUnMarshalFailed,
			"could not unmarshal pending pubsub request. key: %s error: %v", key, err)
	}

	return &pendingPublishContext, nil
}
