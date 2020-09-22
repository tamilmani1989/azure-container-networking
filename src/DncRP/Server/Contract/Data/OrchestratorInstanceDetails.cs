// -----------------------------------------------------------------------------
// <copyright file="OrchestratorInstanceDetails.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;

    public enum OrchestratorType
    {
        Unknown = 0,
        Kubernetes
    }

    public class OrchestratorInstanceProperties
    {
        [JsonProperty("orchestratorAppID")]
        public string AppId
        {
            get; set;
        }

        [JsonProperty("orchestratorTenantID")]
        public string TenantId
        {
            get; set;
        }

        [JsonProperty("clusterRootCA")]
        public string ClusterRootCA
        {
            get; set;
        }

        [JsonProperty("apiServerEndpoint")]
        public string ApiServerEndpoint
        {
            get; set;
        }
    }

    public class OrchestratorInstanceDetails
    {
        [JsonProperty("orchestratorType")]
        public OrchestratorType Type
        {
            get; set;
        }

        [JsonProperty("properties")]
        public OrchestratorInstanceProperties Properties
        {
            get; set;
        }
    }
}
