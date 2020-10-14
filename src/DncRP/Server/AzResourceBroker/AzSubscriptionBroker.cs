// -----------------------------------------------------------------------------
// <copyright file="AzSubscriptionBroker.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.IO;
    using System.Linq;
    using System.Net;
    using System.Net.Http;
    using System.Threading.Tasks;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Logging;
    using NetworkModel = Microsoft.Azure.Management.Network.Models;

    /// <summary>
    /// The Azure VNet Broker to get or update VNet resource.
    /// </summary>
    public class AzSubscriptionBroker : AzResourceBroker
    {
        /// <summary>
        /// The API version for azure VNet operations.
        /// </summary>
        private const string ApiVersion = "2014-04-01";

        /// <summary>
        /// Initializes a new instance AzureVNetBroker class.
        /// </summary>
        /// <param name="httpUtility">HTTP Utility</param>
        /// <param name="logger">Logger</param>
        public AzSubscriptionBroker(
            IHttpUtility httpUtility,
            ILogger logger)
            : base(httpUtility, logger)
        {
            this.logger = logger;
        }

        /// <summary>
        /// Get Tenant info for a given Subscription
        /// </summary>
        /// <param name="subscriptionId"></param>
        /// <returns></returns>
        public async Task<string> GetTenantIdFromSubscriptionIdAsync(string subscriptionId)
        {
            string tenantId = null;
            var requestUri = new Uri($"{ArmEndpoint}/subscriptions/{subscriptionId}?api-version={ApiVersion}");

            var httpRequestMessage = new HttpRequestMessage(
                HttpMethod.Get,
                requestUri);
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

            logger.CLogInformation($"SubscriptionId: {subscriptionId}, TenantId {tenantId}");
            return tenantId;
        }
    }
}
