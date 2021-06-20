// -----------------------------------------------------------------------------
// <copyright file="IOrchestratorOperations.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Threading.Tasks;

    public interface IOrchestratorOperations
    {
        /// <summary>
        /// Get Orchestrator
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="resourceName"></param>
        /// <returns></returns>
        Task<Orchestrator> GetOrchestratorAsync(string subId, string resourceGroup, string resourceName);

        /// <summary>
        /// Create a new Orchestrator or replace an existing one
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="resourceName"></param>
        /// <param name="orchestratorRequest"></param>
        /// <returns></returns>
        Task<Orchestrator> InsertOrReplaceOrchestratorAsync(
            string subId, string resourceGroup, string resourceName, string privateEndpointName, string resourceGuid, OrchestratorRequest orchestratorRequest);

        /// <summary>
        /// Modify an existing Orchestrator
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="resourceName"></param>
        /// <param name="orchestratorRequest"></param>
        /// <returns></returns>
        Task<Orchestrator> MergeOrchestratorAsync(
            string subId, string resourceGroup, string resourceName, OrchestratorRequest orchestratorRequest);

        /// <summary>
        /// Delete Orchestrator
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="resourceName"></param>
        /// <returns></returns>
        Task<Orchestrator> DeleteOrchestratorAsync(string subId, string resourceGroup, string resourceName);
    }
}
