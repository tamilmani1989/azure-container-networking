// --------------------------------------------------------------------------------------------------------------------
// <copyright file="DelegateSubnetProperties.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;

    public class DelegateSubnetProperties : TopLevelResourceProperties
    {
        [JsonProperty("subnetDetails")]
        public SubnetDetails SubnetDetails { get; set; }

        [JsonProperty("controllerDetails")]
        public ControllerDetails ControllerDetails { get; set; }
    }
}
