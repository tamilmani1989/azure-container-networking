package datastoreshim

import (
	"dnc/configuration"
	"dnc/datastore/azuresql"
	"dnc/datastore/azurestorage"
	"dnc/datastore/orchestratorstore"
	"dnc/datastore/testpersistencestore"
	"dnc/logger/loggershim"
	"dnc/utils"
	"fmt"
)

type dataStoreInterfaces interface {
	InsertInPartition(tableName, partitionKey, rowKey, propertyName string, value interface{}) (int, error)
	InsertOrUpdate(tableName, rowKey, propertyName string, value interface{}) error
	Delete(tableName, rowKey string) error
	DeleteInPartition(tableName, partitionKey, rowKey string) error
	Query(tableName, rowKey string) (interface{}, int, error)
	QueryInPartition(tableName, partitionKey, rowKey string) (interface{}, int, error)
	QueryAll(tableName string, context interface{}) (map[string]interface{}, bool, interface{}, int, error)
	QueryAllInPartition(tableName, partitionKey string, context interface{}) (map[string]interface{}, bool, interface{}, int, error)
	KeyExist(tableName, rowKey string) (bool, error)
	IsFeatureSupported(featureID int) bool
}

type insertInPartitionFunc func(tableName, partitionKey, rowKey, propertyName string, value interface{}) (int, error)
type insertOrUpdateFunc func(tableName, rowKey, propertyName string, value interface{}) error
type deleteFunc func(tableName, rowKey string) error
type deleteInPartitionFunc func(tableName, partitionKey, rowKey string) error
type queryFunc func(tableName, rowKey string) (interface{}, int, error)
type queryInPartitionFunc func(tableName, partitionKey, rowKey string) (interface{}, int, error)
type queryAllFunc func(tableName string, context interface{}) (map[string]interface{}, bool, interface{}, int, error)
type queryAllInPartitionFunc func(tableName, partitionKey string, context interface{}) (map[string]interface{}, bool, interface{}, int, error)
type keyExistFunc func(tableName, rowKey string) (bool, error)

// DataStoreShim - Struct to cache the data store management functions
type DataStoreShim struct {
	dataStoreSettings   configuration.DataStoreSettings
	SupportedFeatures   map[int]bool
	InsertInPartition   insertInPartitionFunc
	InsertOrUpdate      insertOrUpdateFunc
	Delete              deleteFunc
	DeleteInPartition   deleteInPartitionFunc
	Query               queryFunc
	QueryInPartition    queryInPartitionFunc
	QueryAll            queryAllFunc
	QueryAllInPartition queryAllInPartitionFunc
	KeyExist            keyExistFunc
}

// NewDataStoreShim - Create a new datastore shim
func NewDataStoreShim(
	dataStoreConfiguration configuration.DataStoreSettings,
	tableToColumnMap map[string]string) *DataStoreShim {
	dataStoreShim := new(DataStoreShim)
	dataStoreShim.dataStoreSettings = dataStoreConfiguration
	dataStoreShim.SupportedFeatures = make(map[int]bool)
	dataStoreShim.Init(tableToColumnMap)
	return dataStoreShim
}

func (wf insertOrUpdateFunc) InsertOrUpdate(tableName, rowKey string, propertyName string, value interface{}) error {
	return wf(tableName, rowKey, propertyName, value)
}

func (wf insertInPartitionFunc) InsertInPartition(tableName, partitionKey, rowKey string, propertyName string, value interface{}) (int, error) {
	return wf(tableName, partitionKey, rowKey, propertyName, value)
}

func (wf deleteFunc) Delete(tableName, rowKey string) error {
	return wf(tableName, rowKey)
}

func (wf deleteInPartitionFunc) DeleteInPartition(tableName, partitionKey, rowKey string) error {
	return wf(tableName, partitionKey, rowKey)
}

func (wf queryFunc) Query(tableName, rowKey string) (interface{}, int, error) {
	return wf(tableName, rowKey)
}

func (wf queryInPartitionFunc) QueryInPartition(tableName, partitionKey, rowKey string) (interface{}, int, error) {
	return wf(tableName, partitionKey, rowKey)
}

func (wf queryAllFunc) QueryAll(tableName string, context interface{}) (map[string]interface{}, bool, interface{}, int, error) {
	return wf(tableName, context)
}

func (wf queryAllInPartitionFunc) QueryAllInPartition(tableName, partitionKey string, context interface{}) (
	map[string]interface{}, bool, interface{}, int, error) {
	return wf(tableName, partitionKey, context)
}

func (wf keyExistFunc) KeyExist(tableName, rowKey string) (bool, error) {
	return wf(tableName, rowKey)
}

// IsFeatureSupported - Function to validate if the given feature is supported
func (dataStoreShim *DataStoreShim) IsFeatureSupported(featureID int) bool {
	if dataStoreShim.SupportedFeatures[featureID] == true {
		return true
	}

	return false
}

// Init - Function to initialize data store.
func (dataStoreShim *DataStoreShim) Init(tableToColumnMap map[string]string) {
	if dataStoreShim.dataStoreSettings.AzureStorageInfo != (configuration.AzureStorageInfo{}) {
		azStorage := azurestorage.AzureStorage{AzureStorageInfo: dataStoreShim.dataStoreSettings.AzureStorageInfo}
		azStorage.Init(tableToColumnMap)
		dataStoreShim.Query = queryFunc(azStorage.Query)
		dataStoreShim.QueryInPartition = queryInPartitionFunc(azStorage.QueryInPartition)
		dataStoreShim.Delete = deleteFunc(azStorage.Delete)
		dataStoreShim.DeleteInPartition = deleteInPartitionFunc(azStorage.DeleteInPartition)
		dataStoreShim.InsertOrUpdate = insertOrUpdateFunc(azStorage.InsertOrUpdate)
		dataStoreShim.InsertInPartition = insertInPartitionFunc(azStorage.InsertInPartition)
		dataStoreShim.QueryAll = queryAllFunc(azStorage.QueryAllRows)
		dataStoreShim.QueryAllInPartition = queryAllInPartitionFunc(azStorage.QueryAllRowsInPartition)
		dataStoreShim.KeyExist = keyExistFunc(azStorage.KeyExist)
		// Default the feature to true
		dataStoreShim.SupportedFeatures[configuration.SharedSubnetDelegation] = true
		loggershim.LS.Debugf("SharedSubnetDelegation %v", dataStoreShim.SupportedFeatures[configuration.SharedSubnetDelegation])
	} else if dataStoreShim.dataStoreSettings.AzureSQLInfo != (configuration.AzureSQLInfo{}) {
		azSQL := azuresql.AzureSQL{AzureSQLInfo: dataStoreShim.dataStoreSettings.AzureSQLInfo}
		azSQL.Init(tableToColumnMap)
		dataStoreShim.Query = queryFunc(azSQL.Query)
		dataStoreShim.Delete = deleteFunc(azSQL.Delete)
		dataStoreShim.InsertOrUpdate = insertOrUpdateFunc(azSQL.InsertOrUpdate)
		dataStoreShim.QueryAll = queryAllFunc(azSQL.QueryAllRows)
		dataStoreShim.KeyExist = keyExistFunc(azSQL.KeyExist)
	} else if dataStoreShim.dataStoreSettings.TestPersistenceStoreInfo != (configuration.TestPersistenceStoreInfo{}) {
		tps := testpersistencestore.TestPersistenceStore{TestPersistenceStoreInfo: dataStoreShim.dataStoreSettings.TestPersistenceStoreInfo}
		tps.Init(tableToColumnMap)
		dataStoreShim.Query = queryFunc(tps.Query)
		dataStoreShim.QueryInPartition = queryInPartitionFunc(tps.QueryInPartition)
		dataStoreShim.Delete = deleteFunc(tps.Delete)
		dataStoreShim.DeleteInPartition = deleteInPartitionFunc(tps.DeleteInPartition)
		dataStoreShim.InsertOrUpdate = insertOrUpdateFunc(tps.InsertOrUpdate)
		dataStoreShim.InsertInPartition = insertInPartitionFunc(tps.InsertInPartition)
		dataStoreShim.QueryAll = queryAllFunc(tps.QueryAllRows)
		dataStoreShim.QueryAllInPartition = queryAllInPartitionFunc(tps.QueryAllRowsInPartition)
		dataStoreShim.KeyExist = keyExistFunc(tps.KeyExist)
		// Default the feature to true
		dataStoreShim.SupportedFeatures[configuration.SharedSubnetDelegation] = true
		loggershim.LS.Debugf("SharedSubnetDelegation %v", dataStoreShim.SupportedFeatures[configuration.SharedSubnetDelegation])
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
