// -----------------------------------------------------------------------------
// <copyright file="IImdsRetriever.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using global::Azure.Security.KeyVault.Secrets;

    public interface IKeyVaultUtility
    {
        KeyVaultSecret SetSecret(string keyVaultUrl, ClientIdentity identity, string secretName, string secretValue);

        KeyVaultSecret GetSecret(string keyVaultUrl, ClientIdentity identity, string secretName, string version);

        DeleteSecretOperation DeleteSecret(string keyVaultUrl, ClientIdentity identity, string secretName);
    }
}
