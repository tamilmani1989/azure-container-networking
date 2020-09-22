// -----------------------------------------------------------------------------
// <copyright file="ControllerRequest.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;

    public class ControllerRequestParameters
    {
        [JsonProperty("location")]
        public string Location
        {
            get; set;
        }
    }

    public class ArmProperties
    {
        [JsonProperty("parameters")]
        public ControllerRequestParameters Parameters
        {
            get; set;
        }
    }

    public class ControllerRequest
    {
        [JsonProperty("properties")]
        public ArmProperties Properties
        {
            get; set;
        }
    }

}
