// -----------------------------------------------------------------------------
// <copyright file="ServiceAssociationLink.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;

    /// <summary>
    /// The service association link resource.
    /// </summary>
    public class ServiceAssociationLink
    {
        /// <summary>
        /// Gets or sets the Id.
        /// </summary>
        [JsonProperty(Required = Required.Default)]
        public string Id { get; set; }

        /// <summary>
        /// Gets or sets the Name.
        /// </summary>
        [JsonProperty(Required = Required.Default)]
        public string Name { get; set; }

        /// <summary>
        /// Gets or sets the ETAG.
        /// </summary>
        [JsonProperty]
        public string Etag { get; set; }

        /// <summary>
        /// Gets or sets the service association link properties.
        /// </summary>
        [JsonProperty]
        public ServiceAssociationLinkProperties Properties { get; set; }
    }
}
