// -----------------------------------------------------------------------------
// <copyright file="ResourceOwnershipRequest.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------


namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;

    /// <summary>
    /// The resource ownership request class.
    /// </summary>
    public class ResourceOwnershipRequest
    {
        /// <summary>
        /// Gets or sets the virtual networks.
        /// </summary>
        [JsonProperty]
        public VirtualNetworkResourceOwnershipRequest[] VirtualNetworks { get; set; }
    }
}
