// -----------------------------------------------------------------------------
// <copyright file="ServiceAssociationLink.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Text.Json.Serialization;

    /// <summary>
    /// The service association link resource.
    /// </summary>
    public class ServiceAssociationLink
    {
        /// <summary>
        /// Gets or sets the Id.
        /// </summary>
        [JsonPropertyName("Id")]
        public string Id { get; set; }

        /// <summary>
        /// Gets or sets the Name.
        /// </summary>
        [JsonPropertyName("Name")]
        public string Name { get; set; }

        /// <summary>
        /// Gets or sets the ETAG.
        /// </summary>
        [JsonPropertyName("Etag")]
        public string Etag { get; set; }

        /// <summary>
        /// Gets or sets the service association link properties.
        /// </summary>
        [JsonPropertyName("Properties")]
        public ServiceAssociationLinkProperties Properties { get; set; }
    }
}
