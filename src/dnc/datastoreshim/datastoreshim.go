package datastoreshim

import (
	"dnc/configuration"
	"dnc/datastore/azurestorage"
	orchestratorstorage "dnc/datastore/orchestratorstore"
	"dnc/datastore/testpersistencestore"
	"dnc/logger/loggershim"
	"dnc/utils"
	"fmt"
	"strings"
)

type insertInPartitionFunc func(tableName, partitionKey, rowKey, propertyName string, value interface{}) (int, error)
type insertInPartitionPropertiesFunc func(tableName, partitionKay, rowKey string, properties map[string]interface{}) (int, error)
type insertOrUpdateFunc func(tableName, rowKey, propertyName string, value interface{}) error
type upsertBatchFunc func(tableName, partitionKey string, batchItems map[string]map[string]interface{}) error
type insertBatchFunc func(tableName, partitionKey string, batchItems map[string]map[string]interface{}) error
type deleteBatchFunc func(tableName, partitionKey string, rowKeys []string) error
type updateInPartitionFunc func(tableName, partitionKay, rowKey string, properties map[string]interface{}) error
type updateInPartitionWithEtagFunc func(tableName, partitionKay, rowKey, etag string, properties map[string]interface{}) error
type deleteFunc func(tableName, rowKey string) error
type deleteInPartitionFunc func(tableName, partitionKey, rowKey string) error
type queryFunc func(tableName, rowKey string) (interface{}, int, error)
type queryInPartitionFunc func(tableName, partitionKey, rowKey string) (interface{}, int, error)
type queryInPartitionWithEtagFunc func(tableName, partitionKey, rowKey string) (interface{}, string, int, error)
type queryInPartitionWithEtagPropertiesFunc func(tableName, partitionKey, rowKey string) (map[string]interface{}, string, int, error)
type queryAllFunc func(tableName string, context interface{}) (map[string]interface{}, bool, interface{}, int, error)
type queryAllInPartitionFunc func(tableName, partitionKey string, context interface{}) (map[string]interface{}, bool, interface{}, int, error)
type keyExistFunc func(tableName, rowKey string) (bool, error)

// DataStoreShim - Struct to cache the data store management functions
type DataStoreShim struct {
	dataStoreSettings                  configuration.DataStoreSettings
	SupportedFeatures                  map[int]bool
	InsertInPartition                  insertInPartitionFunc
	InsertOrUpdate                     insertOrUpdateFunc
	UpsertBatch                        upsertBatchFunc
	InsertBatch                        insertBatchFunc
	DeleteBatch                        deleteBatchFunc
	Delete                             deleteFunc
	DeleteInPartition                  deleteInPartitionFunc
	Query                              queryFunc
	QueryInPartition                   queryInPartitionFunc
	QueryAll                           queryAllFunc
	QueryAllInPartition                queryAllInPartitionFunc
	KeyExist                           keyExistFunc
	UpdateInPartition                  updateInPartitionFunc
	UpdateInPartitionWithEtag          updateInPartitionWithEtagFunc
	QueryInPartitionWithEtag           queryInPartitionWithEtagFunc
	QueryInPartitionWithEtagProperties queryInPartitionWithEtagPropertiesFunc
	InsertInPartitionProperties        insertInPartitionPropertiesFunc
}

// NewDataStoreShim - Create a new datastore shim
func NewDataStoreShim(
	dataStoreSettings configuration.DataStoreSettings,
	dncSettings configuration.DNCSettings,
	tableToColumnMap map[string]string) *DataStoreShim {
	dataStoreShim := new(DataStoreShim)
	dataStoreShim.dataStoreSettings = dataStoreSettings
	dataStoreShim.SupportedFeatures = make(map[int]bool)
	dataStoreShim.Init(tableToColumnMap, dncSettings.ThresholdForLoggingDataStoreOpLatencyMs)
	return dataStoreShim
}

// IsFeatureSupported - Function to validate if the given feature is supported
func (dataStoreShim *DataStoreShim) IsFeatureSupported(featureID int) bool {
	if dataStoreShim.SupportedFeatures[featureID] == true {
		return true
	}

	return false
}

// Init - Function to initialize data store.
func (dataStoreShim *DataStoreShim) Init(
	tableToColumnMap map[string]string,
	thresholdForLoggingDataStoreOpLatencyMs int) {
	if dataStoreShim.dataStoreSettings.AzureStorageInfo != (configuration.AzureStorageInfo{}) {
		azStorage := azurestorage.AzureStorage{AzureStorageInfo: dataStoreShim.dataStoreSettings.AzureStorageInfo}
		azStorage.Init(tableToColumnMap, thresholdForLoggingDataStoreOpLatencyMs)
		dataStoreShim.Query = queryFunc(azStorage.Query)
		dataStoreShim.QueryInPartition = queryInPartitionFunc(azStorage.QueryInPartition)
		dataStoreShim.Delete = deleteFunc(azStorage.Delete)
		dataStoreShim.DeleteInPartition = deleteInPartitionFunc(azStorage.DeleteInPartition)
		dataStoreShim.InsertOrUpdate = insertOrUpdateFunc(azStorage.InsertOrUpdate)
		dataStoreShim.InsertInPartition = insertInPartitionFunc(azStorage.InsertInPartition)
		dataStoreShim.InsertInPartitionProperties = insertInPartitionPropertiesFunc(azStorage.InsertInPartitionProperties)
		dataStoreShim.UpsertBatch = upsertBatchFunc(azStorage.UpsertBatch)
		dataStoreShim.InsertBatch = insertBatchFunc(azStorage.InsertBatch)
		dataStoreShim.DeleteBatch = deleteBatchFunc(azStorage.DeleteBatch)
		dataStoreShim.QueryAll = queryAllFunc(azStorage.QueryAllRows)
		dataStoreShim.QueryAllInPartition = queryAllInPartitionFunc(azStorage.QueryAllRowsInPartition)
		dataStoreShim.KeyExist = keyExistFunc(azStorage.KeyExist)
		dataStoreShim.UpdateInPartition = updateInPartitionFunc(azStorage.UpdateInPartition)
		dataStoreShim.UpdateInPartitionWithEtag = updateInPartitionWithEtagFunc(azStorage.UpdateInPartitionWithEtag)
		dataStoreShim.QueryInPartitionWithEtag = queryInPartitionWithEtagFunc(azStorage.QueryInPartitionWithEtag)
		dataStoreShim.QueryInPartitionWithEtagProperties = queryInPartitionWithEtagPropertiesFunc(azStorage.QueryInPartitionWithEtagProperties)
		// Default the feature to true
		dataStoreShim.SupportedFeatures[configuration.SharedSubnetDelegation] = true
		loggershim.LS.Debugf("SharedSubnetDelegation %v", dataStoreShim.SupportedFeatures[configuration.SharedSubnetDelegation])
		// Set the RegionalStorageUsage feature
		if len(strings.TrimSpace(dataStoreShim.dataStoreSettings.AzureStorageInfo.RegionalStorageConnectionString)) > 0 {
			dataStoreShim.SupportedFeatures[configuration.RegionalStorageAccountUsage] = true
		}
		loggershim.LS.Debugf("RegionalStorageAccountUsage %v", dataStoreShim.SupportedFeatures[configuration.RegionalStorageAccountUsage])
	} else if dataStoreShim.dataStoreSettings.TestPersistenceStoreInfo != (configuration.TestPersistenceStoreInfo{}) {
		tps := testpersistencestore.TestPersistenceStore{TestPersistenceStoreInfo: dataStoreShim.dataStoreSettings.TestPersistenceStoreInfo}
		tps.Init(tableToColumnMap)
		dataStoreShim.Query = queryFunc(tps.Query)
		dataStoreShim.QueryInPartition = queryInPartitionFunc(tps.QueryInPartition)
		dataStoreShim.Delete = deleteFunc(tps.Delete)
		dataStoreShim.DeleteInPartition = deleteInPartitionFunc(tps.DeleteInPartition)
		dataStoreShim.InsertOrUpdate = insertOrUpdateFunc(tps.InsertOrUpdate)
		dataStoreShim.InsertInPartition = insertInPartitionFunc(tps.InsertInPartition)
		dataStoreShim.UpsertBatch = upsertBatchFunc(tps.UpsertBatch)
		dataStoreShim.InsertBatch = insertBatchFunc(tps.InsertBatch)
		dataStoreShim.DeleteBatch = deleteBatchFunc(tps.DeleteBatch)
		dataStoreShim.QueryAll = queryAllFunc(tps.QueryAllRows)
		dataStoreShim.QueryAllInPartition = queryAllInPartitionFunc(tps.QueryAllRowsInPartition)
		dataStoreShim.KeyExist = keyExistFunc(tps.KeyExist)
		dataStoreShim.UpdateInPartition = updateInPartitionFunc(tps.UpdateInPartition)
		dataStoreShim.InsertInPartitionProperties = insertInPartitionPropertiesFunc(tps.InsertInPartitionProperties)
		dataStoreShim.UpdateInPartitionWithEtag = updateInPartitionWithEtagFunc(tps.UpdateInPartitionWithEtag)
		dataStoreShim.QueryInPartitionWithEtag = queryInPartitionWithEtagFunc(tps.QueryInPartitionWithEtag)
		dataStoreShim.QueryInPartitionWithEtagProperties = queryInPartitionWithEtagPropertiesFunc(tps.QueryInPartitionWithEtagProperties)
		// Default the feature to true
		dataStoreShim.SupportedFeatures[configuration.SharedSubnetDelegation] = true
		loggershim.LS.Debugf("SharedSubnetDelegation %v", dataStoreShim.SupportedFeatures[configuration.SharedSubnetDelegation])
		// Set the RegionalStorageUsage feature
		if len(strings.TrimSpace(dataStoreShim.dataStoreSettings.AzureStorageInfo.RegionalStorageConnectionString)) > 0 {
			dataStoreShim.SupportedFeatures[configuration.RegionalStorageAccountUsage] = true
		}
		loggershim.LS.Debugf("RegionalStorageAccountUsage %v", dataStoreShim.SupportedFeatures[configuration.RegionalStorageAccountUsage])
	} else if dataStoreShim.dataStoreSettings.OrchestratorStorageInfo != (configuration.OrchestratorStorageInfo{}) {
		os := orchestratorstorage.OrchestratorStorage{OrchestratorStorageInfo: dataStoreShim.dataStoreSettings.OrchestratorStorageInfo}
		os.Init(tableToColumnMap)
		dataStoreShim.Query = queryFunc(os.Query)
		dataStoreShim.Delete = deleteFunc(os.Delete)
		dataStoreShim.InsertOrUpdate = insertOrUpdateFunc(os.InsertOrUpdate)
		dataStoreShim.QueryAll = queryAllFunc(os.QueryAll)
		dataStoreShim.KeyExist = keyExistFunc(os.KeyExist)
	} else {
		utils.OnErrorFail(fmt.Errorf("Invalid/UnSupported data store"), "")
	}
}
