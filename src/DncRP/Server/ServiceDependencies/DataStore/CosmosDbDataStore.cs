// -----------------------------------------------------------------------------
// <copyright file="CosmosDbDataStore.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Collections.Generic;
    using System.Linq;
    using System.Threading;
    using System.Threading.Tasks;
    using Microsoft.Azure.Cosmos.Table;
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
        }


        /// <summary>
        /// Get Table client.
        /// TODO: This implementation will change based on whatever
        /// TODO: key rotation mechanism we come up with.
        /// </summary>
        /// <returns></returns>
        public async Task<CloudTableClient> GetTableClient()
        {
            await semaphoreSlim.WaitAsync();

            try
            {
                if (this.storageAccount == null)
                {
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
                throw;
            }
            finally
            {
                semaphoreSlim.Release();
            }

            return this.tableClient;
        }
    }
}
