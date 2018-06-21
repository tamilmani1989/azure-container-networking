package testpersistencestore

import (
	"dnc/configuration"
	"dnc/logger/loggershim"
	"net/http"
	"sync"
)

// TestPersistenceStore :- datastore : map[ TableName ] map [ key ] value
type TestPersistenceStore struct {
	configuration.TestPersistenceStoreInfo
	datastore map[string]map[string]interface{}
	locks     map[string]*sync.RWMutex
}

// Init :- Initialize a TestPersistenceStore struct for testing purposes.
func (testpersistencestore *TestPersistenceStore) Init(tableToColumnMap map[string]string) {
	loggershim.LS.Debugf("Initializing TestPersistenceDataStore.datastore : map[tableName]map[rowKey]value.\n")
	testpersistencestore.datastore = make(map[string]map[string]interface{})
	testpersistencestore.locks = make(map[string]*sync.RWMutex)

	for tableName := range tableToColumnMap {
		testpersistencestore.datastore[tableName] = make(map[string]interface{})
		testpersistencestore.locks[tableName] = &sync.RWMutex{}
	}
}

// Delete :- Remove an element using rowKey and tableName.
func (testpersistencestore *TestPersistenceStore) Delete(tableName, rowKey string) error {
	testpersistencestore.locks[tableName].Lock()
	delete(testpersistencestore.datastore[tableName], rowKey)
	testpersistencestore.locks[tableName].Unlock()
	loggershim.LS.Debugf("Deleted value with rowKey [%s] in %s.n\n", rowKey, tableName)
	return nil
}

// InsertOrUpdate :- Insert or update an element using rowKey and tablName.
func (testpersistencestore *TestPersistenceStore) InsertOrUpdate(tableName, rowKey, propertyName string, value interface{}) error {
	var success bool
	var successstr string

	testpersistencestore.locks[tableName].Lock()
	testpersistencestore.datastore[tableName][rowKey], success = value.(string)
	testpersistencestore.locks[tableName].Unlock()

	if success {
		successstr = "Successfully"
	} else {
		successstr = "Unsuccessfully"
	}

	loggershim.LS.Debugf("%s assigned [%s] to [%s] with rowKey [%s].\n", successstr, value, tableName, rowKey)
	return nil
}

// Query :- Query an element using rowKey and tableName.
func (testpersistencestore *TestPersistenceStore) Query(tableName, rowKey string) (interface{}, int, error) {
	testpersistencestore.locks[tableName].RLock()
	value, success := testpersistencestore.datastore[tableName][rowKey]
	testpersistencestore.locks[tableName].RUnlock()
	var status = http.StatusNotFound

	if success {
		status = http.StatusOK
		//loggershim.LS.Debugf("Found value [%s] from [%s] with rowKey [%s].\n", value, tableName, rowKey)
	} else {
		loggershim.LS.Debugf("Failed to query from [%s] with rowKey [%s].\n", tableName, rowKey)
	}

	return value, status, nil
}

// QueryAllRows - Query a row in a given table.
func (testpersistencestore *TestPersistenceStore) QueryAllRows(tableName string, context interface{}) (
	map[string]interface{}, bool, interface{}, int, error) {
	var status = http.StatusNotFound

	testpersistencestore.locks[tableName].RLock()
	defer testpersistencestore.locks[tableName].RUnlock()
	if _, err := testpersistencestore.datastore[tableName]; err {
		status = http.StatusOK
	}

	return testpersistencestore.datastore[tableName], false, nil, status, nil
}

// KeyExist :- Check if key exists in a given table.
func (testpersistencestore *TestPersistenceStore) KeyExist(tableName, rowKey string) (bool, error) {
	testpersistencestore.locks[tableName].RLock()
	_, retrieved := testpersistencestore.datastore[tableName][rowKey]
	testpersistencestore.locks[tableName].RUnlock()
	var existstr string

	if retrieved {
		existstr = "exists"
	} else {
		existstr = "does not exist"
	}

	loggershim.LS.Debugf("rowKey [%s] %s in [%s].\n", rowKey, existstr, tableName)
	return retrieved, nil
}
