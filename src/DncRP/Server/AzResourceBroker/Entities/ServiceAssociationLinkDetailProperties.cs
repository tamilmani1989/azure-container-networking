// -----------------------------------------------------------------------------
// <copyright file="ServiceAssociationLinkDetailProperties.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Text.Json.Serialization;

    /// <summary>
    /// The Azure service association link detail properties.
    /// </summary>
    public class ServiceAssociationLinkDetailProperties
    {
        /// <summary>
        /// Gets or sets the provisioning state.
        /// </summary>
        [JsonPropertyName("ProvisioningState")]
        public string ProvisioningState { get; set; }

        /// <summary>
        /// Gets or sets the primary context request Id.
        /// </summary>
        [JsonPropertyName("PrimaryContextRequestId")]
        public string PrimaryContextRequestId { get; set; }

        /// <summary>
        /// Gets or sets the secondary context request Id.
        /// </summary>
        [JsonPropertyName("SecondaryContextRequestId")]
        public string SecondaryContextRequestId { get; set; }

        /// <summary>
        /// Gets or sets the primary context request Id.
        /// </summary>
        [JsonPropertyName("PrimaryContextId")]
        public string PrimaryContextId { get; set; }

        /// <summary>
        /// Gets or sets the secondary context request Id.
        /// </summary>
        [JsonPropertyName("SecondaryContextId")]
        public string SecondaryContextId { get; set; }
    }
}
