// -----------------------------------------------------------------------------
// <copyright file="OrchestratorOperations.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Threading.Tasks;
    using Microsoft.Extensions.Logging;

    public class OrchestratorOperations : IOrchestratorOperations
    {
        private ITableDataStore dataStore;
        private const string resourceTableName = "OrchestratorInstancesResource";
        private const string configTableName = "OrchestratorInstancesConfiguration";
        private ILogger logger;

        /// <summary>
        /// Create an instance of OrchestratorOperations
        /// </summary>
        /// <param name="tableDataStore"></param>
        /// <param name="logger"></param>
        public OrchestratorOperations(
            ITableDataStore tableDataStore,
            ILogger logger)
        {
            this.dataStore = tableDataStore;
            this.logger = logger;

            this.dataStore.CreateTable(resourceTableName);
            this.dataStore.CreateTable(configTableName);
        }

        /// <summary>
        /// Get Orchestrator
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="resourceName"></param>
        /// <returns></returns>
        public Task<Orchestrator> GetOrchestratorAsync(string subId, string resourceGroup, string resourceName)
        {
            throw new NotImplementedException();
        }

        /// <summary>
        /// Create a new Orchestrator or replace an existing one
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="resourceName"></param>
        /// <param name="orchestratorRequest"></param>
        /// <returns></returns>
        public Task<Orchestrator> InsertOrReplaceOrchestratorAsync(
            string subId, string resourceGroup, string resourceName, OrchestratorRequest orchestratorRequest)
        {
            throw new NotImplementedException();
        }

        /// <summary>
        /// Modify an existing Orchestrator
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="resourceName"></param>
        /// <param name="orchestratorRequest"></param>
        /// <returns></returns>
        public Task<Orchestrator> MergeOrchestratorAsync(
            string subId, string resourceGroup, string resourceName, OrchestratorRequest orchestratorRequest)
        {
            throw new NotSupportedException();
        }

        /// <summary>
        /// Delete Orchestrator
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="resourceName"></param>
        /// <returns></returns>
        public Task<Orchestrator> DeleteOrchestratorAsync(string subId, string resourceGroup, string resourceName)
        {
            throw new NotImplementedException();
        }
    }
}
