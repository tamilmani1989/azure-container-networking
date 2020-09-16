// -----------------------------------------------------------------------------
// <copyright file="ControllerConfigurationEntity.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Microsoft.Azure.Cosmos.Table;

    /// <summary>
    /// Defines Table entity for ControllerConfigurationEntity
    /// </summary>
    public class ControllerConfigurationEntity : TableEntity
    {
        public ControllerConfigurationEntity()
        {
        }

        /// <summary>
        ///  Defines Table entity for SubnetTokenConfigurationTable
        /// </summary>
        /// <param name="managedDncId"></param>
        /// managedDncId
        /// <param name="controllerId">
        /// RowKey = controllerId
        /// </param>
        public ControllerConfigurationEntity(string managedDncId, string controllerId)
        {
            PartitionKey = managedDncId;
            RowKey = controllerId;
        }

        public string ControllerType
        {
            get; set;
        }

        public string Properties
        {
            get; set;
        }
    }
}
