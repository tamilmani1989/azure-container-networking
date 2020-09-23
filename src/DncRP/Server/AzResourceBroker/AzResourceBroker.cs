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
    using System.Linq;
    using System.Net;
    using System.Net.Http;
    using System.Net.Http.Formatting;
    using System.Net.Http.Headers;
    using System.Text;
    using System.Text.Json;
    using System.Threading.Tasks;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Configuration;
    using Microsoft.Extensions.Logging;
    using Microsoft.Rest.Serialization;
    using Newtonsoft.Json;

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
        /// IAzureAD
        /// </summary>
        protected IAzureAD azureAd;

        /// <summary>
        /// The deserialization settings used for Azure RP communication
        /// </summary>
        protected static readonly JsonSerializerSettings DeserializationSettings = new JsonSerializerSettings
        {
            DateFormatHandling = DateFormatHandling.IsoDateFormat,
            DateTimeZoneHandling = DateTimeZoneHandling.Utc,
            NullValueHandling = NullValueHandling.Ignore,
            ReferenceLoopHandling = ReferenceLoopHandling.Serialize,
            ContractResolver = new ReadOnlyJsonContractResolver(),
            Converters = new List<JsonConverter>
            {
                new Iso8601TimeSpanConverter(),
                new TransformationJsonConverter(),
            }
        };

        /// <summary>
        /// The serialization settings  used for Azure RP communication
        /// </summary>
        protected static readonly JsonSerializerSettings SerializationSettings = new JsonSerializerSettings
        {
            Formatting = Formatting.Indented,
            DateFormatHandling = DateFormatHandling.IsoDateFormat,
            DateTimeZoneHandling = DateTimeZoneHandling.Utc,
            NullValueHandling = NullValueHandling.Ignore,
            ReferenceLoopHandling = ReferenceLoopHandling.Serialize,
            ContractResolver = new ReadOnlyJsonContractResolver(),
            Converters = new List<JsonConverter>
            {
                new Iso8601TimeSpanConverter(),
                new TransformationJsonConverter()
            }
        };

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
        /// The header client request identifier.
        /// </summary>
        public const string HeaderClientRequestId = "x-ms-client-request-id";

        /// <summary>
        /// AzureResourceBroker constructor
        /// </summary>
        /// <param name="authenticationTokenBroker"></param>
        /// <param name="logger"></param>
        public AzResourceBroker(
            IHttpUtility httpUtility, 
            ILogger logger,
            IAzureAD azureAd)
        {
            this.httpUtility = httpUtility;
            this.logger = logger;
            this.azureAd = azureAd;
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
        protected async Task<T> PutResourceAsync<T>(
            string subscriptionId,
            string resourceId, 
            string apiVersion, 
            T requestBody) where T : class
        {
            var requestUri = GetResourceUri(resourceId.TrimEnd('/'), apiVersion);

            return await this.CallServerAndGetAsync<T, T>(
                subscriptionId: subscriptionId,
                httpMethod: HttpMethod.Put,
                requestUri: requestUri,
                requestBody: requestBody);
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
        protected async Task<TResponse> PostResourceActionAsync<TRequest, TResponse>(
            string subscriptionId,
            string resourceId, 
            string action, 
            string apiVersion, 
            TRequest request) where TResponse : class where TRequest : class
        {
            var requestUri = GetResourceActionUri(resourceId.TrimEnd('/'), action.Trim('/'), apiVersion);

            return await this.CallServerAndGetAsync<TRequest, TResponse>(subscriptionId: subscriptionId, httpMethod: HttpMethod.Post, requestUri: requestUri, requestBody: request);
        }

        /// <summary>
        /// Gets the ARM resource.
        /// </summary>
        /// <typeparam name="T">The type of response</typeparam>
        /// <param name="resourceId">The resource Id.</param>
        /// <param name="apiVersion">The API version.</param>
        /// <returns>The resource.</returns>
        protected async Task<T> GetResourceAsync<T>(string subscriptionId, string resourceId, string apiVersion) where T : class
        {
            Uri requestUri = GetResourceUri(resourceId.TrimEnd('/'), apiVersion);

            //Todo: Do we need TokenAccess for read operations?
            return await this.CallServerAndGetAsync<T, T>(subscriptionId: subscriptionId, httpMethod: HttpMethod.Get, requestUri: requestUri, requestBody: null);
        }

        /// <summary>
        /// Try gets the ARM resource. If calls failed with ResourceNotFound, then return null instead of exception.
        /// </summary>
        /// <typeparam name="T">The type of response</typeparam>
        /// <param name="resourceId">The resource Id.</param>
        /// <param name="apiVersion">The API version.</param>
        /// <returns>HttpResponseMessage.</returns>
        protected async Task<HttpResponseMessage> TryGetResourceAsync<T>(string subscriptionId, string resourceId, string apiVersion) where T : class
        {
            Uri requestUri = GetResourceUri(resourceId.TrimEnd('/'), apiVersion);

            //Todo: Do we need TokenAccess for read operations?
            return await this.CallServerAsync<T>(subscriptionId: subscriptionId, httpMethod: HttpMethod.Get, requestUri: requestUri, requestBody: null);
        }

        /// <summary>
        /// Delete resource internal method
        /// </summary>
        /// <param name="resourceId">The resource ID</param>
        /// <param name="apiVersion">The API version</param>
        /// <param name="headers">The HTTP headers.</param>
        protected async Task<T> DeleteResourceAsync<T>(string subscriptionId, string resourceId, string apiVersion) where T : class
        {
            Uri requestUri = GetResourceUri(resourceId.TrimEnd('/'), apiVersion);

            return await this.CallServerAndGetAsync<T, T>(subscriptionId: subscriptionId, httpMethod: HttpMethod.Delete, requestUri: requestUri, requestBody: null)
                .ConfigureAwait(continueOnCapturedContext: false);
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
        private async Task<TResponse> CallServerAndGetAsync<TRequest, TResponse>(
            string subscriptionId,
            HttpMethod httpMethod,
            Uri requestUri,
            TRequest requestBody) where TResponse : class where TRequest : class
        {
            string tenantId = await GetTenantIdFromSubscriptionIdAsync(subscriptionId);
            string accessToken = await azureAd.GetAccessTokenForArm(
                CurrentContext.ManagedIdentity,
                CurrentContext.ArmAccessIdentity,
                tenantId,
                CurrentContext.AzureEnvironment);

            if (String.IsNullOrEmpty(accessToken))
            {
                throw new Exception($"Invalid accesstoken for subscriptionId: {subscriptionId}, tenantId: {tenantId}");
            }

            using (var request = new HttpRequestMessage(httpMethod, requestUri))
            {
                PopulateRequestMessage(request, accessToken, requestBody);
                return await httpUtility.SendRequestAndGetJsonAsync<TResponse>(request, DeserializationSettings);
            }
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
        private async Task<HttpResponseMessage> CallServerAsync<TRequest>(
            string subscriptionId,
            HttpMethod httpMethod,
            Uri requestUri,
            TRequest requestBody) where TRequest : class
        {
            string tenantId = await GetTenantIdFromSubscriptionIdAsync(subscriptionId);
            string accessToken = await azureAd.GetAccessTokenForArm(
                CurrentContext.ManagedIdentity,
                CurrentContext.ArmAccessIdentity,
                tenantId,
                CurrentContext.AzureEnvironment);

            if (String.IsNullOrEmpty(accessToken))
            {
                throw new Exception($"Invalid accesstoken for subscriptionId: {subscriptionId}, tenantId: {tenantId}");
            }

            using (HttpRequestMessage request = new HttpRequestMessage(httpMethod, requestUri))
            {
                PopulateRequestMessage(request, accessToken, requestBody);
                return await httpUtility.SendAsync(request);
            }
        }

        private void PopulateRequestMessage<TRequest>(HttpRequestMessage request, string accessToken, TRequest requestBody) where TRequest : class
        {
            if (!string.IsNullOrWhiteSpace(accessToken))
            {
                request.Headers.Authorization = new AuthenticationHeaderValue(scheme: "Bearer", parameter: accessToken);
            }

            request.Headers.Add(name: HeaderClientRequestId, value: Guid.NewGuid().ToString());
            request.Headers.Add(name: "User-Agent", value: "cloud-console-rp/1.0");

            if (requestBody != null)
            {
                string json = JsonUtility.Serialize(requestBody, SerializationSettings);
                request.Content = new StringContent(json, Encoding.UTF8, "application/json");

                logger.LogInformation($"RequestMessage: { request}, Content: { json }");
            }
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

        private async Task<string> GetTenantIdFromSubscriptionIdAsync(string subscriptionId)
        {
            string tenantId = null;
            var httpRequestMessage = new HttpRequestMessage(
                HttpMethod.Get,
                string.Format("https://management.azure.com/subscriptions/{0}?api-version=2014-04-01", subscriptionId));
            HttpResponseMessage httpResonseMessage = await httpUtility.SendAsync(httpRequestMessage);

            // if the subscription is invalid (either a bad guid or a non-existant subscription), the status code will be a 404
            // meaning the tenantId will be returned as null
            if (httpResonseMessage.StatusCode == HttpStatusCode.Unauthorized)
            {
                if (httpResonseMessage.Headers.WwwAuthenticate.Count <= 0)
                {
                    throw new Exception(
                        $"No auth context found for authenticating to subscription {subscriptionId}");
                }

                var auth = httpResonseMessage.Headers.WwwAuthenticate.First();
                var param = auth.Parameter;
                try
                {
                    var authUrl = param.Split(',')[0].Split('=')[1];
                    tenantId = authUrl.Substring(authUrl.LastIndexOf('/') + 1, 36);
                }
                catch (Exception ex)
                {
                    throw new InvalidOperationException($"Error parsing tenantId from WwwAuthenticate header. Was trying to parse {param}", ex);
                }
            }
            else
            {
                throw new Exception($"Expected 401 response from unauthorized request to ARM, got {httpResonseMessage.StatusCode} instead.");
            }

            return tenantId;
        }
    }
}
