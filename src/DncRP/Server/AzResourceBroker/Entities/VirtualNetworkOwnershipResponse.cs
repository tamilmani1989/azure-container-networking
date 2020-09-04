// -----------------------------------------------------------------------------
// <copyright file="VirtualNetworkOwnershipResponse.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Text.Json.Serialization;

    /// <summary>
    /// The virtual network ownership response class.
    /// </summary>
    public class VirtualNetworkOwnershipResponse
    {
        /// <summary>
        /// Gets or sets the Id.
        /// </summary>
        [JsonPropertyName("Id")]
        public string Id { get; set; }

        /// <summary>
        /// Gets or sets the request properties.
        /// </summary>
        [JsonPropertyName("Properties")]
        public VirtualNetworkOwnershipResponseProperties Properties { get; set; }

        /// <summary>
        /// Gets or sets the ownership action.
        /// </summary>
        [JsonPropertyName("OwnershipAction")]
        public VirtualNetworkOwnershipAction OwnershipAction { get; set; }

        /// <summary>
        /// Gets or sets the provisioning state.
        /// </summary>
        [JsonPropertyName("ProvisioningState")]
        public string ProvisioningState { get; set; }

        /// <summary>
        /// Gets or sets the type.
        /// </summary>
        [JsonPropertyName("Type")]
        public string Type { get; set; }
    }
}
