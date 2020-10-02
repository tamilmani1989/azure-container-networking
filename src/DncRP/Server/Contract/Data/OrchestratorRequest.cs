// -----------------------------------------------------------------------------
// <copyright file="OrchestratorRequest.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;

    public enum OrchestratorKind
    {
        Unknown = 0,
        Kubernetes
    }

    public class OrchestratorIdentityParameter
    {
        [JsonProperty("type")]
        public MSIIdentityType IdentityType
        {
            get; set;
        }
    }

    public class OrchestratorRequestParameters
    {
        [JsonProperty("location")]
        public string Location
        {
            get; set;
        }

        [JsonProperty("kind")]
        public OrchestratorKind Kind
        {
            get; set;
        }

        [JsonProperty("identity")]
        public OrchestratorIdentityParameter IdentityParameter
        {
            get; set;
        }

        [JsonProperty("properties")]
        public OrchestratorProperties Properties
        {
            get; set;
        }
    }

    public class OrchestratorArmProperties
    {
        [JsonProperty("parameters")]
        public OrchestratorRequestParameters Parameters
        {
            get; set;
        }
    }

    public class OrchestratorRequest
    {
        [JsonProperty("properties")]
        public OrchestratorArmProperties Properties
        {
            get; set;
        }

        [JsonIgnore]
        public MSIIdentityMetadata Identity
        {
            get; set;
        }
    }
}
