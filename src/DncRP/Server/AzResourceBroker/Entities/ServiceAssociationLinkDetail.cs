// -----------------------------------------------------------------------------
// <copyright file="ServiceAssociationLinkDetail.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Text.Json.Serialization;

    /// <summary>
    /// The Azure service association link detail object.
    /// </summary>
    public class ServiceAssociationLinkDetail
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
        /// Gets or sets the name of the SAL detail properties.
        /// </summary>
        [JsonPropertyName("Properties")]
        public ServiceAssociationLinkDetailProperties Properties { get; set; }
    }
}
