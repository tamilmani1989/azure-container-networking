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
    }
}
