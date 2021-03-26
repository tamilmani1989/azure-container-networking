package datamodel

import (
	"dnc/api"
	"dnc/datastoreshim"
	"encoding/json"
	"fmt"
	"net/http"
)

type NcPendingPublishRequestRepo struct {
	dncPartitionKey string
	data            datastoreshim.DataStoreShim
}

func NewNcPendingPublishRequestRepo(dataStore datastoreshim.DataStoreShim) *NcPendingPublishRequestRepo {
	return &NcPendingPublishRequestRepo{
		data: dataStore,
	}
}

// RemoveNcPendingPublishContext - Remove pending NC publish / unpublish context from the table.
func (p *NcPendingPublishRequestRepo) RemoveNcPendingPublishContext(
	networkID, networkContainerID string,
	pubsubAzId string) *api.JsonErr {
	key := GetNcPendingPublishContextKey(networkID, networkContainerID, pubsubAzId)
	err := p.data.Delete(NcPendingPublishContextTableName, key)
	if err != nil {
		return api.InternalServerError(api.StatusInternalServerError, fmt.Sprintf(
			"Failed to delete NcPendingPublishRequestContext for key:[%s] due to error:[%s]",
			key,
			err.Error()))
	}

	return nil
}

// TryGetPendingCleanupRequest - Query a network container by key.
func (p *NcPendingPublishRequestRepo) TryGetNcPendingPublish(requestKey string) (*NcPendingPublishRequestContext, *api.JsonErr) {
	entity, status, err := p.data.Query(NcPendingPublishContextTableName, requestKey)

	if err != nil && status != http.StatusNotFound {
		return nil, api.InternalServerError(api.StatusInternalServerError, fmt.Sprintf(
			"Failed to retrieve NcPendingPublishRequestContext for key:[%s] due to error:[%s]",
			requestKey,
			err.Error()))
	}

	if entity == nil && status == http.StatusNotFound {
		return nil, nil
	}

	serializedContextData, ok := entity.Properties[NcPendingPublishContextTag].(string)
	if !ok {
		return nil, api.InternalServerError(api.StatusInternalServerError, fmt.Sprintf(
			"Failed to retrieve entity context data from [%s]", entity.Properties[NcPendingPublishContextTag]))
	}

	var pendingPublishContext NcPendingPublishRequestContext
	if err := json.Unmarshal([]byte(serializedContextData), &pendingPublishContext); err != nil {
		return nil, api.InternalServerError(api.JsonBodyUnMarshalFailed, fmt.Sprintf(
			"Failed to unmarshal pending NC publish context for key:[%s] due to error:[%s]",
			requestKey, err))
	}

	return &pendingPublishContext, nil
}
