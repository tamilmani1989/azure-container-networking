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
	AnyRowExists(tableName string) (bool, error)

	/*
	  todo: passing an arbitrary filter string to these methods makes mocking difficult.
	   we should consider a more constrained API surface once we have a good idea of the extent of our querying needs,
	   so that we can have a well defined data type that represents predicates and limits.
	   ex.
	   type Filter struct {
	       Partition  string
	       Properties map[string][]string
	       Count      int
	   }
	   QueryFilter(filter Filter) ([]*storage.Entity, error)
	   from the above, implementations can either build a filter string for the cosmos API, or use the fields to find results
	   in the in-memory datastore.
	*/
	QueryFilter(tableName, filter string) ([]*storage.Entity, error)
	QueryFilterTopN(tableName, filter string, count int) ([]*storage.Entity, error)
	// Deprecated: Use InsertEntity
	InsertInPartition(tableName, partitionKey, rowKey, propertyName string, value interface{}) (int, error)
	// Deprecated: Use InsertEntity
	InsertInPartitionProperties(tableName, partitionKey, rowKey string, properties map[string]interface{}) (int, error)
	// Deprecated: Use InsertEntity
	Insert(tableName, rowKey, propertyName string, value interface{}) (int, error)
	InsertEntity(tableName string, entity storage.Entity) error
	InsertOrUpdateEntity(tableName string, entity storage.Entity) error
	// Deprecated: Use InsertOrUpdateEntity
	InsertOrUpdate(tableName, rowKey, propertyName string, value interface{}) error
	// Deprecated: Use InsertOrUpdateEntity
	InsertOrUpdateProperties(tableName, partitionKey, rowKey string, properties map[string]interface{}) error
	// Deprecated: Use InsertEntityBatch
	InsertBatch(tableName, partitionKey string, batchItems map[string]map[string]interface{}) error
	InsertEntityBatch(tableName string, entities []storage.Entity) error
	UpdateEntityBatch(tableName string, entities []storage.Entity) error
	// Deprecated: Use UpsertEntityBatch
	UpsertBatch(tableName string, batchItems map[string]map[string]interface{}) error
	UpsertEntityBatch(tableName string, entities []storage.Entity) error
	MergeEntityBatch(tableName string, entities []storage.Entity) error
	UpdateWithEtag(tableName, rowKey, etag string, propertyName string, value interface{}) error
	UpdateEntity(tableName string, entity storage.Entity) error
	// Deprecated: Use UpdateEntity
	UpdateInPartition(tableName, partitionKey, rowKey string, properties map[string]interface{}) error
	// Deprecated: Use UpdateEntity
	UpdateInPartitionWithEtag(tableName, partitionKey, rowKey, etag string, properties map[string]interface{}) error
	Delete(tableName, rowKey string) error
	DeleteWithEtag(tableName, rowKey string, etag string) error
	DeleteInPartition(tableName, partitionKey, rowKey string) error
	DeleteInPartitionWithEtag(tableName, partitionKey, rowKey, etag string) error
	// todo: do we need to delete with etags? Assuming we want our deletes to be idempotent, this may not be the case.
	DeleteBatch(tableName, partitionKey string, rowKeys []string) error
	WipePartition(tablename string) error
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

// NewDataStoreShimForCleanup - Create a new datastore shim using the shared StorageClient.
// This is currently initialized by DNC cleanupService (used in AKSSwift scenario) per cluster partition.
// Note for now StorageClient will be initialized by the main thread and will be shared across AzureStorage instances.
func NewDataStoreShimEx(
	partitionKey string,
	isUsingCosmosDB bool,
	storageClient storage.Client,
	latencyThresholdMs int,
	tableToColumnMap map[string]string,
	maxBatchSize int) (DataStoreShim, error) {
	loggershim.LS.Debugf("Initializing NewDataStoreShimEx for partitionKey %s, isUsingCosmosDB %s, maxBatchSize %d", partitionKey, isUsingCosmosDB, maxBatchSize)
	azStorage, err := azurestorage.NewAzureStorageEx(partitionKey, isUsingCosmosDB, storageClient, loggershim.LS, tableToColumnMap, latencyThresholdMs, maxBatchSize)
	if err != nil {
		return nil, fmt.Errorf("could not initialize azure storage: %v", err)
	}

	return azStorage, nil
}
