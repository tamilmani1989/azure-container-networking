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
        Task<string> GetAccessToken(
            ClientIdentity keyVaultClientIdentity,
            ClientIdentity azureADClientIdentity,
            string resource,
            string authorityUrl,
            string azureEnvironment);

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
        /// <param name="azureEnvironment"></param>
        /// <returns></returns>
        Task<string> GetAccessToken(
            ClientIdentity keyVaultClientIdentity,
            ClientIdentity azureADClientIdentity,
            string resource,
            string azureEnvironment);

        /// <summary>
        /// Get access token for ARM resource from Azure AD using a client identity that is stored
        /// in key vault. In this case, first a call will be made to KeyVault
        /// to fetch client identity and then using this fetched identity, call
        /// will be made to AzureAD to fetch token. To access key vault, an identity
        /// is needed in this case
        /// </summary>
        /// <param name="keyVaultClientIdentity"></param>
        /// <param name="azureADClientIdentity"></param>
        /// <param name="tenantId"></param>
        /// <param name="azureEnvironment"></param>
        /// <returns></returns>
        Task<string> GetAccessTokenForArm(
            ClientIdentity keyVaultClientIdentity,
            ClientIdentity azureADClientIdentity,
            string tenantId,
            string azureEnvironment);
    }
}
