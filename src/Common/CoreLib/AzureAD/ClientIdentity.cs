// -----------------------------------------------------------------------------
// <copyright file="ClientIdentity.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    /// <summary>
    /// Client identity type
    /// </summary>
    public enum ClientIdentityType
    {
        Unknown = 0,
        ManagedIdentity,
        SpClientSecret,
        SpClientCertificate
    }

    /// <summary>
    /// Client identity tag. Basically a value to just recognize
    /// what this identity will be used for or what it is. Value can be
    /// any user friendly name that helps us remember purpose of identity
    /// </summary>
    public enum ClientIdentityTags
    {
        Unknown = 0,
        ManagedIdentity,
        AzureADArmToken
    }

    /// <summary>
    /// Identity of client to request token from AzureAD
    /// For ManagedIdentity -
    ///     No parameter is mandatory for managed. App ID can be passed in and is recommended but is optional.
    /// For ServicePrincipalAppKey -
    ///     AppId, TenantId, AppKey is needed. Only exception is when key is fetched from KeyVault.
    ///     In that case AppKey won't be validated.
    /// For ServicePrincipalClientCertificate -
    ///     AppId, TenantId, KeyVaultCertPath is needed. Certificate has to be stored in KeyVault
    /// </summary>
    public class ClientIdentity
    {
        public ClientIdentityType IdentityType
        {
            get; set;
        }

        public string AppId
        {
            get; set;
        }

        public string TenantId
        {
            get; set;
        }

        public string AppKey
        {
            get; set;
        }

        public string KeyVaultCertPath
        {
            get; set;
        }

        public ClientIdentityTags Tag
        {
            get; set;
        }
    }
}
