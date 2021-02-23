package azurestorage

import (
	"crypto/sha256"
	"dnc/aiwrapper"
	"dnc/configuration"
	"dnc/logger/loggershim"
	"dnc/utils"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-container-networking/aitelemetry"
	"github.com/Azure/azure-sdk-for-go/storage"
)

const (
	// default timeout for calls
	defaultTimeout = 30
)

// AzureStorage = Storage object to manage tables in a partition
type AzureStorage struct {
	partitionKey              string // todo: this should not be within this wrapper, but passed by calling code
	tableNameToTableClientMap map[string]storage.TableServiceClient
	maxBatchSize              int
	latencyThresholdMs        int // Max time latency allowed for data store operations before recording the operation latency in the log.
	logger                    loggershim.LoggerInterface
	isCosmosDb                bool
}

// Init - Initialize azure storage.
func NewAzureStorage(config configuration.AzureStorageInfo, logger loggershim.LoggerInterface, tableToColumnMap map[string]string, latencyThresholdMs int, maxBatchSize int) (*AzureStorage, error) {
	logger.Debugf("[Azure Storage] Environment type for azure store settings %s", config.Environment)

	storageClient, err := utils.GetStorageClient(config.AccountName, config.Key, config.Environment, config.IsUsingCosmosDb)
	if err != nil {
		return nil, fmt.Errorf("Create client failed: %v", err)
	}

	tblCl := storageClient.GetTableService()

	azureStorage := AzureStorage{
		partitionKey:       config.PartitionKey,
		maxBatchSize:       maxBatchSize,
		latencyThresholdMs: latencyThresholdMs,
		logger:             logger,
		isCosmosDb:         config.IsUsingCosmosDb,
	}

	logger.Debugf("[Azure Storage] Max transaction batch size set to %d", maxBatchSize)

	azureStorage.initTableNameToTableClientMap(tblCl, tableToColumnMap)

	if len(strings.TrimSpace(config.RegionalStorageConnectionString)) > 0 {
		if err := azureStorage.initRegionalStorage(config.RegionalStorageConnectionString); err != nil {
			return nil, fmt.Errorf("Could not init regional storage: %v", err)
		}
	}

	logger.Debugf("[Azure Storage] Ensuring table schema")
	if err := azureStorage.ensureSchema(tableToColumnMap); err != nil {
		return nil, fmt.Errorf("Could not ensure table schema: %v", err)
	}

	return &azureStorage, nil
}

func (a *AzureStorage) ensureSchema(tables map[string]string) error {
	for tableName := range tables {
		tblSvc := a.tableNameToTableClientMap[tableName]
		tbl := tblSvc.GetTableReference(tableName)
		if err := tbl.Create(defaultTimeout, storage.EmptyPayload, nil); err != nil {
			switch e := err.(type) {
			case storage.AzureStorageServiceError:
				if e.StatusCode == http.StatusConflict {
					a.logger.Debugf("[Azure Storage] Table %s already exists", tableName)
					continue
				}
				return fmt.Errorf("[Azure Storage] Could not create table %s: %+v", tableName, e)
			default:
				return fmt.Errorf("[Azure Storage] Could not create table %s: %v", tableName, err)
			}
		}
		a.logger.Debugf("[Azure Storage] Table %s created successfully", tableName)
	}
	return nil
}

// initRegionalStorage - Initialize the table client needed to access the regional storage
func (a *AzureStorage) initRegionalStorage(connectionString string) error {
	a.logger.Debugf("[Azure Storage] InitRegionalStorage with %x", sha256.Sum256([]byte(connectionString)))
	regionalStorageClient, err := storage.NewClientFromConnectionString(connectionString)
	if err != nil {
		return fmt.Errorf("Create regional client failed: %v", err)
	}

	regionalTableClient := regionalStorageClient.GetTableService()
	a.tableNameToTableClientMap["IPAllocationsTable"] = regionalTableClient
	a.logger.Debugf("[Azure Storage] Updated settings for table IPAllocationsTable to use regional storage")
	return nil
}

// initTableNameToTableClientMap - Set tableClient as default client to all the tables
func (a *AzureStorage) initTableNameToTableClientMap(tableClient storage.TableServiceClient, tableToColumnMap map[string]string) {
	a.tableNameToTableClientMap = make(map[string]storage.TableServiceClient)
	for tableName := range tableToColumnMap {
		a.tableNameToTableClientMap[tableName] = tableClient
	}
}

func (a *AzureStorage) recordProcessingTime(start time.Time, operation, tableName, partitionKey, rowKey string) {
	timeElapsedMs := time.Since(start).Milliseconds()
	if timeElapsedMs > int64(a.latencyThresholdMs) {
		a.logger.Debugf("[AzureStorage] Operation: %s, Table: %s, PartitionKey: %s,"+
			" rowKey: %s, TimeMs: %v", operation, tableName, partitionKey, rowKey, timeElapsedMs)

		if a.isCosmosDb {
			sendDataStoreMetric(aiwrapper.DataStoreOperationTimeMetric, float64(timeElapsedMs), operation, tableName)
		}
	}
}

// todo: ai wrapper should be passed as a dependency
func sendDataStoreMetric(metricName string, metricValue float64, operation, tableName string) {
	metric := aitelemetry.Metric{
		Name:             metricName,
		Value:            metricValue,
		CustomDimensions: make(map[string]string),
	}

	metric.CustomDimensions[aiwrapper.OperationStr] = operation
	metric.CustomDimensions[aiwrapper.TableNameStr] = tableName
	go aiwrapper.SendMetric(metric)
}

// Delete - Deletes a row in a given table.
func (a *AzureStorage) Delete(tableName, rowKey string) error {
	defer a.recordProcessingTime(time.Now(), "Delete", tableName, a.partitionKey, rowKey)

	tableClient, ok := a.tableNameToTableClientMap[tableName]
	if !ok {
		return fmt.Errorf("Cannot find valid table client for table %s", tableName)
	}
	table := tableClient.GetTableReference(tableName)
	entity := table.GetEntityReference(a.partitionKey, rowKey)

	return entity.Delete(true, nil)
}

// DeleteWithEtag - Deletes a row in a given table if etag matches
func (a *AzureStorage) DeleteWithEtag(tableName, rowKey string, etag string) error {
	if etag == "" {
		return fmt.Errorf("DeleteWithEtag Etag must not be empty %s, rowKey %s", tableName, rowKey)
	}

	defer a.recordProcessingTime(time.Now(), "DeleteWithEtag", tableName, a.partitionKey, rowKey)

	tableClient, ok := a.tableNameToTableClientMap[tableName]
	if !ok {
		return fmt.Errorf("Cannot find valid table client for table %s", tableName)
	}
	table := tableClient.GetTableReference(tableName)
	entity := table.GetEntityReference(a.partitionKey, rowKey)
	entity.OdataEtag = etag

	return entity.Delete(false, nil)
}

// DeleteInPartition - Deletes a row in the partition of the given table.
func (a *AzureStorage) DeleteInPartition(tableName, partitionKey, rowKey string) error {
	defer a.recordProcessingTime(time.Now(), "DeleteInPartition", tableName, partitionKey, rowKey)

	tableClient, ok := a.tableNameToTableClientMap[tableName]
	if !ok {
		return fmt.Errorf("Cannot find valid table client for table %s", tableName)
	}
	table := tableClient.GetTableReference(tableName)
	entity := table.GetEntityReference(partitionKey, rowKey)
	return entity.Delete(true, nil)
}

// DeleteInPartitionWithEtag - Deletes a row in the partition of the given table if Etag matches.
func (a *AzureStorage) DeleteInPartitionWithEtag(tableName, partitionKey, rowKey, etag string) error {
	defer a.recordProcessingTime(time.Now(), "DeleteInPartition", tableName, partitionKey, rowKey)

	tableClient, ok := a.tableNameToTableClientMap[tableName]
	if !ok {
		return fmt.Errorf("Cannot find valid table client for table %s", tableName)
	}
	table := tableClient.GetTableReference(tableName)
	entity := table.GetEntityReference(partitionKey, rowKey)
	entity.OdataEtag = etag

	return entity.Delete(false, nil)
}

// Insert only - Inserts a row in a given table.
// This will fail if the entity already exists
func (a *AzureStorage) Insert(tableName, rowKey, propertyName string, value interface{}) (int, error) {
	defer a.recordProcessingTime(time.Now(), "Insert", tableName, a.partitionKey, rowKey)

	properties := make(map[string]interface{})
	properties[propertyName] = value
	return a.insertRowInPartition(tableName, a.partitionKey, rowKey, properties)
}

// InsertOrUpdate - InsertOrUpdate a row in a given table.
func (a *AzureStorage) InsertOrUpdate(tableName, rowKey, propertyName string, value interface{}) error {
	defer a.recordProcessingTime(time.Now(), "InsertOrUpdate", tableName, a.partitionKey, rowKey)

	properties := map[string]interface{}{
		propertyName: value,
	}
	return a.insertOrUpdateRow(tableName, a.partitionKey, rowKey, properties)
}

// UpdateInPartition - Update (overwrite) a row in a given table if it exists.
func (a *AzureStorage) UpdateInPartition(tableName, partitionKey, rowKey string, properties map[string]interface{}) error {
	defer a.recordProcessingTime(time.Now(), "UpdateInPartition", tableName, partitionKey, rowKey)

	return a.updateRowWithEtag(tableName, partitionKey, rowKey, "", properties)
}

// UpdateInPartitionWithEtag - Update (overwrite) a row in a given table if it exists.
// Match with specified etag for update.
func (a *AzureStorage) UpdateInPartitionWithEtag(tableName, partitionKey, rowKey, etag string, properties map[string]interface{}) error {
	defer a.recordProcessingTime(time.Now(), "UpdateInPartitionWithEtag", tableName, partitionKey, rowKey)

	return a.updateRowWithEtag(tableName, partitionKey, rowKey, etag, properties)
}

// InsertInPartition - Insert a row in the specific partition of the given table with one property
func (a *AzureStorage) InsertInPartition(tableName, partitionKey, rowKey, propertyName string, value interface{}) (int, error) {
	defer a.recordProcessingTime(time.Now(), "InsertInPartition", tableName, partitionKey, rowKey)

	properties := make(map[string]interface{})
	properties[propertyName] = value
	return a.insertRowInPartition(tableName, partitionKey, rowKey, properties)
}

// InsertInPartitionProperties - Insert a row in the specific partition of the given table
// with multiple properties
func (a *AzureStorage) InsertInPartitionProperties(tableName, partitionKey, rowKey string, properties map[string]interface{}) (int, error) {
	defer a.recordProcessingTime(time.Now(), "InsertInPartitionProperties", tableName, partitionKey, rowKey)

	return a.insertRowInPartition(tableName, partitionKey, rowKey, properties)
}

// InsertOrUpdateProperties - Inserts or updates a row of the given table with multiple properties
func (a *AzureStorage) InsertOrUpdateProperties(tableName, partitionKey, rowKey string, properties map[string]interface{}) error {
	defer a.recordProcessingTime(time.Now(), "InsertOrUpdateProperties", tableName, partitionKey, rowKey)

	return a.insertOrUpdateRow(tableName, partitionKey, rowKey, properties)
}

// UpdateWithEtag - Update (overwrite) a row in a given table if it exists.
// Match with specified etag for update.
// todo export list of properties and instead of update, expose MergeWithEtag
func (a *AzureStorage) UpdateWithEtag(tableName, rowKey, etag string, propertyName string, value interface{}) error {
	if etag == "" {
		return fmt.Errorf("UpdateWithEtag Etag must not be empty %s, rowKey %s", tableName, rowKey)
	}
	defer a.recordProcessingTime(time.Now(), "UpdateInPartitionWithEtag", tableName, a.partitionKey, rowKey)
	properties := make(map[string]interface{})
	properties[propertyName] = value
	return a.updateRowWithEtag(tableName, a.partitionKey, rowKey, etag, properties)
}

// Deprecated: Use UpsertEntityBatch.
// UpsertBatch inserts or replaces a batch of rows (row key to properties) for a given table and partition key
// There is a limit of 100 items per batch
func (a *AzureStorage) UpsertBatch(tableName string, batchItems map[string]map[string]interface{}) error {
	tableClient, ok := a.tableNameToTableClientMap[tableName]
	if !ok {
		return fmt.Errorf("cannot find valid table client for table %s", tableName)
	}

	table := tableClient.GetTableReference(tableName)

	batch := table.NewBatch()
	for rowKey, rowProperties := range batchItems {
		entity := table.GetEntityReference(a.partitionKey, rowKey)
		entity.Properties = rowProperties
		// todo: is it correct to always use If-Match:*? not providing an etag basically forces the upsert
		// entity.OdataEtag = ""
		batch.InsertOrReplaceEntity(entity, false) // underlying sdk code does not ever use the second param
	}

	return batch.ExecuteBatch()
}

// Deprecated: Use InsertEntityBatch.
// InsertBatch inserts a batch of rows for a given table and partition key
// The key in the batch represents the row key
// The value in the batch (map[string]interface{}) represents row properties and their respective values
// Documented limit is 100, but in practice it is over 1000
func (a *AzureStorage) InsertBatch(tableName, partitionKey string, batchItems map[string]map[string]interface{}) error {
	if len(batchItems) > a.maxBatchSize {
		return fmt.Errorf("batch size %d exceeds limit of %d", len(batchItems), a.maxBatchSize)
	}

	tableClient, ok := a.tableNameToTableClientMap[tableName]
	if !ok {
		return fmt.Errorf("cannot find valid table client for table %s", tableName)
	}

	table := tableClient.GetTableReference(tableName)

	batch := table.NewBatch()
	for rowKey, rowProperties := range batchItems {
		entity := table.GetEntityReference(partitionKey, rowKey)
		entity.Properties = rowProperties
		batch.InsertEntity(entity)
	}

	return batch.ExecuteBatch()
}

// DeleteBatch deletes a batch of rows in a given table based on the given partition and row keys
// There is a limit of 100 items per batch
func (a *AzureStorage) DeleteBatch(tableName, partitionKey string, rowKeys []string) error {
	if len(rowKeys) > a.maxBatchSize {
		return fmt.Errorf("batch size should not exceed %d", a.maxBatchSize)
	}

	tableClient, ok := a.tableNameToTableClientMap[tableName]
	if !ok {
		return fmt.Errorf("cannot find valid table client for table %s", tableName)
	}

	table := tableClient.GetTableReference(tableName)

	batch := table.NewBatch()
	for _, rowKey := range rowKeys {
		entity := table.GetEntityReference(partitionKey, rowKey)
		batch.DeleteEntityByForce(entity, true)
	}

	return batch.ExecuteBatch()
}

// Deprecated: Use MergeEntityBatch.
// TODO: add test for MergeBatch function https://msazure.visualstudio.com/One/_workitems/edit/9077936
// TODO: handle ETags for Merge https://msazure.visualstudio.com/One/_workitems/edit/9132517
// If the entity's ETag differs from that specified with the merge request, the merge operation fails with status
// code 412 (Precondition Failed). This error indicates that the entity has been changed on the server since it was
// retrieved. To resolve this error, retrieve the entity again and reissue the request.
func (a *AzureStorage) MergeBatch(tableName, partitionKey string, batchItems map[string]map[string]interface{}) error {
	if len(batchItems) > a.maxBatchSize {
		return fmt.Errorf("batch size %d exceeds limit of %d", len(batchItems), a.maxBatchSize)
	}

	tableClient, ok := a.tableNameToTableClientMap[tableName]
	if !ok {
		return fmt.Errorf("cannot find valid table client for table %s", tableName)
	}

	table := tableClient.GetTableReference(tableName)

	batch := table.NewBatch()
	for rowKey, rowProperties := range batchItems {
		entity := table.GetEntityReference(partitionKey, rowKey)
		entity.Properties = rowProperties
		batch.MergeEntity(entity)
	}

	return batch.ExecuteBatch()
}

func (a *AzureStorage) InsertEntityBatch(tableName string, entities []storage.Entity) error {
	return a.batchOperation(tableName, entities, storage.InsertOp)
}

/*
	note: from https://docs.microsoft.com/en-us/rest/api/storageservices/update-entity2#remarks:
	"If the If-Match header is missing from the request in versions 2011-08-18 and later, then the
	service performs an Insert Or Replace Entity (upsert) operation."

    This means if we use update, we must ALWAYS set an etag value in order to synchronize concurrent
    requests properly. Failing to provide one will not result in failure, and will override any changes
    made in other sessions (lost update problem).
*/

func (a *AzureStorage) UpdateEntityBatch(tableName string, entities []storage.Entity) error {
	return a.batchOperation(tableName, entities, storage.ReplaceOp)
}

func (a *AzureStorage) UpsertEntityBatch(tableName string, entities []storage.Entity) error {
	return a.batchOperation(tableName, entities, storage.InsertOrReplaceOp)
}

func (a *AzureStorage) MergeEntityBatch(tableName string, entities []storage.Entity) error {
	return a.batchOperation(tableName, entities, storage.MergeOp)
}

func (a *AzureStorage) batchOperation(tableName string, entities []storage.Entity, op storage.Operation) error {
	if len(entities) > a.maxBatchSize {
		return fmt.Errorf("batch op %v: batch size %d exceeds limit of %d", op, len(entities), a.maxBatchSize)
	}

	tableClient, ok := a.tableNameToTableClientMap[tableName]
	if !ok {
		return fmt.Errorf("cannot find valid table client for table %s", tableName)
	}

	table := tableClient.GetTableReference(tableName)

	batch := table.NewBatch()
	for i := range entities {
		e := entities[i]
		// ramiro's opinion: the partition in which we are storing things should be explicit at the repo layer.
		// maintaining a partition in what otherwise should be a thin wrapper over the azure sdk is confusing, since
		// we don't always use it, and will be troublesome as well if in the future we need to have repos that manage
		// more than one partition (similar to ip allocations).
		// todo: storage entities should be input with partitions, but since that may require more refactoring, we
		//  use the wrapper's partition if the entity doesn't have one.
		if e.PartitionKey == "" {
			e.PartitionKey = a.partitionKey
		}

		e.Table = table

		switch op {
		case storage.InsertOp:
			batch.InsertEntity(&e)
		case storage.ReplaceOp:
			batch.ReplaceEntity(&e)
		case storage.InsertOrReplaceOp:
			batch.InsertOrReplaceEntity(&e, false) // second parameter not used by sdk
		case storage.MergeOp:
			batch.MergeEntity(&e)
		default:
			return fmt.Errorf("unsupported operation %v", op)
		}
	}

	return batch.ExecuteBatch()
}

// Query - Query a row in a given table.
func (a *AzureStorage) Query(tableName, rowKey string) (*storage.Entity, int, error) {
	defer a.recordProcessingTime(time.Now(), "Query", tableName, a.partitionKey, rowKey)

	return a.getEntity(tableName, a.partitionKey, rowKey)
}

// QueryInPartition - Query a row in the given partition of the table.
func (a *AzureStorage) QueryInPartition(tableName, partitionKey, rowKey string) (*storage.Entity, int, error) {
	defer a.recordProcessingTime(time.Now(), "QueryInPartition", tableName, partitionKey, rowKey)

	return a.getEntity(tableName, partitionKey, rowKey)
}

// QueryAllRows - Query a row in a given table.
func (a *AzureStorage) QueryAllRows(tableName string, context interface{}) ([]*storage.Entity, bool, interface{}, int, error) {
	defer a.recordProcessingTime(time.Now(), "QueryAllRows", tableName, a.partitionKey, "NA")

	return a.getEntities(tableName, a.partitionKey, context)
}

// QueryAllRowsInPartition - Query all rows in the specific partition of the given table.
func (a *AzureStorage) QueryAllRowsInPartition(tableName, partitionKey string, context interface{}) ([]*storage.Entity, bool, interface{}, int, error) {
	defer a.recordProcessingTime(time.Now(), "QueryAllRowsInPartition", tableName, partitionKey, "NA")

	return a.getEntities(tableName, partitionKey, context)
}

// QueryFilter - Query rows using a filter.
func (a *AzureStorage) QueryFilter(tableName, filter string) ([]*storage.Entity, error) {
	defer a.recordProcessingTime(time.Now(), "QueryFilter", tableName, "NA", "NA")

	return a.queryFilter(tableName, filter)
}

// QueryFilter - Query rows using a filter format and arguments.
func (a *AzureStorage) QueryFilterf(tableName, filterFmt string, args ...interface{}) ([]*storage.Entity, error) {
	defer a.recordProcessingTime(time.Now(), "QueryFilterf", tableName, "NA", "NA")

	return a.queryFilter(tableName, fmt.Sprintf(filterFmt, args...))
}

// todo: need to determine a max size for filter, since its sent over a GET call to the storage API and fails if it goes over
func (a *AzureStorage) queryFilter(tableName, filter string) ([]*storage.Entity, error) {
	tableClient, ok := a.tableNameToTableClientMap[tableName]
	if !ok {
		return nil, fmt.Errorf("cannot find valid table client for table %s", tableName)
	}

	table := tableClient.GetTableReference(tableName)

	queryOptions := storage.QueryOptions{Filter: filter}

	queryResult, err := table.QueryEntities(defaultTimeout, storage.FullMetadata, &queryOptions)
	if err != nil {
		return nil, err
	}
	entities := queryResult.Entities

	for queryResult.NextLink != nil {
		nextResults, err := queryResult.NextResults(nil)
		if err != nil {
			return nil, err
		}
		entities = append(entities, nextResults.Entities...)
		queryResult = nextResults
	}

	return entities, nil
}

func (a *AzureStorage) QueryFilterTopN(tableName, filter string, count int) ([]*storage.Entity, error) {
	defer a.recordProcessingTime(time.Now(), "QueryFilterTopN", tableName, "NA", "NA")

	tableClient, ok := a.tableNameToTableClientMap[tableName]
	if !ok {
		return nil, fmt.Errorf("cannot find valid table client for table %s", tableName)
	}

	table := tableClient.GetTableReference(tableName)

	queryOptions := storage.QueryOptions{Filter: filter}

	result, err := table.QueryEntities(defaultTimeout, storage.FullMetadata, &queryOptions)
	if err != nil {
		return nil, err
	}

	var entities []*storage.Entity

	for i := 0; i < len(result.Entities); i++ {
		entities = append(entities, result.Entities[i])
		if len(entities) == count {
			return entities, nil
		}
	}

	for result.NextLink != nil {
		result, err = result.NextResults(nil)
		if err != nil {
			return nil, err
		}

		for i := 0; i < len(result.Entities); i++ {
			entities = append(entities, result.Entities[i])
			if len(entities) == count {
				return entities, nil
			}
		}
	}

	return entities, nil
}

func (a *AzureStorage) insertOrUpdateRow(tableName, partitionKey, rowKey string, properties map[string]interface{}) error {
	tableClient, ok := a.tableNameToTableClientMap[tableName]
	if !ok {
		return fmt.Errorf("Cannot find valid table client for table %s", tableName)
	}
	table := tableClient.GetTableReference(tableName)
	entity := table.GetEntityReference(partitionKey, rowKey)
	entity.Properties = properties
	err := entity.InsertOrReplace(nil)
	return err
}

func (a *AzureStorage) updateRowWithEtag(tableName, partitionKey, rowKey, etag string, properties map[string]interface{}) error {
	tableClient, ok := a.tableNameToTableClientMap[tableName]
	if !ok {
		return fmt.Errorf("Cannot find valid table client for table %s", tableName)
	}
	table := tableClient.GetTableReference(tableName)
	entity := table.GetEntityReference(partitionKey, rowKey)

	// todo: this method should always be called with a non-empty etag.
	//  by fetching the entity when etag is not provided (wasteful operation if etag is non-empty)
	//  a race condition arises, since the latest entity state could have been modified between the time
	//  the properties passed to this function were computed and this fetch.
	//  if the intention is to always update, then a force update should be used, and also made evident
	//  to calling code up the stack.
	err := entity.Get(defaultTimeout, storage.FullMetadata, nil)
	if err != nil {
		return err
	}

	if etag != "" {
		entity.OdataEtag = etag
	}

	entity.Properties = properties
	err = entity.Update(false, nil)
	return err
}

func (a *AzureStorage) insertRowInPartition(tableName, partitionKey, rowKey string, properties map[string]interface{}) (int, error) {
	tableClient, ok := a.tableNameToTableClientMap[tableName]
	if !ok {
		return http.StatusInternalServerError, fmt.Errorf("Cannot find valid table client for table %s", tableName)
	}
	table := tableClient.GetTableReference(tableName)
	entity := table.GetEntityReference(partitionKey, rowKey)
	entity.Properties = properties

	if err := entity.Insert(storage.FullMetadata, nil); err != nil {
		storageErr, ok := err.(storage.AzureStorageServiceError)
		if ok {
			return storageErr.StatusCode, err
		}

		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func (a *AzureStorage) getEntity(tableName string, partitionKey string, rowKey string) (*storage.Entity, int, error) {
	tableClient, ok := a.tableNameToTableClientMap[tableName]
	if !ok {
		return nil, http.StatusInternalServerError, fmt.Errorf("Cannot find valid table client for table %s", tableName)
	}
	table := tableClient.GetTableReference(tableName)
	entity := table.GetEntityReference(partitionKey, rowKey)
	err := entity.Get(defaultTimeout, storage.FullMetadata, nil)
	if err != nil {
		storageErr, ok := err.(storage.AzureStorageServiceError)
		if ok {
			return nil, storageErr.StatusCode, err
		}

		return nil, http.StatusInternalServerError, err
	}

	return entity, http.StatusOK, nil
}

func (a *AzureStorage) getEntities(tableName string, partitionKey string, context interface{}) ([]*storage.Entity, bool, interface{}, int, error) {
	var (
		entitiesResult *storage.EntityQueryResult
		err            error
	)

	if context == nil {
		tableClient, ok := a.tableNameToTableClientMap[tableName]
		if !ok {
			return nil, false, nil, http.StatusInternalServerError, fmt.Errorf("Cannot find valid table client for table %s", tableName)
		}
		table := tableClient.GetTableReference(tableName)
		queryOptions := storage.QueryOptions{
			Filter: "PartitionKey eq '" + partitionKey + "'",
		}

		entitiesResult, err = table.QueryEntities(defaultTimeout, storage.FullMetadata, &queryOptions)
	} else {
		entitiesResult = context.(*storage.EntityQueryResult)
		entitiesResult, err = entitiesResult.NextResults(nil)
	}
	if err != nil {
		storageErr, ok := err.(storage.AzureStorageServiceError)
		if ok {
			return nil, false, nil, storageErr.StatusCode, err
		}
		return nil, false, nil, http.StatusInternalServerError, err
	}

	result := entitiesResult.Entities

	var hasMoreRows = false
	var returnContext interface{}
	if entitiesResult.NextLink != nil {
		hasMoreRows = true
		returnContext = entitiesResult
	}
	return result, hasMoreRows, returnContext, http.StatusOK, nil
}
