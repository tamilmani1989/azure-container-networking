// -----------------------------------------------------------------------------
// <copyright file="CosmosDbDataStore.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Collections.Concurrent;
    using System.Collections.Generic;
    using System.Linq;
    using System.Net.Http;
    using System.Threading;
    using System.Threading.Tasks;
    using Microsoft.AspNetCore.Mvc.Filters;
    using Microsoft.Azure.Cosmos.Table;
    using Microsoft.Azure.KeyVault.WebKey;
    using Microsoft.Extensions.Logging;

    /// <summary>
    /// Initialize CosmosDB Table storage
    /// </summary>
    public class CosmosDbDataStore : ITableDataStore
    {
        private IConnectionStringProvider connectionStringProvider;
        private CloudStorageAccount storageAccount;
        private CloudTableClient tableClient;
        ILogger<CosmosDbDataStore> logger;
        private readonly SemaphoreSlim semaphoreSlim;

        ConcurrentDictionary<string, CloudTable> tableMap;

        /// <summary>
        /// Create an instance of CosmosDbDataStorage
        /// TODO: Pass the DataStore Settings
        /// </summary>
        /// <param name="connectionStringProvider">
        /// Connection string provider
        /// </param>
        public CosmosDbDataStore(
            IConnectionStringProvider connectionStringProvider,
            ILogger<CosmosDbDataStore> logger)
        {
            this.connectionStringProvider = connectionStringProvider;
            this.logger = logger;
            this.semaphoreSlim = new SemaphoreSlim(1, 1);
            tableMap = new ConcurrentDictionary<string, CloudTable>();
        }


        /// <summary>
        /// Init the Storage account and table client. 
        /// Todo: We need a way to reset the StorageAccount and Table connections during KeyRotation
        /// </summary>
        /// <returns></returns>
        public async Task InitAsync()
        {
            await semaphoreSlim.WaitAsync();

            try
            {
                if (this.storageAccount == null)
                {
                    logger.LogInformation($"Initializing CosmosDbDataStore connection");

                    IEnumerable<string> connectionStrs = await this.connectionStringProvider.GetConnectionStringsForTableAPI();

                    // For now, just use the first one. Since we are not rotating keys, any one of these should work
                    string connStr = connectionStrs.FirstOrDefault();
                    this.storageAccount = CloudStorageAccount.Parse(connStr);

                    // Create a table client for interacting with the table service
                    this.tableClient = storageAccount.CreateCloudTableClient(new TableClientConfiguration());
                }
            }
            catch (Exception ex)
            {
                this.storageAccount = null;
                this.tableClient = null;

                logger.LogError($"Failed to initialize Storage Account: {ex}");
                Console.WriteLine($"Failed to initialize CosmosDbDataStore connection: {ex}");
                throw;
            }
            finally
            {
                semaphoreSlim.Release();
            }
        }

        /// <summary>
        /// Creates the Table for a given tableName
        /// </summary>
        /// <param name="tableName"></param>
        public void CreateTable(string tableName)
        {
            try
            {
                if (tableMap.ContainsKey(tableName) &&
                    tableMap[tableName] != null &&
                    tableMap[tableName].Exists())
                {
                    logger.LogInformation($"Table: {tableName} is already created.");
                }
                else
                {
                    CloudTable cloudTable = tableClient.GetTableReference(tableName);
                    cloudTable.CreateIfNotExists();

                    // Add into the Table map.
                    // TODO: Revisit this when we need to work on cert rotation
                    bool added = tableMap.TryAdd(tableName, cloudTable);
                    if (!added)
                    {
                        logger.LogError($"TableName already exists: {cloudTable}");
                    }

                    logger.LogInformation($"Created Table {tableName}");
                }
            }
            catch (Exception ex)
            {
                logger.LogError($"Failed to create table: {tableName}, ex:{ex}");
                throw;
            }
        }

        /// <summary>
        /// Inserts or Update a TableEntity into a given table. Idempotent if the entity is already updated.
        /// </summary>
        /// <param name="tableName">table name</param>
        /// <param name="entity">table entity</param>
        /// <returns>Result of an operation</returns>
        public async Task<TableResult> InsertOrMergeEntityAsync(string tableName, TableEntity entity)
        {
            try
            {
                CloudTable cloudTable = GetTable(tableName);

                // Execute the operation.
                TableOperation insertOrMergeOperation = TableOperation.InsertOrMerge(entity);
                TableResult tableResult = await ExecuteAsync(cloudTable, insertOrMergeOperation, "InsertOrMergeEntity");

                return tableResult;
            }
            catch (Exception ex)
            {
                logger.LogError($"Failed to Insert entry in table: {tableName}, entity :{entity} ex:{ex}");
                throw;
            }
        }

        /// <summary>
        /// Inserts or replace a TableEntity into a given table. Idempotent if the entity is already updated.
        /// </summary>
        /// <param name="tableName">table name</param>
        /// <param name="entity">table entity</param>
        /// <returns>Result of an operation</returns>
        public async Task<TableResult> InsertOrReplaceEntityAsync(string tableName, TableEntity entity)
        {
            try
            {
                CloudTable cloudTable = GetTable(tableName);

                // Execute the operation.
                TableOperation insertOrReplaceOperation = TableOperation.InsertOrReplace(entity);
                TableResult tableResult = await ExecuteAsync(cloudTable, insertOrReplaceOperation, "InsertOrReplace");

                return tableResult;
            }
            catch (Exception ex)
            {
                logger.LogError($"Failed to Insert entry in table: {tableName}, entity :{entity} ex:{ex}");
                throw;
            }
        }

        /// <summary>
        /// Delete a TableEntity in the given table.
        /// </summary>
        /// <param name="tableName">table name</param>
        /// <param name="entity">table entity</param>
        /// <returns>Result of an operation</returns>
        public async Task<TableResult> DeleteEntityAsync(string tableName, TableEntity entity)
        {
            try
            {
                CloudTable cloudTable = GetTable(tableName);

                // Execute the operation.
                TableOperation deleteOperation = TableOperation.Delete(entity);
                TableResult tableResult = await ExecuteAsync(cloudTable, deleteOperation, "Delete");

                return tableResult;
            }
            catch (Exception ex)
            {
                logger.LogError($"Failed to Delete entry in table: {tableName}, entity :{entity} ex:{ex}");
                throw;
            }
        }

        /// <summary>
        /// Get the Table entity of a given tablename
        /// </summary>
        /// <typeparam name="T">TableEntity type</typeparam>
        /// <param name="tableName">table name</param>
        /// <param name="partitionKey">parition Key</param>
        /// <param name="rowKey">row key</param>
        /// <returns>Retrieved TableEntity</returns>
        public async Task<TableResult> GetEntityAsync<T>(string tableName, string partitionKey, string rowKey) where T : ITableEntity
        {
            try
            {
                CloudTable cloudTable = GetTable(tableName);

                // Execute the operation.
                TableOperation getOperation = TableOperation.Retrieve<T>(partitionKey, rowKey);
                TableResult tableResult = await ExecuteAsync(cloudTable, getOperation, "GetEntity");

                return tableResult;
            }
            catch (Exception ex)
            {
                logger.LogError($"Failed to Get entry in table: {tableName}, partitionKey :{partitionKey}, rowKey: {rowKey}, ex:{ex}");
                throw;
            }
        }

        /// <summary>
        /// Get the list of Table Entities for a given table query
        /// </summary>
        /// <typeparam name="T">TableEntity type</typeparam>
        /// <param name="tableName">table name</param>
        /// <param name="query">Table query</param>
        /// <param name="rowKey">row key</param>
        /// <returns>Retrieved TableEntity</returns>
        public async Task<ICollection<T>> GetEntitiesAsync<T>(string tableName, TableQuery<T> query) where T : ITableEntity, new()
        {
            try
            {
                CloudTable cloudTable = GetTable(tableName);
                TableContinuationToken continuationToken = null;
                List<T> resultList = new List<T>();

                do
                {
                    // Loop until a null continuation token is received, indicating the end of the table.
                    var tableQueryResult = await cloudTable.ExecuteQuerySegmentedAsync(query, continuationToken);
                    if (tableQueryResult.Results?.Count > 0)
                    {
                        resultList.AddRange(tableQueryResult.Results);
                    }

                    continuationToken = tableQueryResult.ContinuationToken;
                }
                while (continuationToken != null);

                return resultList;
            }
            catch (Exception ex)
            {
                logger.LogError($"Failed to Get entities in table: {tableName}, ex:{ex}");
                throw;
            }
        }

        private async Task<TableResult> ExecuteAsync(CloudTable cloudTable, TableOperation operation, string opName)
        {
            TableResult tableResult = await cloudTable.ExecuteAsync(operation);

            // Get the request units consumed by the current operation. 
            // RequestCharge of a TableResult is only applied to Azure Cosmos DB
            if (tableResult.RequestCharge.HasValue)
            {
                // TODO: Change it to a Metric event
                logger.LogInformation($"{cloudTable.Name}: Request Charge of {opName} Operation: " + tableResult.RequestCharge);
            }

            return tableResult;
        }

        private CloudTable GetTable(string tableName)
        {
            CloudTable cloudTable;
            bool tableExists = tableMap.TryGetValue(tableName, out cloudTable);
            if (!tableExists)
            {
                throw new Exception($"cloudTable is not created for TableName:{tableName}");
            }

            ValidateTableExists(cloudTable);

            return cloudTable;
        }

        private void ValidateTableExists(CloudTable cloudTable)
        {
            if (cloudTable == null)
            {
                throw new ArgumentNullException("cloudTable is not created");
            }

            if (!cloudTable.Exists())
            {
                throw new InvalidOperationException($"Cloud Table {cloudTable.Name} doesnt exists");
            }
        }
    }
}
