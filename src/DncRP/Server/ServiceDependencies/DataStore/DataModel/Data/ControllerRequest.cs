// -----------------------------------------------------------------------------
// <copyright file="ControllerRequest.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Text.Json.Serialization;

    public enum ControllerType
    {
        Unknown = 0,
        Kubernetes
    }

    public class ControllerRequestProperties
    {
        [JsonPropertyName("serverAppID")]
        public string ServerAppID
        {
            get; set;
        }

        [JsonPropertyName("serverTenantID")]
        public string ServerTenantID
        {
            get; set;
        }

        [JsonPropertyName("clusterRootCA")]
        public string ClusterRootCA
        {
            get; set;
        }

        [JsonPropertyName("apiServerEndpoint")]
        public string ApiServerEndpoint
        {
            get; set;
        }
    }

    public class ControllerRequestParameters
    {
        [JsonPropertyName("controllerType")]
        public ControllerType Type
        {
            get; set;
        }

        [JsonPropertyName("properties")]
        public ControllerRequestProperties Properties
        {
            get; set;
        }
    }

    public class ControllerRequest
    {
        [JsonPropertyName("controllerParameters")]
        public ControllerRequestParameters Parameters
        {
            get; set;
        }
    }
}
