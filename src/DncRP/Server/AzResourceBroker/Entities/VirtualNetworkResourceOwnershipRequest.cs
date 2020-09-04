// -----------------------------------------------------------------------------
// <copyright file="VirtualNetworkResourceOwnershipRequest.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Text.Json.Serialization;

    /// <summary>
    /// The virtual network resource ownership request class.
    /// </summary>
    public class VirtualNetworkResourceOwnershipRequest
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
        public VirtualNetworkOwnershipRequestProperties Properties { get; set; }

        /// <summary>
        /// Gets or sets the ownership action.
        /// </summary>
        [JsonPropertyName("OwnershipAction")]
        public VirtualNetworkOwnershipAction OwnershipAction { get; set; }
    }
}
