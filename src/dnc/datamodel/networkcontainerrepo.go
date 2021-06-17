package datamodel

import (
	"dnc/api"
	"dnc/datastore/azurestorage"
	"dnc/datastoreshim"
	"dnc/logger/loggershim"
	"encoding/json"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/storage"
)

const (
	networkContainerTableName = "NetworkContainerTable"
	ncKey                     = "Network_%s_ID_%s"
	NCRepositoryName          = "NCRepository"

	// Properties in the NetworkContainerConfigurationTable
	networkContainerConfigProperty = "NetworkContainerConfiguration"
	// subnetKeyProperty is already defined in datamodel
)

type NetworkContainerRepo struct {
	dncPartitionKey string
	dataStore       datastoreshim.DataStoreShim
	logger          loggershim.LoggerInterface
}

type NetworkContainerEntity struct {
	// RowKey is the NetworkContainer key: Network_{NetworkID}_ID_{NCID}
	RowKey     string
	Config     NCConfiguration
	SubnetKey  string
	NodeSetKey string
}

func NewNetworkContainerRepo(
	dataStore datastoreshim.DataStoreShim,
	dncPartitionKey string,
	logger loggershim.LoggerInterface) *NetworkContainerRepo {
	return &NetworkContainerRepo{
		dncPartitionKey: dncPartitionKey,
		dataStore:       dataStore,
		logger:          logger,
	}
}

func (nc *NetworkContainerEntity) ToStorageEntity(dncPartitionKey string) (storage.Entity, *api.JsonErr) {
	e := storage.Entity{
		Table:        &storage.Table{Name: networkContainerTableName},
		PartitionKey: dncPartitionKey,
		RowKey:       nc.RowKey,
		Properties: map[string]interface{}{
			subnetKeyProperty:  nc.SubnetKey,
			NodeSetKeyProperty: nc.NodeSetKey,
		},
	}

	jsonSerializedBytes, err := json.Marshal(nc.Config)
	if err != nil {
		return storage.Entity{}, api.InternalServerErrorf(api.JsonMarshallingFailed,
			"could not marshal NC config: %+v. error: %v", nc.Config, err)
	}

	e.Properties[networkContainerConfigProperty] = string(jsonSerializedBytes)

	return e, nil
}

// GetAllNetworkContainers - Retrieve all network containers
func (n *NetworkContainerRepo) GetAllNetworkContainers() (map[string]NetworkContainerEntity, *api.JsonErr) {
	var context interface{}
	ncDataMap := make(map[string]NetworkContainerEntity)
	for {
		entities, hasMoreData, cx, status, err := n.dataStore.QueryAllRows(networkContainerTableName, context)
		if err != nil {
			if status == http.StatusNotFound {
				return ncDataMap, nil
			}

			return nil, api.InternalServerErrorf(api.StatusInternalServerError,
				"Failed to Query all networkContainerTableName rows due to error: %+v", err)
		}

		for _, entity := range entities {
			nc, errParse := storageEntityToNetworkContainerData(entity)
			if errParse != nil {
				return nil, errParse
			}

			ncDataMap[entity.RowKey] = nc
		}

		if !hasMoreData {
			break
		}
		context = cx
	}

	return ncDataMap, nil
}

func (n *NetworkContainerRepo) AnyNetworkContainerExists() (bool, *api.JsonErr) {
	exists, err := n.dataStore.AnyRowExists(networkContainerTableName)
	if err != nil {
		return false, api.InternalServerErrorf(api.StatusInternalServerError,
			"Failed to Query networkContainerTableName due to error: %+v", err)
	}

	return exists, nil
}

func (n *NetworkContainerRepo) DeleteAllRows() *api.JsonErr {
	err := n.dataStore.WipePartition(networkContainerTableName)
	if err != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError,
			"Failed to wipe networkContainerTableName due to error: %+v", err)
	}

	return nil
}

// DeleteNetworkContainer - Delete network container configuration.
func (n *NetworkContainerRepo) DeleteNetworkContainer(networkID, networkContainerID string) *api.JsonErr {
	ncKey := GetNCKey(networkID, networkContainerID)
	if err := n.dataStore.Delete(networkContainerTableName, ncKey); err != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError,
			"could not delete network container %s in datastore. Error: %v", networkContainerID, err)
	}

	return nil
}

// GetNetworkContainerData - Get network container data
func (n *NetworkContainerRepo) GetNetworkContainerData(networkID, networkContainerID string) (NetworkContainerEntity, *api.JsonErr) {
	nc, err := n.TryQueryNetworkContainer(networkID, networkContainerID)
	if err != nil {
		return NetworkContainerEntity{}, err
	}

	if nc.RowKey == "" {
		return NetworkContainerEntity{}, api.NotFoundf(api.NetworkContainerNotFound, "Network container %s not found", networkContainerID)
	}

	return nc, nil
}

// TryQueryNetworkContainer - Query a network container by key.
func (n *NetworkContainerRepo) TryQueryNetworkContainer(networkID, networkContainerID string) (NetworkContainerEntity, *api.JsonErr) {
	ncKey := GetNCKey(networkID, networkContainerID)
	return n.TryQueryNetworkContainerByKey(ncKey)
}

// TryQueryNetworkContainerByKey - Query a network container by key.
func (n *NetworkContainerRepo) TryQueryNetworkContainerByKey(ncKey string) (NetworkContainerEntity, *api.JsonErr) {
	entity, status, err := n.dataStore.Query(networkContainerTableName, ncKey)
	if err != nil && status != http.StatusNotFound {
		return NetworkContainerEntity{}, api.InternalServerErrorf(api.StatusInternalServerError,
			"could not retrieve data for network container key: %s. error: %v", ncKey, err)
	}

	if entity == nil {
		return NetworkContainerEntity{}, nil
	}

	nc, errParse := storageEntityToNetworkContainerData(entity)
	if errParse != nil {
		return NetworkContainerEntity{}, errParse
	}

	return nc, nil
}

func (n *NetworkContainerRepo) PersistNetworkContainerData(nc *NetworkContainerEntity) *api.JsonErr {
	entity, err := nc.ToStorageEntity(n.dncPartitionKey)
	if err != nil {
		n.logger.Errorf("could not translate NC data to storage entity. NC data: %+v error: %v", nc, err)
		return err
	}

	if err := n.dataStore.InsertOrUpdateEntity(networkContainerTableName, entity); err != nil {
		storageErr, ok := err.(storage.AzureStorageServiceError)
		if ok {
			return &api.JsonErr{
				Code:       api.AzureStorageError,
				Text:       storageErr.Message,
				HTTPStatus: storageErr.StatusCode}
		}

		return api.InternalServerError(api.StatusInternalServerError, err.Error())
	}

	return nil
}

func storageEntityToNetworkContainerData(e *storage.Entity) (NetworkContainerEntity, *api.JsonErr) {
	nc := NetworkContainerEntity{
		RowKey:     e.RowKey,
		SubnetKey:  ExtractStringProperty(e, subnetKeyProperty),
		NodeSetKey: ExtractStringProperty(e, NodeSetKeyProperty),
	}

	var ncConfig NCConfiguration
	if err := json.Unmarshal([]byte(e.Properties[networkContainerConfigProperty].(string)), &ncConfig); err != nil {
		return NetworkContainerEntity{}, api.InternalServerErrorf(api.JsonBodyUnMarshalFailed,
			"could not deserialize NC config for NC: %s. error: %v", e.RowKey, err)
	}

	nc.Config = ncConfig
	return nc, nil
}

func (n *NetworkContainerRepo) QueryNetworkContainersInSubnet(networkID, subnetName string) ([]NetworkContainerEntity, *api.JsonErr) {
	subnetKey := GetSubnetKey(networkID, subnetName)
	return n.QueryNetworkContainers(subnetKeyProperty, subnetKey)
}

func (n *NetworkContainerRepo) QueryNetworkContainersInNodeSet(nodeSet string) ([]NetworkContainerEntity, *api.JsonErr) {
	nodeSetKey := EncodeNodeSetRowKey(nodeSet)
	return n.QueryNetworkContainers(NodeSetKeyProperty, nodeSetKey)
}

func (n *NetworkContainerRepo) QueryNetworkContainers(property string, values ...string) ([]NetworkContainerEntity, *api.JsonErr) {
	filter := azurestorage.BuildFilterOnProperty(n.dncPartitionKey, property, values)

	entities, err := n.dataStore.QueryFilter(networkContainerTableName, filter)
	if err != nil {
		return nil, api.InternalServerError(api.StatusInternalServerError, err.Error())
	}

	rows := make([]NetworkContainerEntity, len(entities))
	for i, entity := range entities {
		nc, errJson := storageEntityToNetworkContainerData(entity)
		if errJson != nil {
			return nil, errJson
		}

		rows[i] = nc
	}

	return rows, nil
}
