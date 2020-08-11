package testpersistencestore

import (
	"dnc/configuration"
	"dnc/logger/loggershim"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type rowProperties struct {
	etag       string
	properties map[string]interface{}
}

// TestPersistenceStore :- datastore : map[ TableName ] map [ key ] value
type TestPersistenceStore struct {
	configuration.TestPersistenceStoreInfo
	//datastore map[string]map[string]interface{}
	datastore map[string]map[string]rowProperties
	locks     map[string]*sync.RWMutex
}

const induceErrorPercent = 5 // Induce 5% error for a particular operation

// Init :- Initialize a TestPersistenceStore struct for testing purposes.
func (testpersistencestore *TestPersistenceStore) Init(tableToColumnMap map[string]string) {
	loggershim.LS.Debugf("Initializing TestPersistenceDataStore.datastore : map[tableName]map[rowKey]value.")
	//testpersistencestore.datastore = make(map[string]map[string]interface{})
	testpersistencestore.datastore = make(map[string]map[string]rowProperties)
	testpersistencestore.locks = make(map[string]*sync.RWMutex)

	for tableName := range tableToColumnMap {
		//testpersistencestore.datastore[tableName] = make(map[string]interface{})
		testpersistencestore.datastore[tableName] = make(map[string]rowProperties)
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
	props := make(map[string]interface{})
	props[propertyName] = value
	tag := testpersistencestore.getEtag()

	testpersistencestore.locks[tableName].Lock()
	testpersistencestore.datastore[tableName][rowKey] = rowProperties{etag: tag, properties: props}
	testpersistencestore.locks[tableName].Unlock()

	loggershim.LS.Debugf("Assigned [%+v] to [%s] with rowKey [%s].", value, tableName, rowKey)
	return nil
}

// UpdateInPartition - Update a row in the given table, if row already exists
func (testpersistencestore *TestPersistenceStore) UpdateInPartition(
	tableName, partitionKay, rowKey string, properties map[string]interface{}) error {
	return testpersistencestore.UpdateInPartitionWithEtag(tableName, partitionKay, rowKey, "", properties)
}

// UpdateInPartitionWithEtag - Update (overwrite) a row in a given table if it exists.
// Match with specified etag for update.
func (testpersistencestore *TestPersistenceStore) UpdateInPartitionWithEtag(
	tableName, partitionKay, rowKey, etag string, properties map[string]interface{}) error {

	testpersistencestore.locks[tableName].Lock()
	defer func() {
		testpersistencestore.locks[tableName].Unlock()
	}()

	rowProps, retrieved := testpersistencestore.datastore[tableName][rowKey]
	if !retrieved {
		return fmt.Errorf("Row with key %s not found in table %s", rowKey, tableName)
	}

	if etag != "" && etag != "*" && etag != rowProps.etag {
		return fmt.Errorf("Caller's Etag does not match with current etag. Row key: %s, table: %s", rowKey, tableName)
	}

	tag := testpersistencestore.getEtag()
	testpersistencestore.datastore[tableName][rowKey] = rowProperties{etag: tag, properties: properties}
	loggershim.LS.Debugf("Update row to [%s] with rowKey [%s]. Caller's etag [%s]", tableName, rowKey, etag)
	return nil
}

// InsertInPartition - Insert a row in the specific partition of the given table.
func (testpersistencestore *TestPersistenceStore) InsertInPartition(
	tableName, partitionKay, rowKey, propertyName string, value interface{}) (int, error) {
	// Induce error to simulate the collisions due to multiple DNCs trying to allocate the same IP
	// in case of shared subnet delegation
	if testpersistencestore.induceError(induceErrorPercent) {
		loggershim.LS.Errorf("Data entry is already present in the table")
		return http.StatusConflict, fmt.Errorf("Data entry is already present in the table")
	}

	props := make(map[string]interface{})
	props[propertyName] = value
	testpersistencestore.InsertInPartitionProperties(tableName, partitionKay, rowKey, props)

	return http.StatusOK, nil
}

// InsertInPartitionProperties - Insert a row in the specific partition of the given table with multiple properties
// If row already exists, it will be updated (overwritten)
func (testpersistencestore *TestPersistenceStore) InsertInPartitionProperties(
	tableName, partitionKay, rowKey string, properties map[string]interface{}) (int, error) {
	// Induce error to simulate the collisions due to multiple DNCs trying to allocate the same IP
	// in case of shared subnet delegation
	if testpersistencestore.induceError(induceErrorPercent) {
		loggershim.LS.Errorf("Data entry is already present in the table")
		return http.StatusConflict, fmt.Errorf("Data entry is already present in the table")
	}

	testpersistencestore.locks[tableName].Lock()
	defer func() {
		testpersistencestore.locks[tableName].Unlock()
	}()

	_, retrieved := testpersistencestore.datastore[tableName][rowKey]
	if retrieved {
		return 0, fmt.Errorf("Row already exists with key %s in table %s", rowKey, tableName)
	}

	tag := testpersistencestore.getEtag()
	testpersistencestore.datastore[tableName][rowKey] = rowProperties{etag: tag, properties: properties}
	loggershim.LS.Debugf("Inserted row to [%s] with rowKey [%s].", tableName, rowKey)

	return http.StatusOK, nil
}

func (testpersistencestore *TestPersistenceStore) UpsertBatch(tableName, _ string, batch map[string]map[string]interface{}) error {
	testpersistencestore.locks[tableName].Lock()
	defer func() {
		testpersistencestore.locks[tableName].Unlock()
	}()

	for rowKey, rowProps := range batch {
		tag := testpersistencestore.getEtag()
		testpersistencestore.datastore[tableName][rowKey] = rowProperties{etag: tag, properties: rowProps}
	}

	return nil
}

func (testpersistencestore *TestPersistenceStore) InsertBatch(tableName, _ string, batch map[string]map[string]interface{}) error {
	testpersistencestore.locks[tableName].Lock()
	defer func() {
		testpersistencestore.locks[tableName].Unlock()
	}()

	for rowKey, rowProps := range batch {
		tag := testpersistencestore.getEtag()
		testpersistencestore.datastore[tableName][rowKey] = rowProperties{etag: tag, properties: rowProps}
	}

	return nil
}

func (testpersistencestore *TestPersistenceStore) DeleteBatch(tableName, _ string, rowKeys []string) error {
	testpersistencestore.locks[tableName].Lock()
	defer func() {
		testpersistencestore.locks[tableName].Unlock()
	}()

	for _, rowKey := range rowKeys {
		delete(testpersistencestore.datastore[tableName], rowKey)
	}

	return nil
}

// Query :- Query an element using rowKey and tableName.
func (testpersistencestore *TestPersistenceStore) Query(tableName, rowKey string) (interface{}, int, error) {
	value, _, status, err := testpersistencestore.queryAllPropertiesEtag(tableName, rowKey)
	switch len(value) {
	case 0:
		return nil, status, err
	case 1:
		for _, v := range value {
			return v, status, err
		}
	}
	return value, status, err
}

// QueryInPartition - Query a row in the given partition of the table.
func (testpersistencestore *TestPersistenceStore) QueryInPartition(tableName, partitionKey, rowKey string) (
	interface{}, int, error) {
	value, _, status, err := testpersistencestore.queryAllPropertiesEtag(tableName, rowKey)
	switch len(value) {
	case 0:
		return nil, status, err
	case 1:
		for _, v := range value {
			return v, status, err
		}
	}
	return value, status, err
}

// QueryInPartitionWithEtag - Query a row in the given partition of the table. Returns etag along with first property
func (testpersistencestore *TestPersistenceStore) QueryInPartitionWithEtag(
	tableName, partitionKey, rowKey string) (interface{}, string, int, error) {

	return testpersistencestore.queryPropertyEtag(tableName, rowKey)
}

// QueryInPartitionWithEtagProperties - Query a row in the given partition of the table.
// Returns etag along with all of the properties
func (testpersistencestore *TestPersistenceStore) QueryInPartitionWithEtagProperties(
	tableName, partitionKey, rowKey string) (map[string]interface{}, string, int, error) {

	return testpersistencestore.queryAllPropertiesEtag(tableName, rowKey)
}

// QueryAllRows - Query all rows in a given table.
func (testpersistencestore *TestPersistenceStore) QueryAllRows(tableName string, context interface{}) (
	map[string]interface{}, bool, interface{}, int, error) {
	var status = http.StatusNotFound

	testpersistencestore.locks[tableName].RLock()
	defer testpersistencestore.locks[tableName].RUnlock()
	if _, err := testpersistencestore.datastore[tableName]; err {
		status = http.StatusOK
	}
	queriedTable := make(map[string]interface{})
	for key, rowProps := range testpersistencestore.datastore[tableName] {
		switch len(rowProps.properties) {
		case 0:
			return nil, false, nil, http.StatusExpectationFailed, fmt.Errorf("no properties found for key %v in table %v", key, tableName)
		case 1:
			for _, value := range rowProps.properties {
				queriedTable[key] = value
			}
		default:
			queriedTable[key] = rowProps.properties
		}
	}
	return queriedTable, false, nil, status, nil
}

// QueryAllRowsInPartition - Query all rows in the specific partition of the given table.
func (testpersistencestore *TestPersistenceStore) QueryAllRowsInPartition(
	tableName, partitionKey string, context interface{}) (map[string]interface{}, bool, interface{}, int, error) {
	return testpersistencestore.QueryAllRows(tableName, context)
}

// Query :- Query an element using rowKey and tableName. It returns ONLY the first property
// found in the row as well as etag of the row
func (testpersistencestore *TestPersistenceStore) queryPropertyEtag(tableName, rowKey string) (interface{}, string, int, error) {
	props, tag, status, err := testpersistencestore.queryAllPropertiesEtag(tableName, rowKey)
	if err == nil {
		if len(props) != 1 {
			return nil, "", http.StatusExpectationFailed, fmt.Errorf("More than one property found")
		}

		for _, val := range props {
			return val, tag, status, nil
		}
	}

	return nil, "", status, err
}

// Query :- Query an element using rowKey and tableName. It returns all the properties
// found in the row as well as etag of the row
func (testpersistencestore *TestPersistenceStore) queryAllPropertiesEtag(
	tableName, rowKey string) (map[string]interface{}, string, int, error) {
	testpersistencestore.locks[tableName].RLock()
	defer testpersistencestore.locks[tableName].RUnlock()
	rowProps, success := testpersistencestore.datastore[tableName][rowKey]
	var status = http.StatusNotFound

	if success {
		status = http.StatusOK
		if rowProps.properties == nil {
			return nil, "", http.StatusExpectationFailed, fmt.Errorf("No properties found")
		}

		return rowProps.properties, rowProps.etag, status, nil
	}

	loggershim.LS.Debugf("Failed to query from [%s] with rowKey [%s].", tableName, rowKey)

	return nil, "", status, fmt.Errorf("Failed to query from [%s] with rowKey [%s]", tableName, rowKey)
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

// Induce error in the processing
// This function induces error with provided percent probability
func (testpersistencestore *TestPersistenceStore) induceError(percentProbability int) bool {
	induceError := false
	// Create a random num generator with a new seed every time to
	// avoid deterministic number generation
	randNumGen := rand.New(rand.NewSource(time.Now().UnixNano()))

	if randNumGen.Intn(100) < percentProbability {
		induceError = true
	}

	return induceError
}

// getEtag - Generate a unique tag (string)
func (testpersistencestore *TestPersistenceStore) getEtag() string {
	etag := fmt.Sprintf("%d", time.Now().Add(time.Second*time.Duration(rand.Int63n(10000))).UnixNano())
	return etag
}
