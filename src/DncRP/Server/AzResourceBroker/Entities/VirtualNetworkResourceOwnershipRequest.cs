// -----------------------------------------------------------------------------
// <copyright file="VirtualNetworkResourceOwnershipRequest.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;

    /// <summary>
    /// The virtual network resource ownership request class.
    /// </summary>
    public class VirtualNetworkResourceOwnershipRequest
    {
        /// <summary>
        /// Gets or sets the Id.
        /// </summary>
        [JsonProperty(Required = Required.Always)]
        public string Id { get; set; }

        /// <summary>
        /// Gets or sets the request properties.
        /// </summary>
        [JsonProperty]
        public VirtualNetworkOwnershipRequestProperties Properties { get; set; }

        /// <summary>
        /// Gets or sets the ownership action.
        /// </summary>
        [JsonProperty(Required = Required.Default)]
        public VirtualNetworkOwnershipAction OwnershipAction { get; set; }
    }
}
