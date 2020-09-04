// -----------------------------------------------------------------------------
// <copyright file="ArmAuthTokenBroker.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Linq;
    using System.Net;
    using System.Net.Http;
    using System.Threading.Tasks;

    ///********* Vipin is implementing this in AzureAD, this will be removed in his PR
    /// Keep this as commented if he needs reference for GetTenantIdForSubscription API

    /// <summary>
    /// 
    /// </summary>
    //public class ArmAuthTokenBroker
    //{
    //    // todo - Create a Singleton HTTPClient per service instead of per module
    //    private static HttpClient httpClient = new HttpClient();

    //    /// <summary>
    //    /// Gets the tenant id from subscription id.
    //    /// TODO: Move this to common file
    //    /// </summary>
    //    /// <param name="subscriptionId">The subscription identifier.</param>
    //    /// <returns>The tenant identifier.</returns>
    //    public static string GetTenantIdFromSubscriptionId(string subscriptionId)
    //    {
    //        string tenantId = null; // also known as directoryId
    //        //TODO: Is ArmEndpoint same for National clouds as well?
    //        var httpRequestMessage = new HttpRequestMessage(
    //            HttpMethod.Get,
    //            string.Format("https://management.azure.com/subscriptions/{0}?api-version=2014-04-01", subscriptionId));
    //        var response = httpClient.SendAsync(httpRequestMessage).Result;

    //        // if the subscription is invalid (either a bad guid or a non-existant subscription), the status code will be a 404
    //        // meaning the tenantId will be returned as null
    //        if (response.StatusCode == HttpStatusCode.Unauthorized)
    //        {
    //            if (response.Headers.WwwAuthenticate.Count <= 0)
    //            {
    //                throw new Exception(
    //                    $"No auth context found for authenticating to subscription {subscriptionId}");
    //            }

    //            var auth = response.Headers.WwwAuthenticate.First();
    //            var param = auth.Parameter;
    //            try
    //            {
    //                var authUrl = param.Split(',')[0].Split('=')[1];
    //                tenantId = authUrl.Substring(authUrl.LastIndexOf('/') + 1, 36);
    //            }
    //            catch (Exception ex)
    //            {
    //                throw new InvalidOperationException($"Error parsing tenantId from WwwAuthenticate header. Was trying to parse {param}", ex);
    //            }
    //        }
    //        else
    //        {
    //            throw new Exception($"Expected 401 response from unauthorized request to ARM, got {response.StatusCode} instead.");
    //        }

    //        return tenantId;
    //    }

    //    /// <summary>
    //    /// Gets the authentication token for the application Id under the scope of the given tenant.
    //    /// </summary>
    //    /// <param name="tenantId">The tenant to authenticate against.</param>
    //    public static Task<string> GetAuthenticationToken(string tenantId)
    //    {
    //        // Todo Implement AuthToken
    //        throw new NotImplementedException();
    //    }
    //}
}
