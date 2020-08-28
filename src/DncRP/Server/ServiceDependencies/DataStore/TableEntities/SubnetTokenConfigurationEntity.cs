// -----------------------------------------------------------------------------
// <copyright file="SubnetTokenConfigurationEntity.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Microsoft.Azure.Cosmos.Table;

    /// <summary>
    /// Defines Table entity for SubnetTokenConfigurationTable
    /// </summary>
    public class SubnetTokenConfigurationEntity : TableEntity
    {
        public SubnetTokenConfigurationEntity()
        {
        }

        public SubnetTokenConfigurationEntity(string vnetId, string subnetName)
        {
            PartitionKey = vnetId;
            RowKey = subnetName;
        }

        public string PrimaryTokenId { get; set; }
        public string SecondaryTokenId { get; set; }
    }
}
