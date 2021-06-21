// --------------------------------------------------------------------------------------------------------------------
// <copyright file="DncController.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Collections.Generic;
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
            logger.CLogInformation("DncController initialized");
        }

        /// <summary>
        /// Get the DNC controller resource
        /// </summary>
        /// <returns></returns>
        [HttpGet]
        public async Task<ContentResult> GetDncResource(string subscriptionId, string resourceGroupName, string resourceName)
        {
            return await this.ExecuteWithRetryAsync(subscriptionId, () => this.tableOperations.ControllerConfiguration.GetDncControllerAsync(
                subscriptionId, resourceGroupName, resourceName), HttpMethod.Get, "GetController");
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
                this.logger.CLogInformation("PutControllerResource body: {0}", jsonBody);

                request = JsonUtility.Deserialize<ControllerRequest>(jsonBody);
            }

            ValidateRequestParameters(request);

            return await this.ExecuteWithRetryAsync(subscriptionId, () => this.tableOperations.ControllerConfiguration.InsertOrReplaceDncControllerAsync(
                subscriptionId, resourceGroupName, resourceName, request), HttpMethod.Put, "PutController");
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

            return await this.ExecuteWithRetryAsync(subscriptionId, () => this.tableOperations.ControllerConfiguration.MergeDncControllerAsync(
                subscriptionId, resourceGroupName, resourceName, requestBody), HttpMethod.Patch, "PatchController");
        }

        /// <summary>
        ///  Delete the DNC controller resource
        /// </summary>
        /// <returns></returns>
        [HttpDelete]
        public async Task<ContentResult> DeleteDncResource(string subscriptionId, string resourceGroupName, string resourceName)
        {
            Controller controller = await this.tableOperations.ControllerConfiguration.GetDncControllerAsync(
                subscriptionId,
                resourceGroupName,
                resourceName);

            List<DelegateSubnet> delegatedSubnetsList = await this.tableOperations.ControllerConfiguration.GetDelegatedSubnetResourcesByControllerId(controller.Id);
            if (delegatedSubnetsList.Count > 0)
            {
                string dsResources = "";
                foreach(DelegateSubnet ds in delegatedSubnetsList)
                {
                    dsResources = dsResources + "," + ds.Name;
                }

                throw new ErrorResponseMessageException(
                        httpStatus: HttpStatusCode.Conflict,
                        errorCode: ErrorResponseCode.DependentResourceExists,
                        errorMessage: $"Dependent delegated subnet resource exists:{dsResources}");
            }


            List<Orchestrator> orchestratorsList = await this.tableOperations.ControllerConfiguration.GetOrchestratorResourcesByControllerId(controller.Id);
            if (orchestratorsList.Count > 0)
            {
                string orchResources = "";
                foreach (Orchestrator o in orchestratorsList)
                {
                    orchResources = orchResources + "," + o.Name;
                }

                throw new ErrorResponseMessageException(
                        httpStatus: HttpStatusCode.Conflict,
                        errorCode: ErrorResponseCode.DependentResourceExists,
                        errorMessage: $"Dependent orchestrator resource exists.{orchResources}");
            }

            return await this.ExecuteWithRetryAsync(subscriptionId, () => this.tableOperations.ControllerConfiguration.DeleteDncControllerAsync(
               subscriptionId, resourceGroupName, resourceName), HttpMethod.Delete, "DeleteController");
        }

        public void ValidateRequestParameters(ControllerRequest request)
        {
            if (string.IsNullOrEmpty(request.Location))
            {
                throw new ErrorResponseMessageException(HttpStatusCode.BadRequest, ErrorResponseCode.None, "location parameter not present");
            }
        }
    }
}
