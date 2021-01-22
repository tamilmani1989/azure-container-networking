package testpersistencestore

import (
	"dnc/logger/loggershim"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
)

type rowProperties struct {
	etag       string
	properties map[string]interface{}
}

// TestPersistenceStore :- datastore : map[ TableName ] map [ key ] value
type TestPersistenceStore struct {
	datastore map[string]map[string]rowProperties
	locks     map[string]*sync.RWMutex
	logger    loggershim.LoggerInterface
}

const induceErrorPercent = 5 // Induce 5% error for a particular operation

// Init :- Initialize a TestPersistenceStore struct for testing purposes.
func (s *TestPersistenceStore) Init(tableToColumnMap map[string]string, logger loggershim.LoggerInterface) {
	s.datastore = make(map[string]map[string]rowProperties)
	s.locks = make(map[string]*sync.RWMutex)
	s.logger = logger

	for tableName := range tableToColumnMap {
		s.datastore[tableName] = make(map[string]rowProperties)
		s.locks[tableName] = &sync.RWMutex{}
	}
}

// Delete :- Remove an element using rowKey and tableName.
func (s *TestPersistenceStore) Delete(tableName, rowKey string) error {
	s.locks[tableName].Lock()
	delete(s.datastore[tableName], rowKey)
	s.locks[tableName].Unlock()
	s.logger.Debugf("Deleted value with rowKey [%s] in %s.n", rowKey, tableName)
	return nil
}

// DeleteInPartition - Deletes a row in the partition of the given table.
func (s *TestPersistenceStore) DeleteInPartition(tableName, partitionKey, rowKey string) error {
	return s.Delete(tableName, rowKey)
}

// InsertOrUpdate :- Insert or update an element using rowKey and tableName.
func (s *TestPersistenceStore) InsertOrUpdate(tableName, rowKey, propertyName string, value interface{}) error {
	props := make(map[string]interface{})
	props[propertyName] = value
	tag := s.getEtag()

	s.locks[tableName].Lock()
	s.datastore[tableName][rowKey] = rowProperties{etag: tag, properties: props}
	s.locks[tableName].Unlock()

	s.logger.Debugf("Assigned [%+v] to [%s] with rowKey [%s].", value, tableName, rowKey)
	return nil
}

// UpdateInPartition - Update a row in the given table, if row already exists
func (s *TestPersistenceStore) UpdateInPartition(tableName, partitionKey, rowKey string, properties map[string]interface{}) error {
	return s.UpdateInPartitionWithEtag(tableName, partitionKey, rowKey, "", properties)
}

// UpdateInPartitionWithEtag - Update (overwrite) a row in a given table if it exists.
// Match with specified etag for update.
func (s *TestPersistenceStore) UpdateInPartitionWithEtag(tableName, partitionKey, rowKey, etag string, properties map[string]interface{}) error {
	s.locks[tableName].Lock()
	defer func() {
		s.locks[tableName].Unlock()
	}()

	rowProps, retrieved := s.datastore[tableName][rowKey]
	if !retrieved {
		return fmt.Errorf("Row with key %s not found in table %s", rowKey, tableName)
	}

	if etag != "" && etag != "*" && etag != rowProps.etag {
		return fmt.Errorf("Caller's Etag does not match with current etag. Row key: %s, table: %s", rowKey, tableName)
	}

	tag := s.getEtag()
	s.datastore[tableName][rowKey] = rowProperties{etag: tag, properties: properties}
	s.logger.Debugf("Update row to [%s] with rowKey [%s]. Caller's etag [%s]", tableName, rowKey, etag)
	return nil
}

// InsertInPartition - Insert a row in the specific partition of the given table.
func (s *TestPersistenceStore) InsertInPartition(tableName, partitionKey, rowKey, propertyName string, value interface{}) (int, error) {
	props := make(map[string]interface{})
	props[propertyName] = value
	return s.InsertInPartitionProperties(tableName, partitionKey, rowKey, props)
}

// InsertInPartitionProperties - Insert a row in the specific partition of the given table with multiple properties
// If row already exists, it will be updated (overwritten)
func (s *TestPersistenceStore) InsertInPartitionProperties(tableName, partitionKey, rowKey string, properties map[string]interface{}) (int, error) {
	s.locks[tableName].Lock()
	defer func() {
		s.locks[tableName].Unlock()
	}()

	_, retrieved := s.datastore[tableName][rowKey]
	if retrieved {
		return 0, fmt.Errorf("Row already exists with key %s in table %s", rowKey, tableName)
	}

	tag := s.getEtag()
	s.datastore[tableName][rowKey] = rowProperties{etag: tag, properties: properties}
	s.logger.Debugf("Inserted row to [%s] with rowKey [%s].", tableName, rowKey)

	return http.StatusOK, nil
}

// InsertOrUpdateProperties - Inserts or updates a row of the given table with multiple properties
func (s *TestPersistenceStore) InsertOrUpdateProperties(tableName, partitionKey, rowKey string, properties map[string]interface{}) error {
	_, httpStatusCode, _ := s.Query(tableName, rowKey)
	if httpStatusCode == http.StatusNotFound {
		_, err := s.InsertInPartitionProperties(tableName, partitionKey, rowKey, properties)
		return err
	}

	return s.UpdateInPartitionWithEtag(tableName, partitionKey, rowKey, "", properties)
}

func (s *TestPersistenceStore) UpsertBatch(tableName string, batch map[string]map[string]interface{}) error {
	s.locks[tableName].Lock()
	defer func() {
		s.locks[tableName].Unlock()
	}()

	for rowKey, rowProps := range batch {
		tag := s.getEtag()
		s.datastore[tableName][rowKey] = rowProperties{etag: tag, properties: rowProps}
	}

	return nil
}

func (s *TestPersistenceStore) InsertBatch(tableName, _ string, batch map[string]map[string]interface{}) error {
	s.locks[tableName].Lock()
	defer func() {
		s.locks[tableName].Unlock()
	}()

	for rowKey, rowProps := range batch {
		tag := s.getEtag()
		s.datastore[tableName][rowKey] = rowProperties{etag: tag, properties: rowProps}
	}

	return nil
}

func (s *TestPersistenceStore) DeleteBatch(tableName, _ string, rowKeys []string) error {
	s.locks[tableName].Lock()
	defer func() {
		s.locks[tableName].Unlock()
	}()

	for _, rowKey := range rowKeys {
		delete(s.datastore[tableName], rowKey)
	}

	return nil
}

// TODO: add tests for MergeBatch function https://msazure.visualstudio.com/One/_workitems/edit/9077936
func (testpersistencestore *TestPersistenceStore) MergeBatch(tableName, _ string, batch map[string]map[string]interface{}) error {
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

// Query :- Query an element using rowKey and tableName.
func (s *TestPersistenceStore) Query(tableName, rowKey string) (*storage.Entity, int, error) {
	return s.queryAllPropertiesEtag(tableName, rowKey)
}

// QueryInPartition - Query a row in the given partition of the table.
func (s *TestPersistenceStore) QueryInPartition(tableName, partitionKey, rowKey string) (*storage.Entity, int, error) {
	return s.queryAllPropertiesEtag(tableName, rowKey)
}

// QueryFilter - Query rows using a filter
func (s *TestPersistenceStore) QueryFilter(tableName, _ string) ([]*storage.Entity, error) {
	// todo: not a straightforward way to mock more sophisticated query functionality
	// for now returning all the entries from the datastore.
	entities, _, _, _, _ := s.QueryAllRows(tableName, nil)
	return entities, nil
}

// QueryAllRows - Query all rows in a given table.
func (s *TestPersistenceStore) QueryAllRows(tableName string, context interface{}) ([]*storage.Entity, bool, interface{}, int, error) {
	var status = http.StatusNotFound

	s.locks[tableName].RLock()
	defer s.locks[tableName].RUnlock()
	if _, err := s.datastore[tableName]; err {
		status = http.StatusOK
	}
	var out []*storage.Entity
	for key, rowProps := range s.datastore[tableName] {
		e := storage.Entity{
			Table:        &storage.Table{Name: tableName},
			PartitionKey: "", // todo: should this be filled?
			RowKey:       key,
			OdataEtag:    rowProps.etag,
			Properties:   rowProps.properties,
		}

		out = append(out, &e)
	}
	return out, false, nil, status, nil
}

// QueryAllRowsInPartition - Query all rows in the specific partition of the given table.
func (s *TestPersistenceStore) QueryAllRowsInPartition(tableName, partitionKey string, context interface{}) ([]*storage.Entity, bool, interface{}, int, error) {
	return s.QueryAllRows(tableName, context)
}

// Query :- Query an element using rowKey and tableName. It returns all the properties
// found in the row as well as etag of the row
func (s *TestPersistenceStore) queryAllPropertiesEtag(tableName, rowKey string) (*storage.Entity, int, error) {
	s.locks[tableName].RLock()
	defer s.locks[tableName].RUnlock()

	rowProps, found := s.datastore[tableName][rowKey]
	if !found {
		s.logger.Debugf("Failed to query from [%s] with rowKey [%s].", tableName, rowKey)
		return nil, http.StatusNotFound, fmt.Errorf("Failed to query from [%s] with rowKey [%s]", tableName, rowKey)
	}

	e := storage.Entity{
		Table:        &storage.Table{Name: tableName},
		PartitionKey: "", // todo: should this be filled?
		RowKey:       rowKey,
		OdataEtag:    rowProps.etag,
		Properties:   rowProps.properties,
	}

	return &e, http.StatusOK, nil
}

// KeyExist :- Check if key exists in a given table.
func (s *TestPersistenceStore) KeyExist(tableName, rowKey string) (bool, error) {
	s.locks[tableName].RLock()
	_, retrieved := s.datastore[tableName][rowKey]
	s.locks[tableName].RUnlock()

	logStr := "does not exist"
	if retrieved {
		logStr = "exists"
	}

	s.logger.Debugf("rowKey [%s] %s in [%s].", rowKey, logStr, tableName)
	return retrieved, nil
}

// Induce error in the processing
// This function induces error with provided percent probability
func (s *TestPersistenceStore) induceError(percentProbability int) bool {
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
func (s *TestPersistenceStore) getEtag() string {
	etag := fmt.Sprintf("%d", time.Now().Add(time.Second*time.Duration(rand.Int63n(10000))).UnixNano())
	return etag
}
