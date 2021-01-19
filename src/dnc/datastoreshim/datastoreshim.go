package datastoreshim

import (
	"dnc/configuration"
	"dnc/datastore/azurestorage"
	"dnc/datastore/testpersistencestore"
	"dnc/logger/loggershim"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/storage"
)

// DataStore - Interface for storage operations
type DataStoreShim interface {
	Query(tableName, rowKey string) (*storage.Entity, int, error)
	QueryAllRows(tableName string, context interface{}) ([]*storage.Entity, bool, interface{}, int, error)
	QueryInPartition(tableName, partitionKey, rowKey string) (*storage.Entity, int, error)
	QueryAllRowsInPartition(tableName, partitionKey string, context interface{}) ([]*storage.Entity, bool, interface{}, int, error)
	QueryFilter(tableName, filter string) ([]*storage.Entity, error)
	InsertInPartition(tableName, partitionKey, rowKey, propertyName string, value interface{}) (int, error)
	InsertInPartitionProperties(tableName, partitionKey, rowKey string, properties map[string]interface{}) (int, error)
	InsertOrUpdate(tableName, rowKey, propertyName string, value interface{}) error
	InsertBatch(tableName, partitionKey string, batchItems map[string]map[string]interface{}) error
	UpsertBatch(tableName string, batchItems map[string]map[string]interface{}) error
	MergeBatch(tableName, partitionKey string, batchItems map[string]map[string]interface{}) error
	UpdateInPartition(tableName, partitionKay, rowKey string, properties map[string]interface{}) error
	UpdateInPartitionWithEtag(tableName, partitionKay, rowKey, etag string, properties map[string]interface{}) error
	Delete(tableName, rowKey string) error
	DeleteInPartition(tableName, partitionKey, rowKey string) error
	DeleteBatch(tableName, partitionKey string, rowKeys []string) error
	KeyExist(tableName, rowKey string) (bool, error)
}

// NewDataStoreShim - Create a new datastore shim
func NewDataStoreShim(dataStoreSettings configuration.DataStoreSettings, latencyThresholdMs int, tableToColumnMap map[string]string) (DataStoreShim, error) {
	if dataStoreSettings.AzureStorageInfo != (configuration.AzureStorageInfo{}) {
		azStorage, err := azurestorage.NewAzureStorage(dataStoreSettings.AzureStorageInfo, loggershim.LS, tableToColumnMap, latencyThresholdMs, dataStoreSettings.MaxBatchSize)
		if err != nil {
			return nil, fmt.Errorf("could not initialize azure storage: %v", err)
		}

		return azStorage, nil
	}

	if dataStoreSettings.TestPersistenceStoreInfo != (configuration.TestPersistenceStoreInfo{}) {
		var tps testpersistencestore.TestPersistenceStore
		tps.Init(tableToColumnMap, loggershim.LS)

		return &tps, nil
	}

	return nil, errors.New("No data store settings found")
}
