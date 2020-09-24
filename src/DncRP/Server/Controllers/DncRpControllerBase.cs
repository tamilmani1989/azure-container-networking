// --------------------------------------------------------------------------------------------------------------------
// <copyright file="DncRpControllerBase.cs" company="Microsoft">
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
    using Microsoft.AspNetCore.Authorization;
    using Microsoft.AspNetCore.Http.Extensions;
    using Microsoft.AspNetCore.Mvc;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Configuration;
    using Microsoft.Extensions.Logging;

    [ApiController]
    [Authorize]
    public class DncRpControllerBase<T> : ControllerBase
    {
        protected readonly ILogger<T> logger;
        protected readonly ITableOperations tableOperations;

        public DncRpControllerBase(
            ILogger<T> logger,
            ITableOperations tableOperations)
        {
            this.logger = logger;
            this.tableOperations = tableOperations;
        }

        /// <summary>
        /// Execute a dnc controller API with retry and convert the result to ContentResult
        /// </summary>
        /// <param name="action"></param>
        /// <param name="includeReponseInResult"></param>
        /// <returns></returns>
        protected async Task<ContentResult> ExecuteWithRetryAsync<TResouce>(
            Func<Task<TResouce>> action,
            HttpMethod method,
            string methodNameForLogging) where TResouce : TopLevelResource
        {
            TResouce response = null;

            try
            {
                response = await AsyncRetry.Retry<TResouce>(async () =>
                {
                    return (TResouce)await action();
                },
                3, TimeSpan.FromSeconds(10), TimeSpan.FromMinutes(5), (e) => e is HttpRequestException).ConfigureAwait(false);
            }
            catch (Exception ex)
            {
                logger.LogError("Error in API: {0}, {1}", methodNameForLogging, ex);
                return this.ConvertToExceptionResult(ex.Message, (int)HttpStatusCode.InternalServerError);
            }

            return this.ConvertToContentResult(response, method, methodNameForLogging);
        }

        /// <summary>
        /// Convert result to JSON
        /// </summary>
        /// <param name="response"></param>
        /// <returns></returns>
        protected ContentResult ConvertToExceptionResult(string message, int code)
        {
            return new ContentResult
            {
                Content = ErrorDetails.CreateErrorAsJson(message, code),
                ContentType = "application/json",
                StatusCode = code
            };
        }

        /// <summary>
        /// Convert result to JSON
        /// </summary>
        /// <param name="response"></param>
        /// <returns></returns>
        protected ContentResult ConvertToContentResult<TResouce>(TResouce response, HttpMethod method, string methodNameForLogging) where TResouce : TopLevelResource
        {
            if (response.Error != null)
            {
                string apiErrorCode = response.Error.ErrorResponseCode == ErrorResponseCode.None
                    ? string.Empty
                    : response.Error.ErrorResponseCode.ToString();

                string jsonError = ErrorDetails.CreateErrorAsJson(
                    response.Error.Error, response.Error.StatusCode, apiErrorCode);
                int errorCode = (int)response.Error.StatusCode;
                logger.LogError("Error in API {0}. Code:{1}, ErrorMsg:{2}", methodNameForLogging, errorCode, jsonError);

                return new ContentResult
                {
                    Content = jsonError,
                    ContentType = "application/json",
                    StatusCode = errorCode
                };
            }

            int statusCode = (int)HttpStatusCode.OK;
            if (response.Properties != null && response.Properties.State != ProvisioningState.Succeeded)
            {
                if (method == HttpMethod.Delete && response.Properties.State == ProvisioningState.Deleting)
                {
                    statusCode = (int)HttpStatusCode.Accepted;
                }
                else if ((method == HttpMethod.Put || method == HttpMethod.Post) &&
                          response.Properties.State == ProvisioningState.Provisioning)
                {
                    statusCode = (int)HttpStatusCode.Created;
                }
            }

            string json =  method == HttpMethod.Delete ? string.Empty : JsonUtility.Serialize(response);

            logger.LogInformation("Success from API {0}. Code:{1}, Response:{2}", methodNameForLogging, statusCode, json);

            return new ContentResult
            {
                Content = json,
                ContentType = "application/json",
                StatusCode = statusCode
            };
        }

        protected async Task<TRequest> GetRequestAsync<TRequest>(Stream requestBody, string operationName)
        {
            if (requestBody == null)
            {
                throw new EmptyRequestBody("Empty request body");
            }

            TRequest request = default(TRequest);
            using (StreamReader stream = new StreamReader(requestBody))
            {
                string jsonBody = await stream.ReadToEndAsync();
                this.logger.LogInformation($"{operationName} Request body: {jsonBody}");

                request = JsonUtility.Deserialize<TRequest>(jsonBody);
            }

            return request;
        }

        
    }
}
