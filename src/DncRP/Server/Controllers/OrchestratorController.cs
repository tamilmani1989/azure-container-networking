// --------------------------------------------------------------------------------------------------------------------
// <copyright file="OrchestratorController.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.IO;
    using System.Net;
    using System.Net.Http;
    using System.Threading.Tasks;
    using Microsoft.AspNetCore.Mvc;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Logging;

    /// <summary>
    /// Class to handle CRUD operations for Orchestrator
    /// </summary>
    [Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DelegatedNetwork/orchestrators/{resourceName}")]
    public class OrchestratorController : DncRpControllerBase<OrchestratorController>
    {
        public OrchestratorController(
            ITableOperations tableOperations,
            ILogger<OrchestratorController> logger) : base(logger, tableOperations)
        {
            logger.CLogInformation("OrchestratorController initialized");
        }

        /// <summary>
        /// Get the Orchestrator resource
        /// </summary>
        /// <returns></returns>
        [HttpGet]
        public async Task<ContentResult> GetOrchestratorResource(string subscriptionId, string resourceGroupName, string resourceName)
        {
            return await this.ExecuteWithRetryAsync(subscriptionId, () => this.tableOperations.OrchestratorOperations.GetOrchestratorAsync(
                subscriptionId, resourceGroupName, resourceName), HttpMethod.Get, "GetOrchestrator");
        }

        /// <summary>
        ///  PUT the Orchestrator resource. It will create Orchestrator resource if it doesn't exists
        /// </summary>
        /// <returns></returns>
        [HttpPut]
        public async Task<ContentResult> PutOrchestratorResource(string subscriptionId, string resourceGroupName, string resourceName)
        {
            if (HttpContext.Request.Body == null)
            {
                throw new EmptyRequestBody("Empty request body");
            }

            OrchestratorRequest request = null;
            using (StreamReader stream = new StreamReader(HttpContext.Request.Body))
            {
                string jsonBody = await stream.ReadToEndAsync();
                this.logger.CLogInformation("PutOrchestratorResource body: {0}", jsonBody);

                request = JsonUtility.Deserialize<OrchestratorRequest>(jsonBody);
                request.Identity = this.GetIdentityFromRequestHeaders(HttpContext.Request, request);
            }

            return await this.ExecuteWithRetryAsync(subscriptionId, () => this.tableOperations.OrchestratorOperations.InsertOrReplaceOrchestratorAsync(
                subscriptionId, resourceGroupName, resourceName, request), HttpMethod.Put, "PutOrchestrator");
        }

        /// <summary>
        ///  Patch the Orchestrator resource
        /// </summary>
        /// <returns></returns>
        [HttpPatch]
#pragma warning disable CS1998 // Async method lacks 'await' operators and will run synchronously. NotSupported yet
        public async Task<ContentResult> PatchOrchestratorResource(string subscriptionId, string resourceGroupName, string resourceName)
#pragma warning restore CS1998 // Async method lacks 'await' operators and will run synchronously. NotSupported yet
        {
            throw new NotSupportedException();
        }

        /// <summary>
        ///  Delete the Orchestrator resource
        /// </summary>
        /// <returns></returns>
        [HttpDelete]
        public async Task<ContentResult> DeleteOrchestratorResource(string subscriptionId, string resourceGroupName, string resourceName)
        {
            return await this.ExecuteWithRetryAsync(subscriptionId, () => this.tableOperations.OrchestratorOperations.DeleteOrchestratorAsync(
               subscriptionId, resourceGroupName, resourceName), HttpMethod.Delete, "DeleteOrchestrator");
        }
    }
}
