// -----------------------------------------------------------------------------
// <copyright file="TestAzureAD.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP.Unittests
{
    using System.Net.Http;
    using System.Threading.Tasks;
    using Microsoft.Azure.ContainerNetworking.CoreLib;

#pragma warning disable CS1998 // Async method lacks 'await' operators and will run synchronously. Test code.

    internal class TestAzureAD : IAzureAD
    {
        public async Task<string> GetAccessToken(
            ClientIdentity clientIdentity,
            string resource,
            string azureEnvironment)
        {
            return "testToken";
        }

        public async Task<string> GetAccessToken(
            ClientIdentity keyVaultClientIdentity,
            ClientIdentity azureADClientIdentity,
            string resource,
            string azureEnvironment)
        {
            return "testToken";
        }

        public async Task<string> GetAccessTokenForArm(
            ClientIdentity keyVaultClientIdentity,
            ClientIdentity azureADClientIdentity,
            string tenantId,
            string azureEnvironment)
        {
            return "testToken";
        }

        public async Task<string> GetAccessToken(
           string connectionString,
           string resource,
           string azureEnvironment)
        {
            return "testToken";
        }

        /// <summary>
        /// Get access token from Azure AD using a client identity that is stored
        /// in key vault. In this case, first a call will be made to KeyVault
        /// to fetch client identity and then using this fetched identity, call
        /// will be made to AzureAD to fetch token. To access key vault, an identity
        /// is needed in this case
        /// </summary>
        /// <param name="keyVaultClientIdentity"></param>
        /// <param name="azureADClientIdentity"></param>
        /// <param name="resource"></param>
        /// <param name="authorityUrl"></param>
        /// <param name="azureEnvironment"></param>
        /// <returns></returns>
        public async Task<string> GetAccessToken(
            ClientIdentity keyVaultClientIdentity,
            ClientIdentity azureADClientIdentity,
            string resource,
            string authorityUrl,
            string azureEnvironment)
        {
            return "testToken";
        }
    }
}
