// -----------------------------------------------------------------------------
// <copyright file="SubscriptionResourceEntity.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Microsoft.Azure.Cosmos.Table;
    using System;

    /// <summary>
    /// Defines Table entity for ControllerResource
    /// </summary>
    public class SubscriptionResourceEntity : TableEntity
    {
        public SubscriptionResourceEntity()
        {
        }

        /// <summary>
        ///  Defines Table entity for SubscriptionResourceEntity
        /// </summary>
        /// <param name="subId"></param>
        /// PartitionKey = subscription ID
        /// <param name="tenantId">
        /// RowKey = tenantId
        /// </param>
        public SubscriptionResourceEntity(string subId, string tenantId)
        {
            PartitionKey = subId;
            RowKey = tenantId;
        }

        public string RegistrationState
        {
            get; set;
        }

        public DateTime CreatedDateUtc
        {
            get; set;
        }

        public string RequestBody
        {
            get; set;
        }
    }
}