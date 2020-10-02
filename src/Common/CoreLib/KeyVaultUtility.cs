// -----------------------------------------------------------------------------
// <copyright file="KeyVaultUtility.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using System;
    using global::Azure.Core;
    using global::Azure.Identity;
    using global::Azure.Security.KeyVault.Secrets;

    public class KeyVaultUtility : IKeyVaultUtility
    {
        public KeyVaultSecret SetSecret(string keyVaultUrl, ClientIdentity identity, string secretName, string secretValue)
        {
            TokenCredential credential;

            if (identity.IdentityType == ClientIdentityType.ManagedIdentity)
            {
                credential = new ManagedIdentityCredential(identity.AppId);
            }
            else if (identity.IdentityType == ClientIdentityType.SpClientSecret)
            {
                credential = new ClientSecretCredential(identity.TenantId, identity.AppId, identity.AppKey);
            }
            else
            {
                throw new NotSupportedException($"Identity type {identity.IdentityType} not supported yet");
            }

            SecretClient client = new SecretClient(new Uri(keyVaultUrl), credential);
            KeyVaultSecret secret = new KeyVaultSecret(secretName, secretValue);
            return client.SetSecret(secret);
        }

        public KeyVaultSecret GetSecret(string keyVaultUrl, ClientIdentity identity, string secretName, string version)
        {
            TokenCredential credential;

            if (identity.IdentityType == ClientIdentityType.ManagedIdentity)
            {
                credential = new ManagedIdentityCredential(identity.AppId);
            }
            else if (identity.IdentityType == ClientIdentityType.SpClientSecret)
            {
                credential = new ClientSecretCredential(identity.TenantId, identity.AppId, identity.AppKey);
            }
            else
            {
                throw new NotSupportedException($"Identity type {identity.IdentityType} not supported yet");
            }

            SecretClient client = new SecretClient(new Uri(keyVaultUrl), credential);
            return client.GetSecret(secretName, version);
        }

        public DeleteSecretOperation DeleteSecret(string keyVaultUrl, ClientIdentity identity, string secretName)
        {
            TokenCredential credential;

            if (identity.IdentityType == ClientIdentityType.ManagedIdentity)
            {
                credential = new ManagedIdentityCredential(identity.AppId);
            }
            else if (identity.IdentityType == ClientIdentityType.SpClientSecret)
            {
                credential = new ClientSecretCredential(identity.TenantId, identity.AppId, identity.AppKey);
            }
            else
            {
                throw new NotSupportedException($"Identity type {identity.IdentityType} not supported yet");
            }

            SecretClient client = new SecretClient(new Uri(keyVaultUrl), credential);
            return client.StartDeleteSecret(secretName);
        }
    }
}
