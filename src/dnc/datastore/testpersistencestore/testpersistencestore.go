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

type queryFilterPredicate func(rowKey string, properties map[string]interface{}) bool

func defaultQueryFilterPredicate(_ string, _ map[string]interface{}) bool {
	return true
}

// TestPersistenceStore :- datastore : map[ TableName ] map [ key ] value
type TestPersistenceStore struct {
	datastore map[string]map[string]rowProperties
	locks     map[string]*sync.RWMutex
	logger    loggershim.LoggerInterface
	// predicates to use as filters for complex functions
	QueryFilterFunc     queryFilterPredicate
	QueryFilterTopNFunc queryFilterPredicate
}

const (
	IPAllocationsTableName   = "IPAllocationsTable"
	ToBeDeletedOwnerProperty = "ToBeDeletedOwner"
)

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

// DeleteWithEtag - Deletes a row in a given table if etag matches
func (s *TestPersistenceStore) DeleteWithEtag(tableName, rowKey string, etag string) error {
	s.locks[tableName].Lock()
	defer func() {
		s.locks[tableName].Unlock()
	}()

	rowProps, retrieved := s.datastore[tableName][rowKey]
	if !retrieved {
		s.logger.Debugf("Ignore the request, as value with rowKey [%s] in %s. doesnt exist", rowKey, tableName)
		return nil
	}

	// Must match the etag
	if etag != rowProps.etag {
		return fmt.Errorf("Caller's Etag does not match with current etag. Row key: %s, table: %s", rowKey, tableName)
	}

	delete(s.datastore[tableName], rowKey)

	s.logger.Debugf("Deleted value with rowKey [%s] in %s.n", rowKey, tableName)
	return nil
}

// DeleteInPartition - Deletes a row in the partition of the given table.
func (s *TestPersistenceStore) DeleteInPartition(tableName, partitionKey, rowKey string) error {
	return s.Delete(tableName, rowKey)
}

// DeleteInPartitionWithEtag - Deletes a row in the partition of the given table if Etag matches.
func (s *TestPersistenceStore) DeleteInPartitionWithEtag(tableName, partitionKey, rowKey, etag string) error {
	s.locks[tableName].Lock()
	defer func() {
		s.locks[tableName].Unlock()
	}()

	rowProps, retrieved := s.datastore[tableName][rowKey]
	if !retrieved {
		return fmt.Errorf("Entity not found: %s, table: %s", rowKey, tableName)
	}

	// Must match the etag
	if etag != rowProps.etag {
		return fmt.Errorf("Caller's Etag does not match with current etag. Row key: %s, table: %s", rowKey, tableName)
	}

	delete(s.datastore[tableName], rowKey)

	s.logger.Debugf("Deleted value with rowKey [%s] in %s.n", rowKey, tableName)
	return nil
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

// UpdateWithEtag - Update (overwrite) a row in a given table if it exists.
// Match with specified etag for update.
func (s *TestPersistenceStore) UpdateWithEtag(tableName, rowKey, etag string, propertyName string, value interface{}) error {
	s.locks[tableName].Lock()
	defer func() {
		s.locks[tableName].Unlock()
	}()

	rowProps, retrieved := s.datastore[tableName][rowKey]
	if !retrieved {
		return fmt.Errorf("Row with key %s not found in table %s", rowKey, tableName)
	}

	// todo fix this later to return specific error type so that caller can retry
	if etag != rowProps.etag {
		return fmt.Errorf("Caller's Etag does not match with current etag. Row key: %s, table: %s", rowKey, tableName)
	}

	props := make(map[string]interface{})
	props[propertyName] = value
	tag := s.getEtag()
	s.datastore[tableName][rowKey] = rowProperties{etag: tag, properties: props}
	s.logger.Debugf("Update row to [%s] with rowKey [%s]. Caller's etag [%s]", tableName, rowKey, etag)
	return nil
}

// Insert - Insert a row in the configured partition of the client
func (s *TestPersistenceStore) Insert(tableName, rowKey, propertyName string, value interface{}) (int, error) {
	props := make(map[string]interface{})
	props[propertyName] = value
	return s.InsertInPartitionProperties(tableName, "", rowKey, props)
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

// Deprecated: Use UpsertEntityBatch.
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

// Deprecated: Use InsertEntityBatch.
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

func (s *TestPersistenceStore) WipePartition(tableName string) error {
	s.locks[tableName].Lock()
	defer func() {
		s.locks[tableName].Unlock()
	}()

	for rowKey, _ := range s.datastore[tableName] {
		delete(s.datastore[tableName], rowKey)
	}

	return nil
}

func (s *TestPersistenceStore) InsertEntityBatch(tableName string, entities []storage.Entity) error {
	table, ok1 := s.datastore[tableName]
	tableLock, ok2 := s.locks[tableName]
	if !ok1 || !ok2 {
		return fmt.Errorf("table %s not initialized", tableName)
	}

	tableLock.Lock()
	defer tableLock.Unlock()

	for _, entity := range entities {
		if _, ok := table[entity.RowKey]; ok {
			return storage.AzureStorageServiceError{
				Message:    fmt.Sprintf("entity with row key %s in table %s already exists", entity.RowKey, tableName),
				StatusCode: http.StatusConflict,
			}
		}
	}

	for _, entity := range entities {
		table[entity.RowKey] = rowProperties{etag: s.getEtag(), properties: entity.Properties}
	}

	return nil
}

func (s *TestPersistenceStore) UpdateEntityBatch(tableName string, entities []storage.Entity) error {
	table, ok1 := s.datastore[tableName]
	tableLock, ok2 := s.locks[tableName]
	if !ok1 || !ok2 {
		return fmt.Errorf("table %s not initialized", tableName)
	}

	tableLock.Lock()
	defer tableLock.Unlock()

	// todo: should error with precondition failed to simulate cosmos
	for _, entity := range entities {
		current, ok := table[entity.RowKey]
		if !ok {
			return fmt.Errorf("entity with row key %s in table %s cannot be updated as it doesn't exist", entity.RowKey, tableName)
		}
		if entity.OdataEtag != current.etag {
			return fmt.Errorf("etag %s for row key %s does not match current etag %s in table %s", entity.OdataEtag, entity.RowKey, current.etag, tableName)
		}
	}

	for _, entity := range entities {
		table[entity.RowKey] = rowProperties{etag: s.getEtag(), properties: entity.Properties}
	}

	return nil
}

func (s *TestPersistenceStore) UpsertEntityBatch(tableName string, entities []storage.Entity) error {
	table, ok1 := s.datastore[tableName]
	tableLock, ok2 := s.locks[tableName]
	if !ok1 || !ok2 {
		return fmt.Errorf("table %s not initialized", tableName)
	}

	tableLock.Lock()
	defer tableLock.Unlock()

	// no double scan, since this method forces an insert or update
	for _, entity := range entities {
		table[entity.RowKey] = rowProperties{etag: s.getEtag(), properties: entity.Properties}
	}

	return nil
}

func (s *TestPersistenceStore) MergeEntityBatch(tableName string, entities []storage.Entity) error {
	table, ok1 := s.datastore[tableName]
	tableLock, ok2 := s.locks[tableName]
	if !ok1 || !ok2 {
		return fmt.Errorf("table %s not initialized", tableName)
	}

	tableLock.Lock()
	defer tableLock.Unlock()

	// todo: should error with precondition failed to simulate cosmos
	for _, entity := range entities {
		current, ok := table[entity.RowKey]
		if !ok {
			return fmt.Errorf("entity with row key %s in table %s cannot be updated as it doesn't exist", entity.RowKey, tableName)
		}
		if entity.OdataEtag != current.etag {
			return fmt.Errorf("etag %s for row key %s does not match current etag %s in table %s", entity.OdataEtag, entity.RowKey, current.etag, tableName)
		}
	}

	for _, entity := range entities {
		mergedProps := table[entity.RowKey].properties
		if mergedProps == nil {
			mergedProps = make(map[string]interface{})
		}
		for prop, val := range entity.Properties {
			mergedProps[prop] = val
		}

		table[entity.RowKey] = rowProperties{etag: s.getEtag(), properties: mergedProps}
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

func (s *TestPersistenceStore) QueryFilter(tableName, _ string) ([]*storage.Entity, error) {
	pred := defaultQueryFilterPredicate
	if s.QueryFilterFunc != nil {
		pred = s.QueryFilterFunc
	}
	return s.queryFilter(tableName, "", pred, 0)
}

func (s *TestPersistenceStore) QueryFilterTopN(tableName, _ string, count int) ([]*storage.Entity, error) {
	pred := defaultQueryFilterPredicate
	if s.QueryFilterTopNFunc != nil {
		pred = s.QueryFilterTopNFunc
	}
	return s.queryFilter(tableName, "", pred, count)
}

func (s *TestPersistenceStore) queryFilter(tableName, _ string, pred queryFilterPredicate, count int) ([]*storage.Entity, error) {
	table, ok1 := s.datastore[tableName]
	tableLock, ok2 := s.locks[tableName]
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("table %s not initialized", tableName)
	}

	tableLock.RLock()
	defer tableLock.RUnlock()

	var results []*storage.Entity
	for key, rowProps := range table {
		if !pred(key, rowProps.properties) {
			continue
		}

		e := storage.Entity{
			Table:        &storage.Table{Name: tableName},
			PartitionKey: "",
			RowKey:       key,
			OdataEtag:    rowProps.etag,
			Properties:   rowProps.properties,
		}

		results = append(results, &e)
		if len(results) == count { // len will always be at least 1, passing count <1 returns all rows
			break
		}
	}

	return results, nil
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

func (s *TestPersistenceStore) AnyRowExists(tableName string) (bool, error) {
	s.locks[tableName].RLock()
	defer s.locks[tableName].RUnlock()

	_, found := s.datastore[tableName]
	if !found {
		return false, fmt.Errorf("tableName %s not found", tableName)
	}

	if len(s.datastore[tableName]) > 0 {
		return true, nil
	}

	return false, nil
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

// getEtag - Generate a unique tag (string)
func (s *TestPersistenceStore) getEtag() string {
	etag := fmt.Sprintf("%d", time.Now().Add(time.Second*time.Duration(rand.Int63n(10000))).UnixNano())
	return etag
}

// Seed is a utility to seed datastore for testing.
// It should only be called before any accesses to the datastore.
func (s *TestPersistenceStore) Seed(tableToEntities map[string][]storage.Entity) {
	for tableName, entities := range tableToEntities {
		_ = s.InsertEntityBatch(tableName, entities)
	}
}
