// -----------------------------------------------------------------------------
// <copyright file="AzureAD.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using System;
    using System.Security.Cryptography.X509Certificates;
    using System.Threading.Tasks;
    using Microsoft.Azure.KeyVault;
    using Microsoft.Azure.KeyVault.Models;
    using Microsoft.Azure.Services.AppAuthentication;
    using Microsoft.Identity.Client;

    public class AzureAD : IAzureAD
    {
        /// <summary>
        /// Get access token from AzureAD using user managed identity
        /// </summary>
        /// <param name="clientIdentity">
        /// client identity that will be used to authenticate with AzureAD
        /// to fetch token
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
            if (clientIdentity == null)
            {
                throw new ArgumentNullException("clientIdentity");
            }

            string connStr = this.GetTokenProviderConnectionString(clientIdentity);
            return await GetAccessToken(connStr, resource, azureEnvironment).ConfigureAwait(false);
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
            if (keyVaultClientIdentity == null)
            {
                throw new ArgumentNullException("keyVaultlientIdentity");
            }

            if (azureADClientIdentity == null)
            {
                throw new ArgumentNullException("azureADClientIdentity");
            }

            if (string.IsNullOrWhiteSpace(resource))
            {
                throw new ArgumentNullException("resource");
            }

            if (string.IsNullOrWhiteSpace(authorityUrl))
            {
                throw new ArgumentNullException("authorityUrl");
            }

            if (string.IsNullOrWhiteSpace(azureEnvironment))
            {
                throw new ArgumentNullException("azureEnvironment");
            }

            // https://docs.microsoft.com/en-us/azure/key-vault/general/service-to-service-authentication#connection-string-support
            string kvConnString = this.GetTokenProviderConnectionString(keyVaultClientIdentity);
            X509Certificate2 cert = await this.FetchCertificateFromKV(
                kvConnString, azureADClientIdentity.KeyVaultCertPath, azureEnvironment).ConfigureAwait(false);

            Uri authority = new Uri(authorityUrl);
            IConfidentialClientApplication app = ConfidentialClientApplicationBuilder.Create(azureADClientIdentity.AppId)
                   .WithCertificate(cert)
                   .WithAuthority(authority)
                   .Build();

            string[] ResourceIds = new string[] { resource };

            AuthenticationResult result = await app.AcquireTokenForClient(ResourceIds).WithSendX5C(true).ExecuteAsync().ConfigureAwait(false);
            return result.AccessToken;
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
        /// <param name="azureEnvironment"></param>
        /// <returns></returns>
        public async Task<string> GetAccessToken(
            ClientIdentity keyVaultClientIdentity,
            ClientIdentity azureADClientIdentity,
            string resource,
            string azureEnvironment)
        {
            if (keyVaultClientIdentity == null)
            {
                throw new ArgumentNullException("keyVaultlientIdentity");
            }

            if (azureADClientIdentity == null)
            {
                throw new ArgumentNullException("azureADClientIdentity");
            }

            if (string.IsNullOrWhiteSpace(azureADClientIdentity.TenantId))
            {
                throw new ArgumentNullException("azureADClientIdentity.TenantId");
            }

            if (string.IsNullOrWhiteSpace(resource))
            {
                throw new ArgumentNullException("resource");
            }

            if (string.IsNullOrWhiteSpace(azureEnvironment))
            {
                throw new ArgumentNullException("azureEnvironment");
            }

            return await GetAccessToken(
                keyVaultClientIdentity,
                azureADClientIdentity,
                resource,
                $"{AzureInfo.GetAzureADInstance(azureEnvironment, true)}{azureADClientIdentity.TenantId}",
                azureEnvironment);
        }

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
        public async Task<string> GetAccessTokenForArm(
            ClientIdentity keyVaultClientIdentity,
            ClientIdentity azureADClientIdentity,
            string tenantId,
            string azureEnvironment)
        {
            if (keyVaultClientIdentity == null)
            {
                throw new ArgumentNullException("keyVaultlientIdentity");
            }

            if (azureADClientIdentity == null)
            {
                throw new ArgumentNullException("azureADClientIdentity");
            }

            if (string.IsNullOrWhiteSpace(tenantId))
            {
                throw new ArgumentNullException("tenantId");
            }

            if (string.IsNullOrWhiteSpace(azureEnvironment))
            {
                throw new ArgumentNullException("azureEnvironment");
            }

            string resource = $"{AzureInfo.GetArmEndpoint(azureEnvironment)}//.default";

            return await GetAccessToken(
               keyVaultClientIdentity,
               azureADClientIdentity,
               resource,
               $"{AzureInfo.GetAzureADInstance(azureEnvironment, true)}{tenantId}",
               azureEnvironment).ConfigureAwait(false);
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
            if (string.IsNullOrWhiteSpace(connectionString))
            {
                throw new ArgumentNullException("connectionString");
            }

            if (string.IsNullOrWhiteSpace(resource))
            {
                throw new ArgumentNullException("resource");
            }

            if (string.IsNullOrWhiteSpace(azureEnvironment))
            {
                throw new ArgumentNullException("azureEnvironment");
            }

            // https://docs.microsoft.com/en-us/azure/key-vault/general/service-to-service-authentication#connection-string-support
            AzureServiceTokenProvider azureServiceTokenProvider = new AzureServiceTokenProvider(
                connectionString,
                AzureInfo.GetAzureADInstance(azureEnvironment, true));

            return await azureServiceTokenProvider.GetAccessTokenAsync(resource).ConfigureAwait(false);
        }

        /// <summary>
        /// Get connection string for a given authentication type for AzureServiceTokenProvider class
        /// </summary>
        /// <param name="clientIdentity"></param>
        /// <returns></returns>
        private string GetTokenProviderConnectionString(ClientIdentity clientIdentity)
        {
            string connectionString = null;
            if (clientIdentity.IdentityType == ClientIdentityType.ManagedIdentity)
            {
                connectionString = string.IsNullOrWhiteSpace(clientIdentity.AppId)
                                   ? TokenProviderConnectionStringFormats.ManagedIdentityFormat
                                   : string.Format(TokenProviderConnectionStringFormats.ManagedIdentityClientIdFormat, clientIdentity.AppId);
            }
            else if (clientIdentity.IdentityType == ClientIdentityType.SpClientSecret)
            {
                if (string.IsNullOrWhiteSpace(clientIdentity.AppId) ||
                    string.IsNullOrWhiteSpace(clientIdentity.AppKey))
                {
                    throw new ArgumentException(
                        $"For identity type '{ClientIdentityType.SpClientSecret}' AppId,AppKey are required in clientIdentity");
                }

                connectionString = string.Format(TokenProviderConnectionStringFormats.ServicePrincipalAppKeyFormat,
                    clientIdentity.AppId, clientIdentity.TenantId, clientIdentity.AppKey);
            }
            else
            {
                throw new NotSupportedException($"identity type {clientIdentity.IdentityType} is not supported yet");
            }

            return connectionString;
        }

        private void ValidateADClientIdentityForKVScenario(ClientIdentity clientIdentity, string paramName)
        {
            //TODO:  For now only certificate is supported. Later on client secret can be added too
            if (clientIdentity.IdentityType == ClientIdentityType.SpClientCertificate)
            {
                if (string.IsNullOrWhiteSpace(clientIdentity.AppId) ||
                   string.IsNullOrWhiteSpace(clientIdentity.TenantId) ||
                   string.IsNullOrWhiteSpace(clientIdentity.KeyVaultCertPath))
                {
                    throw new ArgumentException(
                        $"For identity type '{ClientIdentityType.SpClientCertificate}' AppId,TenantId,KeyVaultCertPath are required in {paramName}");
                }
            }
            else
            {
                throw new NotSupportedException($"Identity type  {clientIdentity.IdentityType} not supported when identity is stored in key vault");
            }
        }

        /// <summary>
        /// Fetch certificate from a key vault
        /// </summary>
        /// <param name="ConnString"></param>
        /// <param name="secretIdentifier"></param>
        /// <returns></returns>
        private async Task<X509Certificate2> FetchCertificateFromKV(string ConnString, string secretIdentifier, string azureEnvironment)
        {
            return await AsyncRetry.Retry<X509Certificate2>(async () =>
           {
               // Instantiate a new KeyVaultClient object, with an access token to Key Vault
               // https://docs.microsoft.com/en-us/azure/key-vault/general/service-to-service-authentication#connection-string-support
               AzureServiceTokenProvider azureServiceTokenProvider = new AzureServiceTokenProvider(
                  ConnString, AzureInfo.GetAzureADInstance(azureEnvironment, true));

               KeyVaultClient client = new KeyVaultClient(new KeyVaultClient.AuthenticationCallback(azureServiceTokenProvider.KeyVaultTokenCallback));

               SecretBundle secret = await client.GetSecretAsync(secretIdentifier).ConfigureAwait(false);

               byte[] certBytes = Convert.FromBase64String(secret.Value);
               X509Certificate2 sslCert = new X509Certificate2(certBytes);

               return sslCert;
           },
           3, TimeSpan.FromSeconds(10), TimeSpan.FromMinutes(5), (e) => true).ConfigureAwait(false);
        }
    }
}
