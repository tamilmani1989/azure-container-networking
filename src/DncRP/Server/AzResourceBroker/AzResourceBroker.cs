// -----------------------------------------------------------------------------
// <copyright file="AzResourceBroker.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Collections.Generic;
    using System.Net.Http;
    using System.Threading.Tasks;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Configuration;
    using Microsoft.Extensions.Logging;

    /// <summary>
    /// AzureResourceBroker used to GET/PUT/POST HTTP calls to other Service Providers.
    /// </summary>
    public class AzResourceBroker
    {
        /// <summary>
        /// Stores Logger
        /// </summary>
        protected ILogger logger;

        /// <summary>
        /// HTTPUtility
        /// </summary>
        protected IHttpUtility httpUtility;

        /// <summary>
        /// Gets the ARM endpoint.
        /// </summary>
        protected virtual string ArmEndpoint
        {
            get
            {
                return AzureInfo.GetArmEndpoint(CurrentContext.AzureEnvironment);
            }
        }

        /// <summary>
        /// AzureResourceBroker constructor
        /// </summary>
        /// <param name="authenticationTokenBroker"></param>
        /// <param name="logger"></param>
        public AzResourceBroker(
            IHttpUtility httpUtility, 
            ILogger logger)
        {
            this.httpUtility = httpUtility;
            this.logger = logger;
        }

        /// <summary>
        /// Puts the ARM resource.
        /// </summary>
        /// <typeparam name="T">The type of the resource</typeparam>
        /// <param name="subscriptionId">subscriptionId.</param>
        /// <param name="resourceId">The resource Id.</param>
        /// <param name="apiVersion">The API version.</param>
        /// <param name="resource">The desired resource.</param>
        /// <returns>The new resource.</returns>
        protected Task<T> PutResource<T>(
            string subscriptionId,
            string resourceId, 
            string apiVersion, 
            T resource) where T : class
        {
            var requestUri = GetResourceUri(resourceId.TrimEnd('/'), apiVersion);

            return this.CallServerInternal<T, T>(subscriptionId: subscriptionId, httpMethod: HttpMethod.Put, requestUri: requestUri, requestBody: resource);
        }

        /// <summary>
        /// Posts the ARM resource's action.
        /// </summary>
        /// <typeparam name="TRequest">The type of request</typeparam>
        /// <typeparam name="TResponse">The type of response</typeparam>
        /// <param name="subscriptionId">SubscriptionId.</param>
        /// <param name="resourceId">The resource Id.</param>
        /// <param name="action">The action.</param>
        /// <param name="apiVersion">The API version.</param>
        /// <param name="request">The request.</param>
        /// <returns>The response.</returns>
        protected Task<TResponse> PostResourceActionAsync<TRequest, TResponse>(
            string subscriptionId,
            string resourceId, 
            string action, 
            string apiVersion, 
            TRequest request) where TResponse : class where TRequest : class
        {
            var requestUri = GetResourceActionUri(resourceId.TrimEnd('/'), action.Trim('/'), apiVersion);

            return this.CallServerInternal<TRequest, TResponse>(subscriptionId: subscriptionId, httpMethod: HttpMethod.Post, requestUri: requestUri, requestBody: request);
        }

        /// <summary>
        /// Get the access token and invoke the HTTPUtility SendAsync
        /// </summary>
        /// <typeparam name="TRequest">RequestType</typeparam>
        /// <typeparam name="TResponse">ResponseType</typeparam>
        /// <param name="subscriptionId">subscription Id of the resource</param>
        /// <param name="httpMethod">HTTP method type (put/post)</param>
        /// <param name="requestUri">Request URI</param>
        /// <param name="requestBody">Request body of type TRequest</param>
        /// <returns>response of type TResponse</returns>
        private async Task<TResponse> CallServerInternal<TRequest, TResponse>(
            string subscriptionId,
            HttpMethod httpMethod,
            Uri requestUri,
            TRequest requestBody) where TResponse : class where TRequest : class
        {
            // TODO Call AzureAD to get AccessToken (pending in Vipins PR) 
            //var tenantId = this.armAuthTokenBroker.GetTenantIdFromSubscriptionId(subscriptionId);
            //var accessToken = await this.armAuthTokenBroker
            //    .GetAuthenticationToken(subscriptionId)
            //    .ConfigureAwait(continueOnCapturedContext: false);
            string accessToken = "";

            return await httpUtility.SendRequestAndGetJsonAsync<TRequest, TResponse>(httpMethod, requestUri, requestBody, accessToken);
        }

        /// <summary>
        /// Gets ResourceUri.
        /// Create URI template as {ArmEndpoint}/{resourceId}?api-version={apiVersion}"
        /// </summary>
        /// <param name="resourceId">Resource Id in URI format</param>
        /// <param name="apiversion">API version</param>
        /// <returns></returns>
        private Uri GetResourceUri(
            string resourceId,
            string apiVersion)
        {
            return new Uri($"{ArmEndpoint}{resourceId}?api-version={apiVersion}");
        }

        /// <summary>
        /// Gets Action ResourceUri.
        /// Create URI template as {ArmEndpoint}/{resourceId}/{action}?api-version={apiVersion}"
        /// </summary>
        /// <param name="resourceId">Resource Id in URI format</param>
        /// <param name="action">Resource action</param>
        /// <param name="apiversion">apiversion</param>
        /// <returns></returns>
        private Uri GetResourceActionUri(
            string resourceId,
            string action,
            string apiVersion)
        {
            return new Uri($"{ArmEndpoint}{resourceId}/{action}?api-version={apiVersion}");
        }
    }
}
