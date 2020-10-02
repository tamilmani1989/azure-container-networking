// --------------------------------------------------------------------------------------------------------------------
// <copyright file="DelegateSubnet.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP.ServiceTests
{
    using Newtonsoft.Json;

    public class DelegateSubnetResponseProperties
    {
        /// <summary>
        /// Gets or sets the type of the resource.
        /// </summary>
        [JsonProperty(Required = Required.Default, PropertyName = "resourceGuid")]
        public string ResourceGuid { get; set; }

        /// <summary>
        /// Gets or sets the type of the resource.
        /// </summary>
        [JsonProperty(Required = Required.Default, PropertyName = "provisioningState")]
        public string State { get; set; }

        [JsonProperty("subnetDetails")]
        public SubnetDetails SubnetDetails { get; set; }

        [JsonProperty("controllerDetails")]
        public ControllerDetails ControllerDetails { get; set; }
    }

    /// <summary>
    /// We cant use DelegateSubnet to deserialize because of derived/base relation and not adding TypeNameHandling.
    /// Hence created a contract for validating the response. 
    /// Later, we will use the SDK generated from Swagger as a way to validate the response.
    /// </summary>
    public class DelegateSubnetResponse
    {
        /// <summary>
        /// Gets the id for the resource.
        /// </summary>
        [JsonProperty(Required = Required.Always, PropertyName = "id")]
        public string Id { get; set; }

        /// <summary>
        /// Gets or sets the location of the resource.
        /// </summary>
        [JsonProperty(Required = Required.Default, PropertyName = "location")]
        public string Location { get; set; }

        /// <summary>
        /// Gets or sets the resource name.
        /// </summary>
        [JsonProperty(Required = Required.Default, PropertyName = "name")]
        public string Name { get; set; }

        /// <summary>
        /// Gets or sets the type of the resource.
        /// </summary>
        [JsonProperty(Required = Required.Default, PropertyName = "type")]
        public string Type { get; protected set; }

        /// <summary>
        /// Properties of a TopLevelResource
        /// </summary>
        [JsonProperty(PropertyName = "properties")]
        public DelegateSubnetResponseProperties Properties { get; protected set; }

    }
}
