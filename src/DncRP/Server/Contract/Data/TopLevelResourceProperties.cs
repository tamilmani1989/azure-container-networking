// --------------------------------------------------------------------------------------------------------------------
// <copyright file="ResourceDefinition.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Text.Json.Serialization;

    public abstract class TopLevelResourceProperties
    {
        /// <summary>
        /// Gets or sets the type of the resource.
        /// </summary>
        [JsonPropertyName("resourceGuid")]
        public string ResourceGuid { get; set; }

        /// <summary>
        /// Gets or sets the type of the resource.
        /// </summary>
        [JsonPropertyName("state")]
        public string State { get; set; }
    }
}
