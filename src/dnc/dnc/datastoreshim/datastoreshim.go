package datastoreshim

import (
	"dnc/configuration"
	"dnc/datastore/azuresql"
	"dnc/datastore/azurestorage"
	"dnc/datastore/orchestratorstore"
	"dnc/datastore/testpersistencestore"
	"dnc/utils"
	"fmt"
)

type dataStoreInterfaces interface {
	InsertOrUpdate(tableName, rowKey, propertyName string, value interface{}) error
	Delete(tableName, rowKey string) error
	Query(tableName, rowKey string) (interface{}, int, error)
	QueryAll(tableName string, context interface{}) (map[string]interface{}, bool, interface{}, int, error)
	KeyExist(tableName, rowKey string) (bool, error)
}

type insertOrUpdateFunc func(tableName, rowKey, propertyName string, value interface{}) error
type deleteFunc func(tableName, rowKey string) error
type queryFunc func(tableName, rowKey string) (interface{}, int, error)
type queryAllFunc func(tableName string, context interface{}) (map[string]interface{}, bool, interface{}, int, error)
type keyExistFunc func(tableName, rowKey string) (bool, error)

// DataStoreShim - Struct to cache the data store management functions
type DataStoreShim struct {
	dataStoreSettings configuration.DataStoreSettings
	InsertOrUpdate    insertOrUpdateFunc
	Delete            deleteFunc
	Query             queryFunc
	QueryAll          queryAllFunc
	KeyExist          keyExistFunc
}

// NewDataStoreShim - Create a new datastore shim
func NewDataStoreShim(
	dataStoreConfiguration configuration.DataStoreSettings,
	tableToColumnMap map[string]string) *DataStoreShim {
	dataStoreShim := new(DataStoreShim)
	dataStoreShim.dataStoreSettings = dataStoreConfiguration
	dataStoreShim.Init(tableToColumnMap)
	return dataStoreShim
}

func (wf insertOrUpdateFunc) InsertOrUpdate(tableName, rowKey string, propertyName string, value interface{}) error {
	return wf(tableName, rowKey, propertyName, value)
}

func (wf deleteFunc) Delete(tableName, rowKey string) error {
	return wf(tableName, rowKey)
}

func (wf queryFunc) Query(tableName, rowKey string) (interface{}, int, error) {
	return wf(tableName, rowKey)
}

func (wf queryAllFunc) QueryAll(tableName string, context interface{}) (map[string]interface{}, bool, interface{}, int, error) {
	return wf(tableName, context)
}

func (wf keyExistFunc) KeyExist(tableName, rowKey string) (bool, error) {
	return wf(tableName, rowKey)
}

// Init - Function to initialize data store.
func (dataStoreShim *DataStoreShim) Init(tableToColumnMap map[string]string) {

	if dataStoreShim.dataStoreSettings.AzureStorageInfo != (configuration.AzureStorageInfo{}) {
		azStorage := azurestorage.AzureStorage{AzureStorageInfo: dataStoreShim.dataStoreSettings.AzureStorageInfo}
		azStorage.Init(tableToColumnMap)
		dataStoreShim.Query = queryFunc(azStorage.Query)
		dataStoreShim.Delete = deleteFunc(azStorage.Delete)
		dataStoreShim.InsertOrUpdate = insertOrUpdateFunc(azStorage.InsertOrUpdate)
		dataStoreShim.QueryAll = queryAllFunc(azStorage.QueryAllRows)
		dataStoreShim.KeyExist = keyExistFunc(azStorage.KeyExist)
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
		dataStoreShim.Delete = deleteFunc(tps.Delete)
		dataStoreShim.InsertOrUpdate = insertOrUpdateFunc(tps.InsertOrUpdate)
		dataStoreShim.QueryAll = queryAllFunc(tps.QueryAllRows)
		dataStoreShim.KeyExist = keyExistFunc(tps.KeyExist)
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
