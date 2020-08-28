// -----------------------------------------------------------------------------
// <copyright file="SubnetTokenConfiguration.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Threading.Tasks;
    using Microsoft.Azure.Cosmos.Table;

    /// <summary>
    /// DataModel for SubnetTokenConfiguration operations
    /// </summary>
    public class SubnetTokenConfiguration : ISubnetTokenConfiguration
    {
        private ITableDataStore dataStore;
        private CloudTable subnetTokenTable;
        private const string tableName = "SubnetTokenConfiguration";

        /// <summary>
        /// Create an instance of TableOperations
        /// TODO: Pass the DataStore Settings
        /// </summary>
        /// <param name="storageConnectionString">Connection string, configuration from appsettings.Json</param>
        public SubnetTokenConfiguration(
            ITableDataStore tableDataStore)
        {
            this.dataStore = tableDataStore;

            // Create a table client for interacting with the table service 
            CloudTableClient tableClient = dataStore.GetTableClient();
            if (tableClient == null)
            {
                throw new ArgumentException("TableClient is not initialized");
            }

            subnetTokenTable = dataStore.GetTableClient().GetTableReference(tableName);
            if (subnetTokenTable.CreateIfNotExists())
            {
                //logger.LogInformation($"Created Table named: {tableName}");
            }
            else
            {
                //logger.LogInformation($"Table {tableName} already exists");
            }
        }

        /// <summary>
        /// Insert of update into SubnetTokenConfiguration table.
        /// </summary>
        /// <param name="vnetId"></param>
        /// <param name="subnetName"></param>
        /// <param name="primaryTokenId"></param>
        /// <param name="secondaryTokenId"></param>
        public async Task InsertOrUpdate(
            string vnetId,
            string subnetName,
            string primaryTokenId,
            string secondaryTokenId)
        {
            try
            {
                // Create the InsertOrReplace table operation
                SubnetTokenConfigurationEntity entity = new SubnetTokenConfigurationEntity(vnetId, subnetName);
                entity.PrimaryTokenId = primaryTokenId;
                entity.SecondaryTokenId = secondaryTokenId;

                TableOperation insertOrMergeOperation = TableOperation.InsertOrMerge(entity);

                // Execute the operation.
                TableResult result = await subnetTokenTable.ExecuteAsync(insertOrMergeOperation);
                SubnetTokenConfigurationEntity insertedCustomer = result.Result as SubnetTokenConfigurationEntity;

                // Get the request units consumed by the current operation. RequestCharge of a TableResult is only applied to Azure Cosmos DB
                if (result.RequestCharge.HasValue)
                {
                    Console.WriteLine("Request Charge of InsertOrMerge Operation: " + result.RequestCharge);
                }
            }
            catch (StorageException e)
            {
                // Add logger
                throw;
            }
        }
    }
}
