// -----------------------------------------------------------------------------
// <copyright file="ServiceAssociationLinkProperties.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Collections.Generic;
    using System.Text.Json.Serialization;

    /// <summary>
    /// The service association link properties.
    /// </summary>
    public class ServiceAssociationLinkProperties
    {
        /// <summary>
        /// Gets or sets the link property of the service association link.
        /// </summary>
        [JsonPropertyName("Link")]
        public string Link { get; set; }

        /// <summary>
        /// Gets or sets the resource type that will be created by the association service.
        /// </summary>
        [JsonPropertyName("LinkedResourceType")]
        public string LinkedResourceType { get; set; }

        /// <summary>
        /// Gets or sets the VNet resource guid. NRP will do additional validation if provided.
        /// </summary>
        [JsonPropertyName("ParentResourceGuid")]
        public string ParentResourceGuid { get; set; }

        /// <summary>
        /// Gets or sets a value indicating whether allow NRP to delete the service association link.
        /// </summary>
        [JsonPropertyName("AllowDelete")]
        public bool? AllowDelete { get; set; }

        /// <summary>
        /// Gets or sets the locations.
        /// </summary>
        [JsonPropertyName("Locations")]
        public string[] Locations { get; set; }

        /// <summary>
        /// Gets or sets the service association detail.
        /// </summary>
        [JsonPropertyName("Details")]
        public IList<ServiceAssociationLinkDetail> Details { get; set; }

        /// <summary>
        /// Gets or sets the provisioning state.
        /// </summary>
        [JsonPropertyName("ProvisioningState")]
        public string ProvisioningState { get; set; }
    }
}
