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
	partitionKey              string
	tableNameToTableClientMap map[string]storage.TableServiceClient
	maxBatchSize              int
	latencyThresholdMs        int // Max time latency allowed for data store operations before recording the operation latency in the log.
	logger                    loggershim.LoggerInterface
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
			default:
				return fmt.Errorf("Could not create table %s: %v", tableName, err)
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

		sendDataStoreMetric(aiwrapper.DataStoreOperationTimeMetric, float64(timeElapsedMs), operation, tableName)
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

// InsertOrUpdate - InsertOrUpdate a row in a given table.
func (a *AzureStorage) InsertOrUpdate(tableName, rowKey, propertyName string, value interface{}) error {
	defer a.recordProcessingTime(time.Now(), "InsertOrUpdate", tableName, a.partitionKey, rowKey)

	return a.insertOrUpdateRow(tableName, rowKey, propertyName, value)
}

// UpdateInPartition - Update (overwrite) a row in a given table if it exists.
func (a *AzureStorage) UpdateInPartition(tableName, partitionKey, rowKey string, properties map[string]interface{}) error {
	defer a.recordProcessingTime(time.Now(), "UpdateInPartition", tableName, a.partitionKey, rowKey)

	return a.updateRowWithEtag(tableName, partitionKey, rowKey, "", properties)
}

// UpdateInPartitionWithEtag - Update (overwrite) a row in a given table if it exists.
// Match with specified etag for update.
func (a *AzureStorage) UpdateInPartitionWithEtag(tableName, partitionKey, rowKey, etag string, properties map[string]interface{}) error {
	defer a.recordProcessingTime(time.Now(), "UpdateInPartitionWithEtag", tableName, a.partitionKey, rowKey)

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

// Query - Query a row in a given table.
func (a *AzureStorage) Query(tableName, rowKey string) (interface{}, int, error) {
	defer a.recordProcessingTime(time.Now(), "Query", tableName, a.partitionKey, rowKey)

	return a.getRow(tableName, a.partitionKey, rowKey)
}

// QueryInPartition - Query a row in the given partition of the table.
func (a *AzureStorage) QueryInPartition(tableName, partitionKey, rowKey string) (interface{}, int, error) {
	defer a.recordProcessingTime(time.Now(), "QueryInPartition", tableName, partitionKey, rowKey)

	return a.getRow(tableName, partitionKey, rowKey)
}

// QueryInPartitionWithEtag - Query a row in the given partition of the table. Returns etag along with first property
func (a *AzureStorage) QueryInPartitionWithEtag(tableName, partitionKey, rowKey string) (interface{}, string, int, error) {
	defer a.recordProcessingTime(time.Now(), "QueryInPartitionWithEtag", tableName, partitionKey, rowKey)

	return a.getRowAndEtag(tableName, partitionKey, rowKey)
}

// QueryInPartitionWithEtagProperties - Query a row in the given partition of the table.
// Returns etag along with all of the properties
func (a *AzureStorage) QueryInPartitionWithEtagProperties(tableName, partitionKey, rowKey string) (map[string]interface{}, string, int, error) {
	defer a.recordProcessingTime(time.Now(), "QueryInPartitionWithEtagProperties", tableName, partitionKey, rowKey)

	return a.getRowEtagProperties(tableName, partitionKey, rowKey)
}

// QueryAllRows - Query a row in a given table.
func (a *AzureStorage) QueryAllRows(tableName string, context interface{}) (
	map[string]interface{}, bool, interface{}, int, error) {
	defer a.recordProcessingTime(time.Now(), "QueryAllRows", tableName, a.partitionKey, "NA")

	return a.getRows(tableName, a.partitionKey, context)
}

// QueryAllRowsInPartition - Query all rows in the specific partition of the given table.
func (a *AzureStorage) QueryAllRowsInPartition(tableName, partitionKey string, context interface{}) (
	map[string]interface{}, bool, interface{}, int, error) {
	defer a.recordProcessingTime(time.Now(), "QueryAllRowsInPartition", tableName, partitionKey, "NA")

	return a.getRows(tableName, partitionKey, context)
}

// QueryFilter - Query rows using a filter
// todo: need to determine a max size for filter, since its sent over a GET call to the storage API and fails if it goes over
func (a *AzureStorage) QueryFilter(tableName, filter string) (map[string]interface{}, error) {
	defer a.recordProcessingTime(time.Now(), "QueryFilter", tableName, "NA", "NA")

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

	results := make(map[string]interface{})
	for _, entity := range entities {
		switch len(entity.Properties) {
		case 0:
			return nil, fmt.Errorf("entity %v, table %v missing properties", entity.RowKey, tableName)
		case 1:
			for _, val := range entity.Properties {
				results[entity.RowKey] = val
			}
		default:
			results[entity.RowKey] = entity.Properties
		}
	}

	return results, nil
}

// QueryRowKeys - Fetch rows for keys provided
func (a *AzureStorage) QueryRowKeys(tableName string, keys []string) (map[string]interface{}, error) {
	defer a.recordProcessingTime(time.Now(), "QueryRowKeys", tableName, a.partitionKey, "NA")

	filter := BuildFilterOnPartition(a.partitionKey, "RowKey", keys)

	return a.QueryFilter(tableName, filter)
}

// KeyExist - check if key exist in a given table.
func (a *AzureStorage) KeyExist(tableName, rowKey string) (bool, error) {
	defer a.recordProcessingTime(time.Now(), "KeyExist", tableName, a.partitionKey, rowKey)
	property, status, err := a.getRow(tableName, a.partitionKey, rowKey)
	if property == nil {
		if status == http.StatusNotFound {
			return false, nil
		}

		return false, err
	}
	return true, nil
}

func (a *AzureStorage) insertOrUpdateRow(tableName, rowKey, propertyName string, value interface{}) error {
	tableClient, ok := a.tableNameToTableClientMap[tableName]
	if !ok {
		return fmt.Errorf("Cannot find valid table client for table %s", tableName)
	}
	table := tableClient.GetTableReference(tableName)
	entity := table.GetEntityReference(a.partitionKey, rowKey)
	properties := make(map[string]interface{})
	properties[propertyName] = value
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

func (a *AzureStorage) getRowEtagProperties(tableName string, partitionKey string, rowKey string) (map[string]interface{}, string, int, error) {
	tableClient, ok := a.tableNameToTableClientMap[tableName]
	if !ok {
		return nil, "", http.StatusInternalServerError, fmt.Errorf("Cannot find valid table client for table %s", tableName)
	}
	table := tableClient.GetTableReference(tableName)
	entity := table.GetEntityReference(partitionKey, rowKey)
	err := entity.Get(defaultTimeout, storage.FullMetadata, nil)
	if err == nil {
		return entity.Properties, entity.OdataEtag, http.StatusOK, nil
	}

	storageErr, ok := err.(storage.AzureStorageServiceError)
	if ok {
		return nil, "", storageErr.StatusCode, err
	}

	return nil, "", http.StatusInternalServerError, err
}

func (a *AzureStorage) getRowAndEtag(tableName string, partitionKey string, rowKey string) (interface{}, string, int, error) {
	properties, etag, status, err := a.getRowEtagProperties(tableName, partitionKey, rowKey)
	if err == nil {
		switch len(properties) {
		case 0:
			return nil, "", http.StatusInternalServerError, fmt.Errorf("entity %v in partition %v, table %v missing properties", rowKey, partitionKey, tableName)
		case 1: // dnc pattern for most tables is one property per row
			for _, val := range properties {
				return val, etag, status, nil
			}
		default: // tables with possibly more than one property. property size may vary per row
			return properties, etag, status, nil
		}
	}

	return nil, "", status, err
}

func (a *AzureStorage) getRow(tableName string, partitionKey string, rowKey string) (interface{}, int, error) {
	property, _, statusCode, err := a.getRowAndEtag(tableName, partitionKey, rowKey)
	return property, statusCode, err
}

func (a *AzureStorage) getRows(tableName string, partitionKey string, context interface{}) (map[string]interface{}, bool, interface{}, int, error) {
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

	result := make(map[string]interface{})
	for _, entity := range entitiesResult.Entities {
		switch len(entity.Properties) {
		case 0:
			return nil, false, nil, http.StatusInternalServerError, fmt.Errorf("entity %v in partition %v, table %v missing properties", entity.RowKey, partitionKey, tableName)
		case 1: // dnc pattern for most tables is one property per row
			for _, val := range entity.Properties {
				result[entity.RowKey] = val
			}
		default: // tables with possibly more than one property. property size may vary per row
			result[entity.RowKey] = entity.Properties
		}
	}
	var hasMoreRows = false
	var returnContext interface{}

	if entitiesResult.NextLink != nil {
		hasMoreRows = true
		returnContext = entitiesResult
	}
	return result, hasMoreRows, returnContext, http.StatusOK, nil
}
