// -----------------------------------------------------------------------------
// <copyright file="VirtualNetworkOwnershipResponseProperties.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;

    /// <summary>
    /// The virtual network ownership response properties class.
    /// </summary>
    public class VirtualNetworkOwnershipResponseProperties
    {
        /// <summary>
        /// Gets or sets a value indicating whether enable service endpoints.
        /// </summary>
        [JsonProperty]
        public bool EnableServiceEndpoints { get; set; }

        /// <summary>
        /// Gets or sets the platform identifier.
        /// </summary>
        [JsonProperty]
        public string PlatformIdentifier { get; set; }

        /// <summary>
        /// Gets or sets the owner.
        /// </summary>
        [JsonProperty]
        public string Owner { get; set; }
    }
}
