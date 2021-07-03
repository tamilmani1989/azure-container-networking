package datamodel

import (
	"dnc/api"
	"dnc/datastoreshim"
	"encoding/json"
	"net/http"
)

const (
	NcPendingPublishContextProperty = "NcPendingPublishContext"
	NcKeyProperty                   = "NcKey"
)

type NcPendingPublishRequestRepo struct {
	dataStore    datastoreshim.DataStoreShim
	partitionKey string
}

// NcPendingPublishRequestEntity Table Schema
type NcPendingPublishRequestEntity struct {
	RowKey        string
	Configuration NcPendingPublishRequestContext
	NCKey         string
}

func NewNcPendingPublishRequestRepo(dataStore datastoreshim.DataStoreShim, partitionKey string) *NcPendingPublishRequestRepo {
	return &NcPendingPublishRequestRepo{
		dataStore:    dataStore,
		partitionKey: partitionKey,
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

	serializedContextData, ok := entity.Properties[NcPendingPublishContextProperty].(string)
	if !ok {
		return nil, api.InternalServerErrorf(api.StatusInternalServerError,
			"Failed to retrieve entity context data from %s", entity.Properties[NcPendingPublishContextProperty])
	}

	var pendingPublishContext NcPendingPublishRequestContext
	if err := json.Unmarshal([]byte(serializedContextData), &pendingPublishContext); err != nil {
		return nil, api.InternalServerErrorf(api.JsonBodyUnMarshalFailed,
			"Failed to unmarshal pending pubsub request for key: %s. error: %v", requestKey, err)
	}

	return &pendingPublishContext, nil
}

// GetNcPendingPublishRequestContexts - Retrieve all NC publish / unpublish pending request contexts
func (p *NcPendingPublishRequestRepo) GetNcPendingPublishRequestContexts() (map[string]NcPendingPublishRequestEntity, *api.JsonErr) {
	var (
		context           interface{}
		requestsToProcess = make(map[string]NcPendingPublishRequestEntity)
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
			ncKeyProp := ExtractStringProperty(entity, NcKeyProperty)
			publishContextJSON := ExtractStringProperty(entity, NcPendingPublishContextProperty)

			var publishConf NcPendingPublishRequestContext
			if err := json.Unmarshal([]byte(publishContextJSON), &publishConf); err != nil {
				return nil, api.InternalServerError(api.JsonBodyUnMarshalFailed, err.Error())
			}

			e := NcPendingPublishRequestEntity{
				RowKey:        entity.RowKey,
				Configuration: publishConf,
				NCKey:         ncKeyProp,
			}
			requestsToProcess[e.RowKey] = e
		}

		if !hasMoreData {
			break
		}

		context = cx
	}

	return requestsToProcess, nil
}

// CreateOrUpdateNcPendingPublishRequestEntity - Creates or updates NC publish / unpublish entity from the table.
func (p *NcPendingPublishRequestRepo) CreateOrUpdateNcPendingPublishRequestEntity(
	entity NcPendingPublishRequestEntity) *api.JsonErr {
	jsonSerializedBytes, err := json.Marshal(entity.Configuration)
	if err != nil {
		return api.InternalServerErrorf(api.JsonMarshallingFailed,
			"could not serialize pending pubsub request. NC: %s. error: %v", entity.Configuration.NetworkContainerID, err)
	}
	properties := map[string]interface{}{
		NcKeyProperty:                   entity.NCKey,
		NcPendingPublishContextProperty: string(jsonSerializedBytes),
	}
	err = p.dataStore.InsertOrUpdateProperties(NcPendingPublishContextTableName, p.partitionKey, entity.RowKey, properties)

	if err != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError,
			"could not persist pending pubsub request. key: %s error: %v", entity.RowKey, err)
	}

	return nil
}

func (p *NcPendingPublishRequestRepo) QueryNcPendingPublishContext(key string) (NcPendingPublishRequestEntity, *api.JsonErr) {
	entity, status, err := p.dataStore.Query(NcPendingPublishContextTableName, key)
	if err != nil && status != http.StatusNotFound {
		return NcPendingPublishRequestEntity{}, api.InternalServerErrorf(api.StatusInternalServerError,
			"could not retrieve pending pubsub request. key: %s error: %v", key, err)
	}

	if entity == nil {
		return NcPendingPublishRequestEntity{}, nil
	}

	serializedContextData := entity.Properties[NcPendingPublishContextProperty].(string)
	var pendingPublishContext NcPendingPublishRequestContext
	if err := json.Unmarshal([]byte(serializedContextData), &pendingPublishContext); err != nil {
		return NcPendingPublishRequestEntity{}, api.InternalServerErrorf(api.JsonBodyUnMarshalFailed,
			"could not unmarshal pending pubsub request. key: %s error: %v", key, err)
	}

	ncKey := ExtractStringProperty(entity, NcKeyProperty)

	ncPendingPublishRequestEntity := NcPendingPublishRequestEntity{
		RowKey:        entity.RowKey,
		Configuration: pendingPublishContext,
		NCKey:         ncKey,
	}

	return ncPendingPublishRequestEntity, nil
}

func (p *NcPendingPublishRequestRepo) AnyPendingPublishRequestExists() (bool, *api.JsonErr) {
	exists, err := p.dataStore.AnyRowExists(NcPendingPublishContextTableName)
	if err != nil {
		return false, api.InternalServerErrorf(api.StatusInternalServerError,
			"Failed to Query NcPendingPublishContextTableName due to error: %+v",
			err)
	}

	return exists, nil
}

func (p *NcPendingPublishRequestRepo) DeleteAllRows() *api.JsonErr {
	err := p.dataStore.WipePartition(NcPendingPublishContextTableName)
	if err != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError,
			"Failed to wipe NcPendingPublishContextTableName due to error: %+v",
			err)
	}

	return nil
}
