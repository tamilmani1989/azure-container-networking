// -----------------------------------------------------------------------------
// <copyright file="DatastoreSettings.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Net;
    using System.Net.Http;
    using System.Net.Http.Headers;
    using System.Security.Cryptography;
    using System.Text;
    using System.Threading;
    using System.Threading.Tasks;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Azure.KeyVault;

    public static class ExtensionMethods
    {
        public static DatastoreSettings FindByRegion(this DatastoreSettings[] list, string region)
        {
            if (list == null || region == null)
            {
                return null;
            }

            foreach (DatastoreSettings s in list)
            {
                if (s.Region.Equals(region, StringComparison.OrdinalIgnoreCase))
                {
                    return s;
                }
            }

            return null;
        }

        // TODO: workitem: 10093300
        public static int GetDeterministicHashCode(string str)
        {
            unchecked
            {
                int hash1 = (5381 << 16) + 5381;
                int hash2 = hash1;

                for (int i = 0; i < str.Length && str[i] != '\0'; i += 2)
                {
                    hash1 = ((hash1 << 5) + hash1) ^ str[i];
                    if (i == str.Length - 1 || str[i + 1] == '\0')
                        break;
                    hash2 = ((hash2 << 5) + hash2) ^ str[i + 1];
                }

                return hash1 + (hash2 * 1566083941);
            }
        }

        public static async Task<MSIIdentity> GetMSIIdentity(IHttpUtility httpUtility, MSIIdentityMetadata identity)
        {
            HttpRequestMessage requestMessage = new HttpRequestMessage
            {
                Method = HttpMethod.Get,
                RequestUri = new Uri(identity.Url),
            };

            HttpResponseMessage response = await httpUtility.SendRequestAndGetHttpResponseAsync(
                requestMessage, HttpCompletionOption.ResponseHeadersRead, CancellationToken.None);

            if (response.StatusCode == HttpStatusCode.Unauthorized)
            {
                var authenticateHeaders = response.Headers.WwwAuthenticate;

                foreach (var authenticateHeader in authenticateHeaders)
                {
                    // Sample authentication header - Bearer authorization = "https://login.windows-ppe.net/5D929AE3-B37C-46AA-A3C8-C1558902F101"
                    // This check basically checks that header starts with keyword Bearer
                    if (HttpBearerChallenge.IsBearerChallenge(authenticateHeader.ToString()))
                    {
                        var challenge = new HttpBearerChallenge(requestMessage.RequestUri, authenticateHeader.ToString());
                        Console.WriteLine("[DNC RP] AuthorizationServer: {0}", challenge.AuthorizationServer);
                        Console.WriteLine("[DNC RP] Resource: {0}", challenge.Resource);
                        Console.WriteLine("[DNC RP] Scope: {0}", challenge.Scope);

                        string resource = $"{challenge.Resource}/.default";
                        IAzureAD azureAD = new AzureAD();
                        string token = await azureAD.GetAccessToken(
                            CurrentContext.ManagedIdentity,
                            CurrentContext.ArmAccessIdentity,
                            resource,
                            challenge.AuthorizationServer,
                            CurrentContext.AzureEnvironment).ConfigureAwait(false);

                        requestMessage = new HttpRequestMessage
                        {
                            Method = HttpMethod.Get,
                            RequestUri = new Uri($"{identity.Url}?{ResourceConstants.IdentityUrlApiVersion}"),
                        };
                        requestMessage.Headers.Authorization = new AuthenticationHeaderValue("Bearer", token);

                        return await httpUtility.SendRequestAndGetJsonAsync<MSIIdentity>(
                            requestMessage, HttpCompletionOption.ResponseHeadersRead, CancellationToken.None);
                    }
                }
            }

            string msg = string.Format("[DNC RP] GetOrchestratorIdentity: Unexpected response code: {0}. {1}",
                response.StatusCode,
                response.Content != null ? response.Content.ToString() : string.Empty);
            Console.WriteLine(msg);

            throw new Exception(msg);
        }
    }
}
