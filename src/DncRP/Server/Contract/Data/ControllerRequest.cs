// -----------------------------------------------------------------------------
// <copyright file="ControllerRequest.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;

    public enum ControllerType
    {
        Unknown = 0,
        Kubernetes
    }

    public class ControllerRequestProperties
    {
        [JsonProperty("serverAppID")]
        public string ServerAppID
        {
            get; set;
        }

        [JsonProperty("serverTenantID")]
        public string ServerTenantID
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

    public class ControllerRequestParameters
    {
        [JsonProperty("controllerType")]
        public ControllerType Type
        {
            get; set;
        }

        [JsonProperty("properties")]
        public ControllerRequestProperties Properties
        {
            get; set;
        }
    }

    public class ControllerRequest
    {
        [JsonProperty("controllerParameters")]
        public ControllerRequestParameters Parameters
        {
            get; set;
        }
    }
}
