// -----------------------------------------------------------------------------
// <copyright file="VirtualNetworkOwnershipResponse.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;

    /// <summary>
    /// The virtual network ownership response class.
    /// </summary>
    public class VirtualNetworkOwnershipResponse
    {
        /// <summary>
        /// Gets or sets the Id.
        /// </summary>
        [JsonProperty]
        public string Id { get; set; }

        /// <summary>
        /// Gets or sets the request properties.
        /// </summary>
        [JsonProperty]
        public VirtualNetworkOwnershipResponseProperties Properties { get; set; }

        /// <summary>
        /// Gets or sets the ownership action.
        /// </summary>
        [JsonProperty]
        public VirtualNetworkOwnershipAction OwnershipAction { get; set; }

        /// <summary>
        /// Gets or sets the provisioning state.
        /// </summary>
        [JsonProperty]
        public string ProvisioningState { get; set; }

        /// <summary>
        /// Gets or sets the type.
        /// </summary>
        [JsonProperty]
        public string Type { get; set; }
    }
}
