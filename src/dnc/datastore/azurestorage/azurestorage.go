package azurestorage

import (
	"dnc/configuration"
	"dnc/logger/loggershim"
	"dnc/utils"
	"fmt"
	"net/http"
	"os"
	"strings"
	"crypto/sha256"

	"github.com/Azure/azure-sdk-for-go/storage"
)

// AzureStorage = Storage object to manage tables in a partition
type AzureStorage struct {
	configuration.AzureStorageInfo
	tableClient               storage.TableServiceClient
	regionalTableClient       storage.TableServiceClient
	tableNameToTableClientMap map[string]*storage.TableServiceClient
}

// Init - Initialize azure storage.
func (azureStorage *AzureStorage) Init(tableToColumnMap map[string]string) {
	storageClient, err := storage.NewBasicClient(
		azureStorage.AzureStorageInfo.AccountName,
		azureStorage.AzureStorageInfo.Key)

	utils.OnErrorFail(err, "Create client failed")
	azureStorage.tableClient = storageClient.GetTableService()
	azureStorage.initTableNameToTableClientMap(tableToColumnMap)
	if len(strings.TrimSpace(azureStorage.AzureStorageInfo.RegionalStorageConnectionString)) > 0 {
		azureStorage.initRegionalStorage()
	}
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

// Delete - Deletes a row in a given table.
func (azureStorage *AzureStorage) Delete(tableName, rowKey string) error {
	tableClient, ok := azureStorage.tableNameToTableClientMap[tableName]
	if !ok {
		return fmt.Errorf("Cannot find valid table client for table %s", tableName)
	}
	table := *tableClient.GetTableReference(tableName)
	entity := table.GetEntityReference(azureStorage.PartitionKey, rowKey)

	// The Delete function may panic due to nil resp (in resp.StatusCode),
	// so we will try to recover and proceed from there.
	defer func() {
		if r := recover(); r != nil {
			loggershim.LS.Errorf("[Azure DNC] Recovered from crash due to %+v\n", r)
		}
	}()

	return entity.Delete(true, nil)
}

// DeleteInPartition - Deletes a row in the partition of the given table.
func (azureStorage *AzureStorage) DeleteInPartition(tableName, partitionKey, rowKey string) error {
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
	return azureStorage.insertOrUpdateRow(tableName, rowKey, propertyName, value)
}

// InsertInPartition - Insert a row in the specific partition of the given table.
func (azureStorage *AzureStorage) InsertInPartition(tableName, partitionKay, rowKey, propertyName string, value interface{}) (
	int, error) {
	return azureStorage.insertRowInPartition(tableName, partitionKay, rowKey, propertyName, value)
}

// Query - Query a row in a given table.
func (azureStorage *AzureStorage) Query(tableName, rowKey string) (interface{}, int, error) {
	return azureStorage.getRow(tableName, azureStorage.AzureStorageInfo.PartitionKey, rowKey)
}

// QueryInPartition - Query a row in the given partition of the table.
func (azureStorage *AzureStorage) QueryInPartition(tableName, partitionKey, rowKey string) (interface{}, int, error) {
	return azureStorage.getRow(tableName, partitionKey, rowKey)
}

// QueryAllRows - Query a row in a given table.
func (azureStorage *AzureStorage) QueryAllRows(tableName string, context interface{}) (
	map[string]interface{}, bool, interface{}, int, error) {
	return azureStorage.getRows(tableName, azureStorage.AzureStorageInfo.PartitionKey, context)
}

// QueryAllRowsInPartition - Query all rows in the specific partition of the given table.
func (azureStorage *AzureStorage) QueryAllRowsInPartition(tableName, partitionKey string, context interface{}) (
	map[string]interface{}, bool, interface{}, int, error) {
	return azureStorage.getRows(tableName, partitionKey, context)
}

// KeyExist - check if key exist in a given table.
func (azureStorage *AzureStorage) KeyExist(tableName, rowKey string) (bool, error) {
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
	tableClient, ok := azureStorage.tableNameToTableClientMap[tableName]
	if !ok {
		return fmt.Errorf("Cannot find valid table client for table %s", tableName)
	}
	table := *tableClient.GetTableReference(tableName)
	table.Create(30, storage.EmptyPayload, nil)
	entity := table.GetEntityReference(azureStorage.PartitionKey, rowKey)
	properties := make(map[string]interface{})
	properties[propertyName] = value
	entity.Properties = properties
	err := entity.InsertOrReplace(nil)
	return err
}

func (azureStorage *AzureStorage) insertRowInPartition(tableName, partitionKay, rowKey, propertyName string, value interface{}) (
	int, error) {
	tableClient, ok := azureStorage.tableNameToTableClientMap[tableName]
	if !ok {
		return http.StatusInternalServerError, fmt.Errorf("Cannot find valid table client for table %s", tableName)
	}
	table := *tableClient.GetTableReference(tableName)
	table.Create(30, storage.EmptyPayload, nil)
	entity := table.GetEntityReference(partitionKay, rowKey)
	properties := make(map[string]interface{})
	properties[propertyName] = value
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

// func InsertOrUpdate1(
// 	updateEntities UpdateEntities) error {

// 	var partitionTableMap = make(map[string]TablePartitionBatchMapTuple)

// 	for _, updateEntity := range updateEntities {
// 		var tableBatch *storage.TableBatch
// 		var table *storage.Table
// 		var entity *storage.Entity
// 		var tablePartitionBatchMapTuple TablePartitionBatchMapTuple
// 		var tableExists bool
// 		var partitionExists bool
// 		if tablePartitionBatchMapTuple, tableExists = partitionTableMap[updateEntity.TableName]; !tableExists {
// 			table = tableClient.GetTableReference(updateEntity.TableName)
// 			table.Create(30, storage.EmptyPayload, nil)
// 			tablePartitionBatchMapTuple = TablePartitionBatchMapTuple{table, make(map[string]*storage.TableBatch)}
// 			partitionTableMap[updateEntity.TableName] = tablePartitionBatchMapTuple
// 		} else {
// 			table = tablePartitionBatchMapTuple.Table
// 		}

// 		if tableBatch, partitionExists = tablePartitionBatchMapTuple.PartitionBatchMap[updateEntity.PartitionKey]; !partitionExists {
// 			tableBatch = table.NewBatch()
// 			tablePartitionBatchMapTuple.PartitionBatchMap[updateEntity.PartitionKey] = tableBatch
// 		}

// 		entity = table.GetEntityReference(updateEntity.PartitionKey, updateEntity.RowKey)
// 		if updateEntity.Type == InsertOrReplaceEntity {
// 			entity.Properties = updateEntity.Properties
// 			tableBatch.InsertOrReplaceEntity(entity, true)
// 		} else {
// 			tableBatch.DeleteEntity(entity, true)
// 		}
// 	}

// 	for tableName, tablePartitionBatchMapTuple := range partitionTableMap {
// 		for partitionName, tableBatch := range tablePartitionBatchMapTuple.PartitionBatchMap {
// 			fmt.Println(tableBatch.BatchEntitySlice[0].PartitionKey, tableBatch.BatchEntitySlice[0].RowKey, tableBatch.BatchEntitySlice[0].Properties, tableBatch.BatchEntitySlice[0].Table.Name)
// 			err := tableBatch.ExecuteBatch()
// 			if err != nil {
// 				fmt.Println("Insert Error", err)
// 				return fmt.Errorf("InsertOrUpdate failed TableName:%s Partition Name %s Error%v", tableName, partitionName, err)
// 			}
// 		}
// 	}

// 	for _, updateEntity := range updateEntities {
// 		tablePartitionBatchMapTuple, _ := partitionTableMap[updateEntity.TableName]
// 		if updateEntity.Type == InsertOrReplaceEntity {
// 			entity := tablePartitionBatchMapTuple.Table.GetEntityReference(updateEntity.PartitionKey, updateEntity.RowKey)
// 			entity.Get(30, storage.FullMetadata, nil)
// 			fmt.Println("Retrieved Entity:", entity)
// 		}
// 	}
// 	return nil
// }

func (azureStorage *AzureStorage) getRow(tableName string, partitionKey string, rowKey string) (interface{}, int, error) {
	tableClient, ok := azureStorage.tableNameToTableClientMap[tableName]
	if !ok {
		return nil, http.StatusInternalServerError, fmt.Errorf("Cannot find valid table client for table %s", tableName)
	}
	table := *tableClient.GetTableReference(tableName)
	entity := table.GetEntityReference(partitionKey, rowKey)
	err := entity.Get(30, storage.FullMetadata, nil)
	if err == nil {
		if len(entity.Properties) != 1 {
			return nil, http.StatusInternalServerError, fmt.Errorf("Properties missing columns")
		}
		for _, val := range entity.Properties {
			return val, http.StatusOK, nil
		}
	}

	storagErr, ok := err.(storage.AzureStorageServiceError)
	if ok {
		return nil, storagErr.StatusCode, err
	}

	return nil, http.StatusInternalServerError, err
}

func (azureStorage *AzureStorage) getRows(tableName string, partitionKey string, context interface{}) (
	map[string]interface{}, bool, interface{}, int, error) {
	var entitiesResult *storage.EntityQueryResult

	var err error
	if context == nil {
		tableClient, ok := azureStorage.tableNameToTableClientMap[tableName]
		if !ok {
			return nil, false, nil, http.StatusInternalServerError, fmt.Errorf("Cannot find valid table client for table %s", tableName)
		}
		table := *tableClient.GetTableReference(tableName)
		queryOptions := storage.QueryOptions{
			Filter: "PartitionKey eq '" + partitionKey + "'",
		}

		entitiesResult, err = table.QueryEntities(30, storage.FullMetadata, &queryOptions)

		if err != nil {
			storagErr, ok := err.(storage.AzureStorageServiceError)
			if ok {
				return nil, false, nil, storagErr.StatusCode, err
			}
			return nil, false, nil, http.StatusInternalServerError, err
		}
	} else {
		entitiesResult = context.(*storage.EntityQueryResult)
		entitiesResult, err = entitiesResult.NextResults(nil)
	}

	var result = make(map[string]interface{})
	for _, entity := range entitiesResult.Entities {
		if len(entity.Properties) != 1 {
			return nil, false, nil, http.StatusInternalServerError, fmt.Errorf("Properties missing columns")
		}
		for _, val := range entity.Properties {
			result[entity.RowKey] = val
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

// type EntityWithContext struct {
// 	RowKey     string
// 	Properties map[string]interface{}
// 	Context    interface{}
// }

// type EntitiesWithContext []EntityWithContext

// getEnvVarOrExit returns the value of specified environment variable or terminates if it's not defined.
func getEnvVarOrExit(varName string) string {
	value := os.Getenv(varName)
	if value == "" {
		loggershim.LS.Errorf("Missing environment variable %s\n", varName)
		os.Exit(1)
	}

	return value
}
