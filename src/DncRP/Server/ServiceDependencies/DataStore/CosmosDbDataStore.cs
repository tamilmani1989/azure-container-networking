// -----------------------------------------------------------------------------
// <copyright file="CosmosDbDataStore.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using Microsoft.Azure.Cosmos.Table;
    using Microsoft.Extensions.Configuration;
    using Microsoft.Extensions.Logging;

    /// <summary>
    /// Initialize CosmosDB Table storage
    /// </summary>
    public class CosmosDbDataStore : ITableDataStore
    {
        private IConfiguration configuration;
        private string connectionString;
        private CloudStorageAccount storageAccount;
        private CloudTableClient tableClient;
        ILogger<CosmosDbDataStore> logger;

        /// <summary>
        /// Create an instance of CosmosDbDataStorage
        /// TODO: Pass the DataStore Settings
        /// </summary>
        /// <param name="storageConnectionString">Connection string, configuration from appsettings.Json</param>
        public CosmosDbDataStore(
            IConfiguration configuration,
            ILogger<CosmosDbDataStore> logger)
        {
            this.configuration = configuration;
            this.logger = logger;

            // Get the connection string
            // TODO - This will be pulled from KeyVault using MSI
            connectionString = this.configuration["DataStoreSettings:ConnectionString"];
            if (string.IsNullOrWhiteSpace(connectionString))
            {
                throw new Exception("Storage Connection String (DataStoreSettings:ConnectionString) is not set");
            }

            Init();
        }

        /// <summary>
        /// Initialize the CloudStorage
        /// </summary>
        /// <returns></returns>
        private void Init()
        {
            // Retrieve storage account information from connection string.try
            try
            {
                storageAccount = CloudStorageAccount.Parse(connectionString);
            }
            catch (Exception ex)
            {
                logger.LogError($"Failed to initialize Storage Account: {ex}");
                throw;
            }

            // Create a table client for interacting with the table service
            tableClient = storageAccount.CreateCloudTableClient(new TableClientConfiguration());
        }

        public CloudTableClient GetTableClient()
        {
            return tableClient;
        }
    }
}
