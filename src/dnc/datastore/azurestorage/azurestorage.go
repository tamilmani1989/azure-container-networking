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

// Max time latency allowed for data store operations before recording the operation latency in the log.
var thresholdForLoggingDataStoreOpLatencyMs int

const (
	// maximum batch transaction size
	// see https://docs.microsoft.com/en-us/rest/api/storageservices/performing-entity-group-transactions#requirements-for-entity-group-transactions
	MaxBatchSize = 100
)

// AzureStorage = Storage object to manage tables in a partition
type AzureStorage struct {
	configuration.AzureStorageInfo
	tableClient               storage.TableServiceClient
	regionalTableClient       storage.TableServiceClient
	tableNameToTableClientMap map[string]*storage.TableServiceClient
}

// Init - Initialize azure storage.
func (azureStorage *AzureStorage) Init(
	tableToColumnMap map[string]string,
	thresholdForLoggingDataStoreOpLatencyMsArg int) {
	loggershim.LS.Debugf("Environment type for azure store settings %s", azureStorage.AzureStorageInfo.Environment)
	storageClient, err := utils.GetStorageClient(
		azureStorage.AzureStorageInfo.AccountName,
		azureStorage.AzureStorageInfo.Key,
		azureStorage.AzureStorageInfo.Environment,
		azureStorage.AzureStorageInfo.IsUsingCosmosDb)

	utils.OnErrorFail(err, "Create client failed")
	azureStorage.tableClient = storageClient.GetTableService()
	azureStorage.initTableNameToTableClientMap(tableToColumnMap)
	thresholdForLoggingDataStoreOpLatencyMs = thresholdForLoggingDataStoreOpLatencyMsArg
	if len(strings.TrimSpace(azureStorage.AzureStorageInfo.RegionalStorageConnectionString)) > 0 {
		azureStorage.initRegionalStorage()
	}

	loggershim.LS.Debugf("[Azure DNC] Ensuring table schema")
	err = azureStorage.ensureSchema(tableToColumnMap)
	utils.OnErrorFail(err, "Could not ensure table schema")
}

func (azureStorage *AzureStorage) ensureSchema(tables map[string]string) error {
	for tableName := range tables {
		tbl := azureStorage.tableClient.GetTableReference(tableName)
		if err := tbl.Create(30, storage.EmptyPayload, nil); err != nil {
			switch e := err.(type) {
			case storage.AzureStorageServiceError:
				if e.StatusCode == http.StatusConflict {
					loggershim.LS.Debugf("[Azure DNC] Table %s already exists", tableName)
					continue
				}
			default:
				return fmt.Errorf("Could not create table %s: %v", tableName, err)
			}
		}
		loggershim.LS.Debugf("[Azure DNC] Table %s created successfully", tableName)
	}
	return nil
}

// initRegionalStorage - Initialize the table client needed to access the regional storage
func (azureStorage *AzureStorage) initRegionalStorage() {
	loggershim.LS.Debugf("[Azure DNC] InitRegionalStorage with %x", sha256.Sum256([]byte(azureStorage.AzureStorageInfo.RegionalStorageConnectionString)))
	regionalStorageClient, err := storage.NewClientFromConnectionString(
		azureStorage.AzureStorageInfo.RegionalStorageConnectionString)
	utils.OnErrorFail(err, "Create regional client failed")

	azureStorage.regionalTableClient = regionalStorageClient.GetTableService()
	azureStorage.updateTableNameToTableClientMap("IPAllocationsTable", &azureStorage.regionalTableClient)
}

// updateTableNameToTableClientMap - Update tableClient to use for the given table
func (azureStorage *AzureStorage) updateTableNameToTableClientMap(tableName string, tableClient *storage.TableServiceClient) {
	azureStorage.tableNameToTableClientMap[tableName] = tableClient
	loggershim.LS.Debugf("[Azure DNC] Updated settings for table: %s to use regional storage", tableName)
}

// initTableNameToTableClientMap - Set tableClient as default client to all the tables
func (azureStorage *AzureStorage) initTableNameToTableClientMap(tableToColumnMap map[string]string) {
	azureStorage.tableNameToTableClientMap = make(map[string]*storage.TableServiceClient)
	for tableName := range tableToColumnMap {
		azureStorage.tableNameToTableClientMap[tableName] = &azureStorage.tableClient
	}
}

// TODO: Move this processing time recording functionality in datastoreshim so that it runs irrespective
// of the underlying storage. The current code is not set up to do that because the datastoreshim just sets
// the function pointers to the selected underlying storage. datastoreshim implementation needs to be changed
// to call recordProcessingTime first and then call the underlying storage function.
func recordProcessingTime(
	start time.Time,
	operation, tableName, partitionKey, rowKey string) {
	timeElapsedMs := time.Since(start).Milliseconds()
	if timeElapsedMs > int64(thresholdForLoggingDataStoreOpLatencyMs) {
		loggershim.LS.Debugf("[AzureStorage] Operation: %s, Table: %s, PartitionKey: %s,"+
			" rowKey: %s, TimeMs: %v", operation, tableName, partitionKey, rowKey, timeElapsedMs)
	}

	sendDataStoreMetric(aiwrapper.DataStoreOperationTimeMetric, float64(timeElapsedMs),
		operation, tableName)
}

func sendDataStoreMetric(
	metricName string,
	metricValue float64,
	operation, tableName string) {
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
func (azureStorage *AzureStorage) Delete(tableName, rowKey string) error {
	defer recordProcessingTime(time.Now(), "Delete", tableName, azureStorage.PartitionKey, rowKey)
	defer utils.PanicRecover()

	tableClient, ok := azureStorage.tableNameToTableClientMap[tableName]
	if !ok {
		return fmt.Errorf("Cannot find valid table client for table %s", tableName)
	}
	table := *tableClient.GetTableReference(tableName)
	entity := table.GetEntityReference(azureStorage.PartitionKey, rowKey)

	return entity.Delete(true, nil)
}

// DeleteInPartition - Deletes a row in the partition of the given table.
func (azureStorage *AzureStorage) DeleteInPartition(tableName, partitionKey, rowKey string) error {
	defer recordProcessingTime(time.Now(), "DeleteInPartition", tableName, partitionKey, rowKey)
	defer utils.PanicRecover()

	tableClient, ok := azureStorage.tableNameToTableClientMap[tableName]
	if !ok {
		return fmt.Errorf("Cannot find valid table client for table %s", tableName)
	}
	table := *tableClient.GetTableReference(tableName)
	entity := table.GetEntityReference(partitionKey, rowKey)
	return entity.Delete(true, nil)
}

// InsertOrUpdate - InsertOrUpdate a row in a given table.
func (azureStorage *AzureStorage) InsertOrUpdate(tableName, rowKey, propertyName string, value interface{}) error {
	defer recordProcessingTime(time.Now(), "InsertOrUpdate", tableName, azureStorage.PartitionKey, rowKey)

	return azureStorage.insertOrUpdateRow(tableName, rowKey, propertyName, value)
}

// UpdateInPartition - Update (overwrite) a row in a given table if it exists.
func (azureStorage *AzureStorage) UpdateInPartition(
	tableName, partitionKay, rowKey string, properties map[string]interface{}) error {
	defer recordProcessingTime(time.Now(), "UpdateInPartition", tableName, azureStorage.PartitionKey, rowKey)

	return azureStorage.updateRowWithEtag(tableName, partitionKay, rowKey, "", properties)
}

// UpdateInPartitionWithEtag - Update (overwrite) a row in a given table if it exists.
// Match with specified etag for update.
func (azureStorage *AzureStorage) UpdateInPartitionWithEtag(
	tableName, partitionKay, rowKey, etag string, properties map[string]interface{}) error {
	defer recordProcessingTime(time.Now(), "UpdateInPartitionWithEtag", tableName, azureStorage.PartitionKey, rowKey)

	return azureStorage.updateRowWithEtag(tableName, partitionKay, rowKey, etag, properties)
}

// InsertInPartition - Insert a row in the specific partition of the given table with one property
func (azureStorage *AzureStorage) InsertInPartition(tableName, partitionKay, rowKey, propertyName string, value interface{}) (
	int, error) {
	defer recordProcessingTime(time.Now(), "InsertInPartition", tableName, partitionKay, rowKey)

	properties := make(map[string]interface{})
	properties[propertyName] = value
	return azureStorage.insertRowInPartition(tableName, partitionKay, rowKey, properties)
}

// InsertInPartitionProperties - Insert a row in the specific partition of the given table
// with multiple properties
func (azureStorage *AzureStorage) InsertInPartitionProperties(
	tableName, partitionKay, rowKey string, properties map[string]interface{}) (int, error) {
	defer recordProcessingTime(time.Now(), "InsertInPartitionProperties", tableName, partitionKay, rowKey)

	return azureStorage.insertRowInPartition(tableName, partitionKay, rowKey, properties)
}

// UpsertBatch inserts or replaces a batch of rows (row key to properties) for a given table and partition key
// There is a limit of 100 items per batch
func (azureStorage *AzureStorage) UpsertBatch(tableName, partitionKey string, batchItems map[string]map[string]interface{}) error {
	tableClient, ok := azureStorage.tableNameToTableClientMap[tableName]
	if !ok {
		return fmt.Errorf("cannot find valid table client for table %s", tableName)
	}

	table := tableClient.GetTableReference(tableName)

	batch := table.NewBatch()
	for rowKey, rowProperties := range batchItems {
		entity := table.GetEntityReference(partitionKey, rowKey)
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
// There is a limit of 100 items per batch
func (azureStorage *AzureStorage) InsertBatch(tableName, partitionKey string, batchItems map[string]map[string]interface{}) error {
	if len(batchItems) > MaxBatchSize {
		return fmt.Errorf("batch size %d exceeds limit of %d", len(batchItems), MaxBatchSize)
	}

	tableClient, ok := azureStorage.tableNameToTableClientMap[tableName]
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
func (azureStorage *AzureStorage) DeleteBatch(tableName, partitionKey string, rowKeys []string) error {
	if len(rowKeys) > MaxBatchSize {
		return fmt.Errorf("batch size should not exceed %d", MaxBatchSize)
	}

	tableClient, ok := azureStorage.tableNameToTableClientMap[tableName]
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
func (azureStorage *AzureStorage) Query(tableName, rowKey string) (interface{}, int, error) {
	defer recordProcessingTime(time.Now(), "Query", tableName, azureStorage.AzureStorageInfo.PartitionKey, rowKey)

	return azureStorage.getRow(tableName, azureStorage.AzureStorageInfo.PartitionKey, rowKey)
}

// QueryInPartition - Query a row in the given partition of the table.
func (azureStorage *AzureStorage) QueryInPartition(tableName, partitionKey, rowKey string) (interface{}, int, error) {
	defer recordProcessingTime(time.Now(), "QueryInPartition", tableName, partitionKey, rowKey)

	return azureStorage.getRow(tableName, partitionKey, rowKey)
}

// QueryInPartitionWithEtag - Query a row in the given partition of the table. Returns etag along with first property
func (azureStorage *AzureStorage) QueryInPartitionWithEtag(tableName, partitionKey, rowKey string) (interface{}, string, int, error) {
	defer recordProcessingTime(time.Now(), "QueryInPartitionWithEtag", tableName, partitionKey, rowKey)

	return azureStorage.getRowAndEtag(tableName, partitionKey, rowKey)
}

// QueryInPartitionWithEtagProperties - Query a row in the given partition of the table.
// Returns etag along with all of the properties
func (azureStorage *AzureStorage) QueryInPartitionWithEtagProperties(
	tableName, partitionKey, rowKey string) (map[string]interface{}, string, int, error) {
	defer recordProcessingTime(time.Now(), "QueryInPartitionWithEtagProperties", tableName, partitionKey, rowKey)

	return azureStorage.getRowEtagProperties(tableName, partitionKey, rowKey)
}

// QueryAllRows - Query a row in a given table.
func (azureStorage *AzureStorage) QueryAllRows(tableName string, context interface{}) (
	map[string]interface{}, bool, interface{}, int, error) {
	defer recordProcessingTime(time.Now(), "QueryAllRows", tableName, azureStorage.AzureStorageInfo.PartitionKey, "NA")

	return azureStorage.getRows(tableName, azureStorage.AzureStorageInfo.PartitionKey, context)
}

// QueryAllRowsInPartition - Query all rows in the specific partition of the given table.
func (azureStorage *AzureStorage) QueryAllRowsInPartition(tableName, partitionKey string, context interface{}) (
	map[string]interface{}, bool, interface{}, int, error) {
	defer recordProcessingTime(time.Now(), "QueryAllRowsInPartition", tableName, partitionKey, "NA")

	return azureStorage.getRows(tableName, partitionKey, context)
}

// KeyExist - check if key exist in a given table.
func (azureStorage *AzureStorage) KeyExist(tableName, rowKey string) (bool, error) {
	defer recordProcessingTime(time.Now(), "KeyExist", tableName, azureStorage.AzureStorageInfo.PartitionKey, rowKey)
	property, status, err := azureStorage.getRow(tableName, azureStorage.AzureStorageInfo.PartitionKey, rowKey)
	if property == nil {
		if status == http.StatusNotFound {
			return false, nil
		}

		return false, err
	}
	return true, nil
}

func (azureStorage *AzureStorage) insertOrUpdateRow(tableName, rowKey, propertyName string, value interface{}) error {
	defer utils.PanicRecover()
	tableClient, ok := azureStorage.tableNameToTableClientMap[tableName]
	if !ok {
		return fmt.Errorf("Cannot find valid table client for table %s", tableName)
	}
	table := tableClient.GetTableReference(tableName)
	entity := table.GetEntityReference(azureStorage.PartitionKey, rowKey)
	properties := make(map[string]interface{})
	properties[propertyName] = value
	entity.Properties = properties
	err := entity.InsertOrReplace(nil)
	return err
}

func (azureStorage *AzureStorage) updateRowWithEtag(
	tableName, partitionKay, rowKey, etag string, properties map[string]interface{}) error {
	defer utils.PanicRecover()
	tableClient, ok := azureStorage.tableNameToTableClientMap[tableName]
	if !ok {
		return fmt.Errorf("Cannot find valid table client for table %s", tableName)
	}
	table := *tableClient.GetTableReference(tableName)
	entity := table.GetEntityReference(partitionKay, rowKey)
	err := entity.Get(30, storage.FullMetadata, nil)
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

func (azureStorage *AzureStorage) insertRowInPartition(
	tableName, partitionKay, rowKey string, properties map[string]interface{}) (int, error) {
	defer utils.PanicRecover()
	tableClient, ok := azureStorage.tableNameToTableClientMap[tableName]
	if !ok {
		return http.StatusInternalServerError, fmt.Errorf("Cannot find valid table client for table %s", tableName)
	}
	table := tableClient.GetTableReference(tableName)
	entity := table.GetEntityReference(partitionKay, rowKey)
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

func (azureStorage *AzureStorage) getRowEtagProperties(
	tableName string, partitionKey string, rowKey string) (map[string]interface{}, string, int, error) {
	defer utils.PanicRecover()
	tableClient, ok := azureStorage.tableNameToTableClientMap[tableName]
	if !ok {
		return nil, "", http.StatusInternalServerError, fmt.Errorf("Cannot find valid table client for table %s", tableName)
	}
	table := *tableClient.GetTableReference(tableName)
	entity := table.GetEntityReference(partitionKey, rowKey)
	err := entity.Get(30, storage.FullMetadata, nil)
	if err == nil {
		return entity.Properties, entity.OdataEtag, http.StatusOK, nil
	}

	storagErr, ok := err.(storage.AzureStorageServiceError)
	if ok {
		return nil, "", storagErr.StatusCode, err
	}

	return nil, "", http.StatusInternalServerError, err
}

func (azureStorage *AzureStorage) getRowAndEtag(
	tableName string, partitionKey string, rowKey string) (interface{}, string, int, error) {
	properties, etag, status, err := azureStorage.getRowEtagProperties(tableName, partitionKey, rowKey)
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

func (azureStorage *AzureStorage) getRow(tableName string, partitionKey string, rowKey string) (interface{}, int, error) {
	property, _, statusCode, err := azureStorage.getRowAndEtag(tableName, partitionKey, rowKey)
	return property, statusCode, err
}

func (azureStorage *AzureStorage) getRows(tableName string, partitionKey string, context interface{}) (
	map[string]interface{}, bool, interface{}, int, error) {
	defer utils.PanicRecover()
	var (
		entitiesResult *storage.EntityQueryResult
		err            error
	)

	if context == nil {
		tableClient, ok := azureStorage.tableNameToTableClientMap[tableName]
		if !ok {
			return nil, false, nil, http.StatusInternalServerError, fmt.Errorf("Cannot find valid table client for table %s", tableName)
		}
		table := tableClient.GetTableReference(tableName)
		queryOptions := storage.QueryOptions{
			Filter: "PartitionKey eq '" + partitionKey + "'",
		}

		entitiesResult, err = table.QueryEntities(30, storage.FullMetadata, &queryOptions)
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
