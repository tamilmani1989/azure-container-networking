// -----------------------------------------------------------------------------
// <copyright file="ServiceAssociationLinkDetailProperties.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;

    /// <summary>
    /// The Azure service association link detail properties.
    /// </summary>
    public class ServiceAssociationLinkDetailProperties
    {
        /// <summary>
        /// Gets or sets the provisioning state.
        /// </summary>
        [JsonProperty]
        public string ProvisioningState { get; set; }

        /// <summary>
        /// Gets or sets the primary context request Id.
        /// </summary>
        [JsonProperty]
        public string PrimaryContextRequestId { get; set; }

        /// <summary>
        /// Gets or sets the secondary context request Id.
        /// </summary>
        [JsonProperty]
        public string SecondaryContextRequestId { get; set; }

        /// <summary>
        /// Gets or sets the primary context request Id.
        /// </summary>
        [JsonProperty]
        public string PrimaryContextId { get; set; }

        /// <summary>
        /// Gets or sets the secondary context request Id.
        /// </summary>
        [JsonProperty]
        public string SecondaryContextId { get; set; }

        /// <summary>
        /// ToString
        /// </summary>
        /// <returns></returns>
        public override string ToString()
        {
            return $"ServiceAssociationLinkDetailProperties: ProvisioningState: {ProvisioningState}, " +
                $"PrimaryContextRequestId: {PrimaryContextRequestId}, SecondaryContextRequestId: {SecondaryContextRequestId}," +
                $"PrimaryContextId: {PrimaryContextId} SecondaryContextId: {SecondaryContextId}";
        }
    }
}
