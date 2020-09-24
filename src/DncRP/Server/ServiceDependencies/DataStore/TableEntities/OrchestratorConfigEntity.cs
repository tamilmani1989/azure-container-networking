// -----------------------------------------------------------------------------
// <copyright file="OrchestratorConfigEntity.cs" company="Microsoft">
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
    public class OrchestratorConfigEntity : TableEntity
    {
        public OrchestratorConfigEntity()
        {
        }

        /// <summary>
        ///  Defines Table entity for OrchestratorResource
        /// </summary>
        /// <param name="managedDncId"></param>
        /// subscription ID
        /// <param name="orchestratorResourceGuid">
        /// RowKey = resourceGroup_resourceName
        /// </param>
        public OrchestratorConfigEntity(string managedDncId, string orchestratorResourceGuid)
        {
            PartitionKey = managedDncId;
            RowKey = orchestratorResourceGuid;
        }

        public string Properties
        {
            get; set;
        }

        public string OrchestratorType
        {
            get; set;
        }
    }
}