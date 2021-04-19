package datamodel

import (
	"dnc/api"
	"dnc/datastoreshim"
	"encoding/json"
	"net/http"
)

type NetworkContainerRepo struct {
	Datastore datastoreshim.DataStoreShim
}

func NewNetworkContainerRepo(ds datastoreshim.DataStoreShim) *NetworkContainerRepo {
	return &NetworkContainerRepo{
		Datastore: ds,
	}
}

// GetAllNetworkContainers - Retrieve all network containers
func (n *NetworkContainerRepo) GetAllNetworkContainers() (map[string]NCConfiguration, *api.JsonErr) {
	var context interface{}
	ncConfigs := make(map[string]NCConfiguration)
	for {
		entities, hasMoreData, cx, status, err := n.Datastore.QueryAllRows(networkContainerTableName, context)
		if err != nil {
			if status == http.StatusNotFound {
				return ncConfigs, nil
			}

			return nil, api.InternalServerErrorf(api.StatusInternalServerError,
				"Failed to Query all networkContainerTableName rows due to error: %+v",
				err)
		}

		for _, entity := range entities {
			serializedNCConfig := entity.Properties[networkContainerConfigurationTag].(string)
			var ncConfig NCConfiguration
			err = json.Unmarshal([]byte(serializedNCConfig), &ncConfig)
			if err != nil {
				return nil, api.InternalServerErrorf(api.JsonBodyUnMarshalFailed,
					"Failed to unmarshal NetworkContainer, SerializedContext %s, rowKey %s due to error: %+v",
					serializedNCConfig, entity.RowKey, err)
			}

			ncConfigs[entity.RowKey] = ncConfig
		}

		if !hasMoreData {
			break
		}
		context = cx
	}

	return ncConfigs, nil
}

func (n *NetworkContainerRepo) AnyNetworkContainerExists() (bool, *api.JsonErr) {
	exists, err := n.Datastore.AnyRowExists(networkContainerTableName)
	if err != nil {
		return false, api.InternalServerErrorf(api.StatusInternalServerError,
			"Failed to Query networkContainerTableName due to error: %+v",
			err)
	}

	return exists, nil
}

func (p *NetworkContainerRepo) DeleteAllRows() *api.JsonErr {
	err := p.Datastore.WipePartition(networkContainerTableName)
	if err != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError,
			"Failed to wipe networkContainerTableName due to error: %+v",
			err)
	}

	return nil
}
