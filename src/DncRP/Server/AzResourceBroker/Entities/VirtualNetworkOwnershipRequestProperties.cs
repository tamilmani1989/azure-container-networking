// -----------------------------------------------------------------------------
// <copyright file="VirtualNetworkOwnershipRequestProperties.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;

    /// <summary>
    /// The virtual network ownership request properties class.
    /// </summary>
    public class VirtualNetworkOwnershipRequestProperties
    {
        /// <summary>
        /// Gets or sets a value indicating whether enable service endpoints.
        /// </summary>
        [JsonProperty]
        public bool EnableServiceEndpoints { get; set; }
    }
}
