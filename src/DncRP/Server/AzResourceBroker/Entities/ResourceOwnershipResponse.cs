// -----------------------------------------------------------------------------
// <copyright file="ResourceOwnershipResponse.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Text.Json.Serialization;

    /// <summary>
    /// The resource ownership response class.
    /// </summary>
    public class ResourceOwnershipResponse
    {
        /// <summary>
        /// Gets or sets the virtual networks.
        /// </summary>
        [JsonPropertyName("VirtualNetworks")]
        public VirtualNetworkOwnershipResponse[] VirtualNetworks { get; set; }
    }
}
