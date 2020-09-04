// -----------------------------------------------------------------------------
// <copyright file="IAzureAD.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using System.Threading.Tasks;

    public interface IAzureAD
    {
        /// <summary>
        /// Get access token from AzureAD using user managed identity
        /// </summary>
        /// <param name="clientIdentity">
        /// Client Identity. If managed identity is being used, pass in ONLY AppId
        /// and that is enough. If service principal with key is being used, pass in
        /// AppId, TenantId and AppKey. Note - For a user assigned managed identity
        /// client id is the AppId.
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
        Task<string> GetAccessToken(
            ClientIdentity clientIdentity, string resource, string azureEnvironment);

        /// <summary>
        /// Get access token from AzureAD with given connection string
        /// </summary>
        /// <param name="connectionString">
        /// <param name="resource">
        /// resource for which token is being requested. For example for ARM -
        /// resource value would be "https://management.azure.com/"
        /// </param>
        /// <param name="azureEnvironment">
        /// Valid values are - azurecloud,azurechinacloud,azureusgovernment,azuregermancloud
        /// Default is azurecloud
        /// </param>
        /// <returns></returns>
        Task<string> GetAccessToken(string connectionString, string resource, string azureEnvironment);
    }
}
