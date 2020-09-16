// --------------------------------------------------------------------------------------------------------------------
// <copyright file="DncController.cs" company="Microsoft">
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
    /// Class to handle CRUD operations on managed DNC cluster
    /// </summary>
    [Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DelegatedNetwork/controller/{resourceName}")]
    public class DncController : DncRpControllerBase<DncController>
    {
        public DncController(
            ITableOperations tableOperations,
            ILogger<DncController> logger) : base(logger, tableOperations)
        {
            logger.LogInformation("DncController initialized");
        }

        /// <summary>
        /// Get the DNC controller resource
        /// </summary>
        /// <returns></returns>
        [HttpGet]
        public async Task<ContentResult> GetDncResource(string subscriptionId, string resourceGroupName, string resourceName)
        {
            return await this.ExecuteWithRetryAsync(() => this.tableOperations.ControllerConfiguration.GetDncControllerAsync(
                subscriptionId, resourceGroupName, resourceName), HttpMethod.Get);
        }

        /// <summary>
        ///  PUT the DNC controller resource. It will create dnc resource if it doesn't exists
        /// </summary>
        /// <returns></returns>
        [HttpPut]
        public async Task<ContentResult> PutDncResource(string subscriptionId, string resourceGroupName, string resourceName)
        {
            if (HttpContext.Request.Body == null)
            {
                throw new EmptyRequestBody("Empty request body");
            }

            ControllerRequest request = null;
            using (StreamReader stream = new StreamReader(HttpContext.Request.Body))
            {
                string jsonBody = await stream.ReadToEndAsync();
                request = JsonUtility.Deserialize<ControllerRequest>(jsonBody);
            }

            return await this.ExecuteWithRetryAsync(() => this.tableOperations.ControllerConfiguration.InsertOrReplaceDncControllerAsync(
                subscriptionId, resourceGroupName, resourceName, request), HttpMethod.Put);
        }

        /// <summary>
        ///  Patch the DNC controller resource
        /// </summary>
        /// <returns></returns>
        [HttpPatch]
        public async Task<ContentResult> PatchDncResource(string subscriptionId, string resourceGroupName, string resourceName)
        {
            // TODO : Get this from HttpContext
            ControllerRequest requestBody = null;

            return await this.ExecuteWithRetryAsync(() => this.tableOperations.ControllerConfiguration.MergeDncControllerAsync(
                subscriptionId, resourceGroupName, resourceName, requestBody), HttpMethod.Patch);
        }

        /// <summary>
        ///  Delete the DNC controller resource
        /// </summary>
        /// <returns></returns>
        [HttpDelete]
        public async Task<ContentResult> DeleteDncResource(string subscriptionId, string resourceGroupName, string resourceName)
        {
            return await this.ExecuteWithRetryAsync(() => this.tableOperations.ControllerConfiguration.DeleteDncControllerAsync(
               subscriptionId, resourceGroupName, resourceName), HttpMethod.Delete);
        }
    }
}
