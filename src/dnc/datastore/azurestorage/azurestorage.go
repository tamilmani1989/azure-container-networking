package azurestorage

import (
	"context"
	"crypto/sha256"
	"dnc/activedirectory"
	"dnc/aiwrapper"
	"dnc/configuration"
	"dnc/logger/loggershim"
	"dnc/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-container-networking/aitelemetry"
	"github.com/Azure/azure-sdk-for-go/services/cosmos-db/mgmt/2021-01-15/documentdb"
	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
)

const (
	// default timeout for calls
	defaultTimeout = 30
	// default to max batch size allowed by azure storage
	defaultBatchSize = 100
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

	if maxBatchSize == 0 {
		logger.Debugf("storage max batch size cannot be 0, defaulting to %d", defaultBatchSize)
		maxBatchSize = defaultBatchSize
	}

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
	if err := azureStorage.ensureSchema(); err != nil {
		return nil, fmt.Errorf("Could not ensure table schema: %v", err)
	}

	return &azureStorage, nil
}

// Init - Initialize azure storage with shared StorageClient passed by the caller.
// This is currently used by CleanupService which will initialize it per cluster partition to drive the cleanup
func NewAzureStorageEx(
	partitionKey string,
	isUsingCosmosDB bool,
	storageClient storage.Client,
	logger loggershim.LoggerInterface,
	tableToColumnMap map[string]string,
	latencyThresholdMs int,
	maxBatchSize int) (*AzureStorage, error) {
	logger.Debugf("[Azure Storage] NewAzureStorageEx PartitionKey %s, isUsingCosmosDB %s", partitionKey, isUsingCosmosDB)

	tblCl := storageClient.GetTableService()

	azureStorage := AzureStorage{
		partitionKey:       partitionKey,
		maxBatchSize:       maxBatchSize,
		latencyThresholdMs: latencyThresholdMs,
		logger:             logger,
		isCosmosDb:         isUsingCosmosDB,
	}

	logger.Debugf("[Azure Storage] Max transaction batch size set to %d", maxBatchSize)
	azureStorage.initTableNameToTableClientMap(tblCl, tableToColumnMap)

	return &azureStorage, nil
}

type dumbToken struct {
	token string
}

func (d *dumbToken) OAuthToken() string { return d.token }

func (a *AzureStorage) EnsureAutoScaledTables(ctx context.Context, resourceID, armtoken, armendpoint string, maxthroughput int32) error {
	if err := a.ensureSchema(); err != nil {
		return fmt.Errorf("Could not ensure table schema: %v", err)
	}
	//any more generic helper I can use?
	comosdb, err := activedirectory.ParseResource(resourceID)
	if err != nil {
		return err
	}
	tblclient := documentdb.NewTableResourcesClientWithBaseURI(armendpoint, comosdb.SubscriptionID)
	//https://github.com/Azure/azure-sdk-for-go/tree/v51.3.0#authentication
	tblclient.Authorizer = autorest.NewBearerAuthorizer(&dumbToken{armtoken})

	for tableName := range a.tableNameToTableClientMap {
		tblthroughput, err := tblclient.GetTableThroughput(ctx, comosdb.ResourceGroupName, comosdb.ResourceName, tableName)
		if err != nil {
			return err
		}
		throughputjson, err := json.Marshal(tblthroughput.Resource)
		if err != nil {
			return err
		}
		a.logger.Debugf("Checking %s: %s", tableName, string(throughputjson))
		if tblthroughput.Resource == nil || tblthroughput.Resource.AutoscaleSettings == nil {
			a.logger.Debugf("migrating %s", tableName)
			future, err := tblclient.MigrateTableToAutoscale(ctx, comosdb.ResourceGroupName, comosdb.ResourceName, tableName)
			if err != nil {
				return err
			}
			if err = future.FutureAPI.WaitForCompletionRef(ctx, tblclient.Client); err != nil {
				return err
			}
			if tblthroughput, err = future.Result(tblclient); err != nil {
				return err
			}
		}

		if tblthroughput.Resource == nil || tblthroughput.Resource.AutoscaleSettings == nil {
			return fmt.Errorf("Don't have autoscale settings after migration of %s", tableName)
		}

		if tblthroughput.Resource.AutoscaleSettings.MaxThroughput == nil || *tblthroughput.Resource.AutoscaleSettings.MaxThroughput < maxthroughput {
			a.logger.Debugf("updating max througput %s", tableName)

			throughputupdate := documentdb.ThroughputSettingsUpdateParameters{
				ThroughputSettingsUpdateProperties: &documentdb.ThroughputSettingsUpdateProperties{
					Resource: &documentdb.ThroughputSettingsResource{
						AutoscaleSettings: &documentdb.AutoscaleSettingsResource{
							MaxThroughput: &maxthroughput,
						},
					},
				},
			}
			future, err := tblclient.UpdateTableThroughput(ctx, comosdb.ResourceGroupName, comosdb.ResourceName, tableName, throughputupdate)
			if err != nil {
				return err
			}
			if err = future.FutureAPI.WaitForCompletionRef(ctx, tblclient.Client); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *AzureStorage) ensureSchema() error {
	for tableName, tblSvc := range a.tableNameToTableClientMap {
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

// InsertEntity inserts the entity into the specified table.
// If the entity contains no partition key, the partition configured in the storage client will be used.
// If an entity with the same row key exists in the partition, errors with conflict.
func (a *AzureStorage) InsertEntity(tableName string, entity storage.Entity) error {
	defer a.recordProcessingTime(time.Now(), "InsertEntity", tableName, entity.PartitionKey, entity.RowKey)
	return a.entityOp(tableName, entity, storage.InsertOp, false)
}

// InsertOrUpdateEntity inserts or updates the entity into the specified table.
// If the entity contains no partition key, the partition configured in the storage client will be used.
// If an entity with the same row key exists in the partition, it will be overwritten by this entity.
// Any Etags in the entity will not be used.
func (a *AzureStorage) InsertOrUpdateEntity(tableName string, entity storage.Entity) error {
	defer a.recordProcessingTime(time.Now(), "InsertOrUpdateEntity", tableName, entity.PartitionKey, entity.RowKey)
	return a.entityOp(tableName, entity, storage.InsertOp, true)
}

// UpdateEntity updates the entity into the specified table.
// If the entity contains no partition key, the partition configured in the storage client will be used.
// If an entity with the row key does not already exist in the partition, errors with not found.
// If the Etag does not match the existing entity's Etag, errors with precondition failed.
func (a *AzureStorage) UpdateEntity(tableName string, entity storage.Entity) error {
	defer a.recordProcessingTime(time.Now(), "UpdateEntity", tableName, entity.PartitionKey, entity.RowKey)
	return a.entityOp(tableName, entity, storage.ReplaceOp, false)
}

func (a *AzureStorage) entityOp(tableName string, entity storage.Entity, op storage.Operation, force bool) error {
	tableClient, ok := a.tableNameToTableClientMap[tableName]
	if !ok {
		return fmt.Errorf("cannot find valid table client for table %s", tableName)
	}

	table := tableClient.GetTableReference(tableName)
	entity.Table = table
	if entity.PartitionKey == "" {
		entity.PartitionKey = a.partitionKey
	}

	// todo: add other operations like merge as needed
	switch op {
	case storage.InsertOp:
		if force {
			return entity.InsertOrReplace(nil)
		}
		// note: storage.FullMetadata is only useful in mutation operations if the etag is to be returned right away.
		return entity.Insert(storage.NoMetadata, nil)
	case storage.ReplaceOp:
		return entity.Update(force, nil)
	case storage.DeleteOp:
		return entity.Delete(force, nil)
	}

	return fmt.Errorf("unsupported operation %v", op)
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

	batches := sliceMaps(batchItems, a.maxBatchSize)
	if len(batches) > 1 {
		msgFmt := "table %s batch upsert: %d entities exceed max batch limit of %d, attempting to persist in multiple batches"
		a.logger.Debugf(msgFmt, tableName, len(batchItems), a.maxBatchSize)
	}

	table := tableClient.GetTableReference(tableName)
	for _, b := range batches {
		var entities []storage.Entity
		for rowKey, rowProperties := range b {
			entity := table.GetEntityReference(a.partitionKey, rowKey)
			entity.Properties = rowProperties
			entities = append(entities, *entity)
		}

		if err := a.singleBatch(table, entities, storage.InsertOrReplaceOp); err != nil {
			return err
		}
	}

	return nil
}

// Deprecated: Use InsertEntityBatch.
// InsertBatch inserts a batch of rows for a given table and partition key
// The key in the batch represents the row key
// The value in the batch (map[string]interface{}) represents row properties and their respective values
// Documented limit is 100, but in practice it is over 1000
func (a *AzureStorage) InsertBatch(tableName, partitionKey string, batchItems map[string]map[string]interface{}) error {
	tableClient, ok := a.tableNameToTableClientMap[tableName]
	if !ok {
		return fmt.Errorf("cannot find valid table client for table %s", tableName)
	}

	batches := sliceMaps(batchItems, a.maxBatchSize)
	if len(batches) > 1 {
		msgFmt := "table %s batch insert: %d entities exceed max batch limit of %d, attempting to persist in multiple batches"
		a.logger.Debugf(msgFmt, tableName, len(batchItems), a.maxBatchSize)
	}

	table := tableClient.GetTableReference(tableName)
	for _, b := range batches {
		var entities []storage.Entity
		for rowKey, rowProperties := range b {
			entity := table.GetEntityReference(partitionKey, rowKey)
			entity.Properties = rowProperties
			entities = append(entities, *entity)
		}

		if err := a.singleBatch(table, entities, storage.InsertOp); err != nil {
			return err
		}
	}

	return nil
}

// DeleteBatch deletes a batch of rows in a given table based on the given partition and row keys
// There is a limit of 100 items per batch
func (a *AzureStorage) DeleteBatch(tableName, partitionKey string, rowKeys []string) error {
	tableClient, ok := a.tableNameToTableClientMap[tableName]
	if !ok {
		return fmt.Errorf("cannot find valid table client for table %s", tableName)
	}

	batches := sliceStrings(rowKeys, a.maxBatchSize)
	if len(batches) > 1 {
		msgFmt := "table %s batch delete: %d keys exceed max batch limit of %d, attempting to delete in multiple batches"
		a.logger.Debugf(msgFmt, tableName, len(rowKeys), a.maxBatchSize)
	}

	table := tableClient.GetTableReference(tableName)

	for _, b := range batches {
		batch := table.NewBatch()
		for _, rowKey := range b {
			entity := table.GetEntityReference(partitionKey, rowKey)
			batch.DeleteEntityByForce(entity, true)
		}

		if err := batch.ExecuteBatch(); err != nil {
			return err
		}
	}
	return nil
}

// Deletes all rows for associated partition (a.partitionKey) from a given table.
func (a *AzureStorage) WipePartition(tableName string) error {
	tableClient, ok := a.tableNameToTableClientMap[tableName]
	if !ok {
		return fmt.Errorf("cannot find valid table client for table %s", tableName)
	}
	table := tableClient.GetTableReference(tableName)

	filter := BuildFilterOnPartition(a.partitionKey)
	entities, err := a.queryFilter(tableName, storage.NoMetadata, filter)
	if err != nil {
		storageErr, ok := err.(storage.AzureStorageServiceError)
		if ok && (storageErr.StatusCode == http.StatusNotFound || storageErr.StatusCode == http.StatusGone)  {
			return nil
		}

		return err
	}

	if len(entities) == 0 {
		return nil
	}

	var entitiesBatch [][]*storage.Entity
	for len(entities) > a.maxBatchSize {
		entities, entitiesBatch = entities[a.maxBatchSize:], append(entitiesBatch, entities[0:a.maxBatchSize])
	}
	entitiesBatch = append(entitiesBatch, entities)

	for i := 0; i < len(entitiesBatch); i++ {
		batch := table.NewBatch()
		for _, storageEntity := range entitiesBatch[i] {
			entity := table.GetEntityReference(a.partitionKey, storageEntity.RowKey)
			batch.DeleteEntityByForce(entity, true)
		}

		err = batch.ExecuteBatch()
		if err != nil {
			return err
		}
	}

	return nil
}

// todo: added for deprecated paths, but should eventually remove
func sliceMaps(batchMap map[string]map[string]interface{}, batchSize int) []map[string]map[string]interface{} {
	var batches []map[string]map[string]interface{}

	batch := make(map[string]map[string]interface{})
	for k, v := range batchMap {
		batch[k] = v
		if len(batch) == batchSize {
			batches = append(batches, batch)
			batch = make(map[string]map[string]interface{})
		}
	}

	if len(batch) > 0 {
		batches = append(batches, batch)
	}

	return batches
}

// todo: added for deprecated path, but should eventually remove
func sliceStrings(keys []string, batchSize int) [][]string {
	var batches [][]string
	for len(keys) > batchSize {
		keys, batches = keys[batchSize:], append(batches, keys[0:batchSize])
	}
	return append(batches, keys)
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
	if len(entities) == 0 {
		a.logger.Debugf("table %s batch op %v: no entities to persist", tableName, op)
		return nil
	}

	batches := sliceEntities(entities, a.maxBatchSize)

	if len(batches) > 1 {
		msgFmt := "table %s batch op %v: %d entities exceed max batch limit of %d, attempting to persist in multiple batches"
		a.logger.Debugf(msgFmt, tableName, op, len(entities), a.maxBatchSize)
	}

	tableClient, ok := a.tableNameToTableClientMap[tableName]
	if !ok {
		return fmt.Errorf("cannot find valid table client for table %s", tableName)
	}

	table := tableClient.GetTableReference(tableName)

	for i := 0; i < len(batches); i++ {
		if err := a.singleBatch(table, batches[i], op); err != nil {
			return err
		}
	}

	return nil
}

func sliceEntities(entities []storage.Entity, batchSize int) [][]storage.Entity {
	var batches [][]storage.Entity
	for len(entities) > batchSize {
		entities, batches = entities[batchSize:], append(batches, entities[0:batchSize])
	}
	return append(batches, entities)
}

func (a *AzureStorage) singleBatch(table *storage.Table, entities []storage.Entity, op storage.Operation) error {
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

		// entities need a more fleshed out table reference than assigned at higher levels in order to work in batch
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

// Checks if any row exists in an associated partition for a given table
func (a *AzureStorage) AnyRowExists(tableName string) (bool, error) {
	defer a.recordProcessingTime(time.Now(), "AnyRowExists", tableName, a.partitionKey, "NA")

	filter := BuildFilterOnPartition(a.partitionKey)
	entities, err := a.queryFilterTopN(tableName, storage.NoMetadata, filter, 1)
	if err != nil {
		storageErr, ok := err.(storage.AzureStorageServiceError)
		if ok && storageErr.StatusCode == http.StatusNotFound {
			return false, nil
		}

		return false, err
	}

	return len(entities) > 0, nil
}

// QueryFilter - Query rows using a filter.
func (a *AzureStorage) QueryFilter(tableName, filter string) ([]*storage.Entity, error) {
	defer a.recordProcessingTime(time.Now(), "QueryFilter", tableName, "NA", "NA")

	return a.queryFilter(tableName, storage.FullMetadata, filter)
}

// QueryFilter - Query rows using a filter format and arguments.
func (a *AzureStorage) QueryFilterf(tableName, filterFmt string, args ...interface{}) ([]*storage.Entity, error) {
	defer a.recordProcessingTime(time.Now(), "QueryFilterf", tableName, "NA", "NA")

	return a.queryFilter(tableName, storage.FullMetadata, fmt.Sprintf(filterFmt, args...))
}

// todo: need to determine a max size for filter, since its sent over a GET call to the storage API and fails if it goes over
func (a *AzureStorage) queryFilter(tableName string, ml storage.MetadataLevel, filter string) ([]*storage.Entity, error) {
	tableClient, ok := a.tableNameToTableClientMap[tableName]
	if !ok {
		return nil, fmt.Errorf("cannot find valid table client for table %s", tableName)
	}

	table := tableClient.GetTableReference(tableName)

	queryOptions := storage.QueryOptions{Filter: filter}

	queryResult, err := table.QueryEntities(defaultTimeout, ml, &queryOptions)
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

	return a.queryFilterTopN(tableName, storage.FullMetadata, filter, count)
}

func (a *AzureStorage) queryFilterTopN(tableName string, ml storage.MetadataLevel, filter string, count int) ([]*storage.Entity, error) {
	tableClient, ok := a.tableNameToTableClientMap[tableName]
	if !ok {
		return nil, fmt.Errorf("cannot find valid table client for table %s", tableName)
	}

	table := tableClient.GetTableReference(tableName)

	// todo: Why not using QueryOption Top?
	queryOptions := storage.QueryOptions{Filter: filter}

	result, err := table.QueryEntities(defaultTimeout, ml, &queryOptions)
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
