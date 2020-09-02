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
        Task<string> GetAccessTokenWithUserManagedIdentity(string userManagedClientId, string resource, string azureEnvironment);

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
