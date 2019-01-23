package testpersistencestore

import (
	"dnc/configuration"
	"dnc/logger/loggershim"
	"fmt"
	"net/http"
	"sync"
)

var dummyCounter int

const conflictCounter = 1

// TestPersistenceStore :- datastore : map[ TableName ] map [ key ] value
type TestPersistenceStore struct {
	configuration.TestPersistenceStoreInfo
	datastore map[string]map[string]interface{}
	locks     map[string]*sync.RWMutex
}

// Init :- Initialize a TestPersistenceStore struct for testing purposes.
func (testpersistencestore *TestPersistenceStore) Init(tableToColumnMap map[string]string) {
	loggershim.LS.Debugf("Initializing TestPersistenceDataStore.datastore : map[tableName]map[rowKey]value.")
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
	loggershim.LS.Debugf("Deleted value with rowKey [%s] in %s.n", rowKey, tableName)
	return nil
}

// DeleteInPartition - Deletes a row in the partition of the given table.
func (testpersistencestore *TestPersistenceStore) DeleteInPartition(tableName, partitionKey, rowKey string) error {
	return testpersistencestore.Delete(tableName, rowKey)
}

// InsertOrUpdate :- Insert or update an element using rowKey and tablName.
func (testpersistencestore *TestPersistenceStore) InsertOrUpdate(tableName, rowKey, propertyName string, value interface{}) error {
	testpersistencestore.locks[tableName].Lock()
	testpersistencestore.datastore[tableName][rowKey] = value
	testpersistencestore.locks[tableName].Unlock()

	loggershim.LS.Debugf("Assigned [%+v] to [%s] with rowKey [%s].", value, tableName, rowKey)
	return nil
}

// InsertInPartition - Insert a row in the specific partition of the given table.
func (testpersistencestore *TestPersistenceStore) InsertInPartition(tableName, partitionKay, rowKey, propertyName string, value interface{}) (
	int, error) {

	// Simulate the scenario where the entry is already present in the table.
	if dummyCounter < conflictCounter {
		dummyCounter++
		loggershim.LS.Debugf("Data entry is already present in the table")
		return http.StatusConflict, fmt.Errorf("Data entry is already present in the table")
	}

	dummyCounter = 0
	testpersistencestore.InsertOrUpdate(tableName, rowKey, propertyName, value)

	return http.StatusOK, nil
}

// Query :- Query an element using rowKey and tableName.
func (testpersistencestore *TestPersistenceStore) Query(tableName, rowKey string) (interface{}, int, error) {
	testpersistencestore.locks[tableName].RLock()
	value, success := testpersistencestore.datastore[tableName][rowKey]
	testpersistencestore.locks[tableName].RUnlock()
	var status = http.StatusNotFound

	if success {
		status = http.StatusOK
		//loggershim.LS.Debugf("Found value [%s] from [%s] with rowKey [%s].", value, tableName, rowKey)
	} else {
		loggershim.LS.Debugf("Failed to query from [%s] with rowKey [%s].", tableName, rowKey)
	}

	return value, status, nil
}

// QueryInPartition - Query a row in the given partition of the table.
func (testpersistencestore *TestPersistenceStore) QueryInPartition(tableName, partitionKey, rowKey string) (
	interface{}, int, error) {
	return testpersistencestore.Query(tableName, rowKey)
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
	queriedTable := make(map[string]interface{})
	for key, value := range testpersistencestore.datastore[tableName] {
		queriedTable[key] = value
	}
	return queriedTable, false, nil, status, nil
}

// QueryAllRowsInPartition - Query all rows in the specific partition of the given table.
func (testpersistencestore *TestPersistenceStore) QueryAllRowsInPartition(tableName, partitionKey string, context interface{}) (
	map[string]interface{}, bool, interface{}, int, error) {
	return testpersistencestore.QueryAllRows(tableName, context)
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

	loggershim.LS.Debugf("rowKey [%s] %s in [%s].", rowKey, existstr, tableName)
	return retrieved, nil
}
