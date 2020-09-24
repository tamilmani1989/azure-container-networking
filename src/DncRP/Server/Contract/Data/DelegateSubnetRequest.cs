// --------------------------------------------------------------------------------------------------------------------
// <copyright file="DelegateRequestParameters.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;
    using System;
    using System.IO;

    public class DelegateSubnetRequestProperties
    {
        [JsonProperty("subnetDetails")]
        public SubnetDetails SubnetDetails { get; set; }

        [JsonProperty("controllerDetails")]
        public ControllerDetails ControllerDetails { get; set; }
    }

    public class DelegateRequestParameters
    {
        [JsonProperty("location")]
        public string Location { get; set; }

        [JsonProperty("properties")]
        public DelegateSubnetRequestProperties Properties { get; set; }
    }

    public class DeletagetSubnetArmProperties
    {
        [JsonProperty("parameters")]
        public DelegateRequestParameters Parameters { get; set;  }
    }

    public class DelegateSubnetRequest
    {
        [JsonProperty("properties")]
        public DeletagetSubnetArmProperties Properties
        {
            get; set;
        }
    }
}
