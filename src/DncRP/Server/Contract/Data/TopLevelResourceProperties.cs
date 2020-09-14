// --------------------------------------------------------------------------------------------------------------------
// <copyright file="ResourceDefinition.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------


namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;

    public abstract class TopLevelResourceProperties
    {
        /// <summary>
        /// Gets or sets the type of the resource.
        /// </summary>
        [JsonProperty(Required = Required.Default, PropertyName = "resourceGuid")]
        public string ResourceGuid { get; set; }

        /// <summary>
        /// Gets or sets the type of the resource.
        /// </summary>
        [JsonProperty(Required = Required.Default, PropertyName = "state")]
        public string State { get; set; }
    }
}
