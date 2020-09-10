// --------------------------------------------------------------------------------------------------------------------
// <copyright file="DncController.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Net;
    using System.Net.Http;
    using System.Text.Json;
    using System.Threading.Tasks;
    using Microsoft.AspNetCore.Authorization;
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
            return await this.ExecuteWithRetryAsync(() => this.tableOperations.ControllerConfiguration.GetDncController(
                subscriptionId, resourceGroupName, resourceName));
        }

        /// <summary>
        ///  PUT the DNC controller resource. It will create dnc resource if it doesn't exists
        /// </summary>
        /// <returns></returns>
        [HttpPut]
        public async Task<ContentResult> PutDncResource(string subscriptionId, string resourceGroupName, string resourceName)
        {
            // TODO : Get this from HttpContext
            ControllerRequest requestBody = null;

            return await this.ExecuteWithRetryAsync(() => this.tableOperations.ControllerConfiguration.InsertOrReplaceDncController(
                subscriptionId, resourceGroupName, resourceName, requestBody));
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

            return await this.ExecuteWithRetryAsync(() => this.tableOperations.ControllerConfiguration.MergeDncController(
                subscriptionId, resourceGroupName, resourceName, requestBody));
        }

        /// <summary>
        ///  Delete the DNC controller resource
        /// </summary>
        /// <returns></returns>
        [HttpDelete]
        public async Task<ContentResult> DeleteDncResource(string subscriptionId, string resourceGroupName, string resourceName)
        {
            return await this.ExecuteWithRetryAsync(() => this.tableOperations.ControllerConfiguration.DeleteDncController(
               subscriptionId, resourceGroupName, resourceName), false);
        }

        /// <summary>
        /// Execute a dnc controller API with retry and convert the result to ContentResult
        /// </summary>
        /// <param name="action"></param>
        /// <param name="includeReponseInResult"></param>
        /// <returns></returns>
        private async Task<ContentResult> ExecuteWithRetryAsync(Func<Task<ControllerResponse>> action, bool includeReponseInResult = true)
        {
            ControllerResponse response = null;

            try
            {
                response = await AsyncRetry.Retry<ControllerResponse>(async () =>
                {
                    return await action();
                },
                3, TimeSpan.FromSeconds(10), TimeSpan.FromMinutes(5), (e) => e is HttpRequestException).ConfigureAwait(false);
            }
            catch (Exception ex)
            {
                logger.LogError("Error in DncController API: {0}", ex);

                response = new ControllerResponse
                {
                    Error = new ResponseError
                    {
                        Error = ex.Message,
                        StatusCode = HttpStatusCode.InternalServerError
                    }
                };
            }

            return this.ConvertToContentResult(response, includeReponseInResult);
        }

        /// <summary>
        /// Convert result to JSON
        /// </summary>
        /// <param name="response"></param>
        /// <returns></returns>
        private ContentResult ConvertToContentResult(ControllerResponse response, bool includeReponseInContent = true)
        {
            if (response.Error != null)
            {
                return new ContentResult
                {
                    Content = ErrorDetails.CreateErrorAsJson(response.Error.Error, (int)response.Error.StatusCode),
                    ContentType = "application/json",
                    StatusCode = (int)response.Error.StatusCode
                };
            }

            int statusCode = response.Properties != null ? (int)response.Properties.State : (int)HttpStatusCode.OK;
            string json = includeReponseInContent ? JsonSerializer.Serialize(response) : "";
            return new ContentResult
            {
                Content = json,
                ContentType = "application/json",
                StatusCode = statusCode
            };
        }
    }
}
