// --------------------------------------------------------------------------------------------------------------------
// <copyright file="ResourceConstants.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------


namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    public class ResourceConstants
    {
        public const string SubscriptionId = "SubscriptionId";
        public const string ResourceGroup = "ResourceGroup";
        public const string ProviderType = "ProviderType";
        public const string ResourceName = "ResourceName";

        public const string IdentityUrlHeader = "x-ms-identity-url";
        public const string IdentityPrincipalIdHeader = "x-ms-identity-principal-id";
        public const string IdentityHomeTenantIdHeader = "x-ms-home-tenant-id";
        public const string IdentityClientTenantIdHeader = "x-ms-client-tenant-id";
        public const string IdentityUrlApiVersion = "api-version=2015-08-31-PREVIEW";
    }
}
