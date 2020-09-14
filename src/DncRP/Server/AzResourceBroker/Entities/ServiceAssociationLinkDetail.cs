// -----------------------------------------------------------------------------
// <copyright file="ServiceAssociationLinkDetail.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;

    /// <summary>
    /// The Azure service association link detail object.
    /// </summary>
    public class ServiceAssociationLinkDetail
    {
        /// <summary>
        /// Gets or sets the Id.
        /// </summary>
        [JsonProperty]
        public string Id { get; set; }

        /// <summary>
        /// Gets or sets the Name.
        /// </summary>
        [JsonProperty]
        public string Name { get; set; }

        /// <summary>
        /// Gets or sets the ETAG.
        /// </summary>
        [JsonProperty]
        public string Etag { get; set; }

        /// <summary>
        /// Gets or sets the name of the SAL detail properties.
        /// </summary>
        [JsonProperty]
        public ServiceAssociationLinkDetailProperties Properties { get; set; }
    }
}
