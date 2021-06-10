// --------------------------------------------------------------------------------------------------------------------
// <copyright file="OrchestratorProperties.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;

    public class OrchestratorProperties : TopLevelResourceProperties
    {
        [JsonProperty("orchestratorAppId")]
        public string AppId
        {
            get; set;
        }

        [JsonProperty("orchestratorTenantId")]
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

        [JsonProperty("privateLinkId")]
        public string PrivateLinkId
        {
            get; set;
        }

        [JsonProperty("controllerDetails")]
        public ControllerDetails ControllerDetails
        {
            get; set;
        }
    }

    public class OrchestratorPropertiesInternal
    {
        [JsonProperty("orchestratorAppId")]
        public string AppId
        {
            get; set;
        }

        [JsonProperty("orchestratorTenantId")]
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

        [JsonProperty("privateLinkId")]
        public string PrivateLinkId
        {
            get; set;
        }

        [JsonProperty("privateEndpointName")]
        public string PrivateEndpointName
        {
            get; set;
        }

        [JsonProperty("clientID")]
        public string ClientId
        {
            get; set;
        }

        [JsonProperty("clientSecretName")]
        public string ClientSecretName
        {
            get; set;
        }
    }
}
