package testpersistencestore

import (
	"dnc/configuration"
	"dnc/logger/loggershim"
	"net/http"
)

// TestPersistenceStore :- datastore : map[ TableName ] map [ key ] value
type TestPersistenceStore struct {
	configuration.TestPersistenceStoreInfo
	datastore map[string]map[string]interface{}
}

// Init :- Initialize a TestPersistenceStore struct for testing purposes.
func (testpersistencestore *TestPersistenceStore) Init(tableToColumnMap map[string]string) {
	loggershim.LS.Log("Initializing TestPersistenceDataStore.datastore : map[tableName]map[rowKey]value.\n")
	testpersistencestore.datastore = make(map[string]map[string]interface{})

	for tableName := range tableToColumnMap {
		testpersistencestore.datastore[tableName] = make(map[string]interface{})
	}
}

// Delete :- Remove an element using rowKey and tableName.
func (testpersistencestore *TestPersistenceStore) Delete(tableName, rowKey string) error {
	delete(testpersistencestore.datastore[tableName], rowKey)
	loggershim.LS.Log("Deleted value with rowKey [%s] in %s.n\n", rowKey, tableName)
	return nil
}

// InsertOrUpdate :- Insert or update an element using rowKey and tablName.
func (testpersistencestore *TestPersistenceStore) InsertOrUpdate(tableName, rowKey, propertyName string, value interface{}) error {
	var success bool
	var successstr string
	testpersistencestore.datastore[tableName][rowKey], success = value.(string)

	if success {
		successstr = "Successfully"
	} else {
		successstr = "Unsuccessfully"
	}

	loggershim.LS.Log("%s assigned [%s] to [%s] with rowKey [%s].\n", successstr, value, tableName, rowKey)
	return nil
}

// Query :- Query an element using rowKey and tableName.
func (testpersistencestore *TestPersistenceStore) Query(tableName, rowKey string) (interface{}, int, error) {
	value, success := testpersistencestore.datastore[tableName][rowKey]
	var status = http.StatusNotFound

	if success {
		status = http.StatusOK
		//loggershim.LS.Log("Found value [%s] from [%s] with rowKey [%s].\n", value, tableName, rowKey)
	} else {
		loggershim.LS.Log("Failed to query from [%s] with rowKey [%s].\n", tableName, rowKey)
	}

	return value, status, nil
}

// QueryAllRows - Query a row in a given table.
func (testpersistencestore *TestPersistenceStore) QueryAllRows(tableName string, context interface{}) (
	map[string]interface{}, bool, interface{}, int, error) {
	var status = http.StatusNotFound
	//var successstr string

	if _, err := testpersistencestore.datastore[tableName]; err {
		status = http.StatusOK
		//successstr = "Found"
	}
	// else {
	// 	successstr = "Unable to find"
	// }

	// loggershim.LS.Log("%s rows in [%s].\n", successstr, tableName)
	return testpersistencestore.datastore[tableName], false, nil, status, nil
}

// KeyExist :- Check if key exists in a given table.
func (testpersistencestore *TestPersistenceStore) KeyExist(tableName, rowKey string) (bool, error) {
	_, retrieved := testpersistencestore.datastore[tableName][rowKey]
	var existstr string

	if retrieved {
		existstr = "exists"
	} else {
		existstr = "does not exist"
	}

	loggershim.LS.Log("rowKey [%s] %s in [%s].\n", rowKey, existstr, tableName)
	return retrieved, nil
}
