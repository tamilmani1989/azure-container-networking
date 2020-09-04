// -----------------------------------------------------------------------------
// <copyright file="AzureAD.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using System;
    using System.Text;
    using System.Threading.Tasks;
    using Microsoft.Azure.Services.AppAuthentication;

    public class AzureAD : IAzureAD
    {
        /// <summary>
        /// Get access token from AzureAD using user managed identity
        /// </summary>
        /// <param name="userManagedClientId">
        /// Client ID of user managed identity. A managed identity can have
        /// object id as well as client id. This function takes client id.
        /// Both client/object ID are guids
        /// </param>
        /// <param name="resource">
        /// resource for which token is being requested. For example for ARM -
        /// resource value would be "https://management.azure.com/"
        /// </param>
        /// <param name="azureEnvironment">
        /// Valid values are - azurecloud,azurechinacloud,azureusgovernment,azuregermancloud
        /// Default is azurecloud
        /// </param>
        /// <returns></returns>
        public async Task<string> GetAccessToken(
            ClientIdentity clientIdentity,
            string resource,
            string azureEnvironment)
        {
            if (string.IsNullOrWhiteSpace(clientIdentity.AppId))
            {
                throw new ArgumentException("AppId is null in clientIdentity");
            }

            StringBuilder sb = new StringBuilder($"RunAs=App;AppId={clientIdentity.AppId}");
            if (!string.IsNullOrWhiteSpace(clientIdentity.TenantId) && !string.IsNullOrWhiteSpace(clientIdentity.AppKey))
            {
                sb.Append($";TenantId={clientIdentity.TenantId};AppKey={clientIdentity.AppKey}");
            }

            return await GetAccessToken(sb.ToString(), resource, azureEnvironment);
        }

        /// <summary>
        /// Get access token
        /// </summary>
        /// <param name="connectionString"></param>
        /// <param name="resource"></param>
        /// <param name="azureEnvironment"></param>
        /// <returns></returns>
        public async Task<string> GetAccessToken(
           string connectionString,
           string resource,
           string azureEnvironment)
        {
            // https://docs.microsoft.com/en-us/azure/key-vault/general/service-to-service-authentication#connection-string-support
            AzureServiceTokenProvider azureServiceTokenProvider = new AzureServiceTokenProvider(
                connectionString,
                AzureInfo.GetAzureADInstance(azureEnvironment, true));

            return await azureServiceTokenProvider.GetAccessTokenAsync(resource);
        }
    }
}
