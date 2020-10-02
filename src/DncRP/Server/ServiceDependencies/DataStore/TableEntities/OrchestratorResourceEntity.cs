// -----------------------------------------------------------------------------
// <copyright file="OrchestratorResourceEntity.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Microsoft.Azure.Cosmos.Table;

    /// <summary>
    /// Defines Table entity for OrchestratorResource
    /// </summary>
    public class OrchestratorResourceEntity : TableEntity
    {
        public OrchestratorResourceEntity()
        {
        }

        /// <summary>
        ///  Defines Table entity for OrchestratorResource
        /// </summary>
        /// <param name="subId"></param>
        /// subscription ID
        /// <param name="resourceGroup_resourceName">
        /// RowKey = resourceGroup_resourceName
        /// </param>
        public OrchestratorResourceEntity(string subId, string resourceGroup_resourceName)
        {
            PartitionKey = subId;
            RowKey = resourceGroup_resourceName;
        }

        public string Location
        {
            get; set;
        }

        public string ResouceGuid
        {
            get; set;
        }

        public string State
        {
            get; set;
        }

        public string CreatedDateUtc
        {
            get; set;
        }

        public string Properties
        {
            get; set;
        }

        public string OrchestratorType
        {
            get; set;
        }

        public string ControllerId
        {
            get; set;
        }
        public string ControllerProperties
        {
            get; set;
        }

        public string IdentityMetadata
        {
            get; set;
        }
    }
}