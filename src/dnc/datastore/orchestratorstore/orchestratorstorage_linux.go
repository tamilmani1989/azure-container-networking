package orchestratorstorage

import "dnc/configuration"

// OrchestratorStorage :- Orchestrator's storage object to manage tables in partition
type OrchestratorStorage struct {
	configuration.OrchestratorStorageInfo
}

// Init :- Initialize orchestrator storage
func (orchestratorstorage *OrchestratorStorage) Init(tableToColumnMap map[string]string) {
	return
}

// InsertOrUpdate :- Insert or update an element using rowKey and tablName.
func (orchestratorstorage *OrchestratorStorage) InsertOrUpdate(tableName, rowKey, propertyName string, value interface{}) error {
	return nil
}

// Delete :- Remove an element using rowKey and tableName.
func (orchestratorstorage *OrchestratorStorage) Delete(tableName, rowKey string) error {
	return nil
}

// Query :- Query an element using rowKey and tableName.
func (orchestratorstorage *OrchestratorStorage) Query(tableName, rowKey string) (interface{}, int, error) {
	return nil, 0, nil
}

// QueryAll - Query a row in a given table.
func (orchestratorstorage *OrchestratorStorage) QueryAll(tableName string, context interface{}) (
	map[string]interface{}, bool, interface{}, int, error) {
	return nil, false, nil, 0, nil
}

// KeyExist :- Check if key exists in a given table.
func (orchestratorstorage *OrchestratorStorage) KeyExist(tableName, rowKey string) (bool, error) {
	return false, nil
}
