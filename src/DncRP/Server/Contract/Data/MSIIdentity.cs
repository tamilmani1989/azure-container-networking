// -----------------------------------------------------------------------------
// <copyright file="OrchestratorIdentity.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using Newtonsoft.Json;

    public enum MSIIdentityType
    {
        Unknown = 0,
        SystemAssigned
    }

    public class MSIIdentity
    {
        [JsonProperty("client_id")]
        public string ClientId
        {
            get; set;
        }

        [JsonProperty("client_secret")]
        public string ClientSecret
        {
            get; set;
        }

        [JsonProperty("client_secret_url")]
        public string ClientSecretUrl
        {
            get; set;
        }

        [JsonProperty("not_before")]
        public DateTime NotBefore
        {
            get; set;
        }

        [JsonProperty("not_after")]
        public DateTime NotAfter
        {
            get; set;
        }

        [JsonProperty("renew_after")]
        public DateTime RenewAfter
        {
            get; set;
        }

        [JsonProperty("cannot_renew_after")]
        public DateTime CannotRenewAfter
        {
            get; set;
        }
    }

    public class MSIIdentityMetadata
    {
        [JsonProperty("Url")]
        public string Url
        {
            get; set;
        }

        [JsonProperty("PrincipalId")]
        public string PrincipalId
        {
            get; set;
        }

        [JsonProperty("HomeTenantId")]
        public string HomeTenantId
        {
            get; set;
        }

        [JsonProperty("ClientTenantId")]
        public string ClientTenantId
        {
            get; set;
        }

        [JsonProperty("type")]
        public MSIIdentityType Type
        {
            get; set;
        }
    }

    public class MSIIdentityResponse
    {
        [JsonProperty("principalId")]
        public string PrincipalId
        {
            get; set;
        }

        [JsonProperty("tenantId")]
        public string TenantId
        {
            get; set;
        }

        [JsonProperty("type")]
        public MSIIdentityType Type
        {
            get; set;
        }
    }
}
