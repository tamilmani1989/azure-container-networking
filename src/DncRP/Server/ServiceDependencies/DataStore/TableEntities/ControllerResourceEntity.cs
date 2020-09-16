// -----------------------------------------------------------------------------
// <copyright file="ControllerResourceEntity.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Microsoft.Azure.Cosmos.Table;

    /// <summary>
    /// Defines Table entity for ControllerResource
    /// </summary>
    public class ControllerResourceEntity : TableEntity
    {
        public ControllerResourceEntity()
        {
        }

        /// <summary>
        ///  Defines Table entity for ControllerResourceEntity
        /// </summary>
        /// <param name="subId"></param>
        /// subscription ID
        /// <param name="resourceGroup_controllerName">
        /// RowKey = resourcegroup_controllerName
        /// </param>
        public ControllerResourceEntity(string subId, string resourceGroup_controllerName)
        {
            PartitionKey = subId;
            RowKey = resourceGroup_controllerName;
        }

        public string ControllerId
        {
            get; set;
        }

        public string ControllerType
        {
            get; set;
        }

        public string ManagedDNCId
        {
            get; set;
        }

        public string Properties
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
    }
}