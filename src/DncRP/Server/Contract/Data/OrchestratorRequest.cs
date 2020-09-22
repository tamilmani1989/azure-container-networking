// -----------------------------------------------------------------------------
// <copyright file="OrchestratorRequest.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;

    public class OrchestratorRequestParameters
    {
        [JsonProperty("location")]
        public string Location
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
    }
}
