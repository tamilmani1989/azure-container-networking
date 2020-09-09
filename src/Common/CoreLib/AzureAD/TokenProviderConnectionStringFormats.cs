// -----------------------------------------------------------------------------
// <copyright file="TokenProviderConnectionStringFormats.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    /// <summary>
    /// CosmosDb Url formats
    /// </summary>
    public static class TokenProviderConnectionStringFormats
    {
        /// <summary>
        /// Managed Identity with app or client id format
        /// </summary>
        public const string ManagedIdentityClientIdFormat = "RunAs=App;AppId={0}";

        public const string ManagedIdentityFormat = "RunAs=App";

        public const string ServicePrincipalAppKeyFormat = "RunAs=App;AppId={0};TenantId={1};AppKey={2}";
    }
}
