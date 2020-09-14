// -----------------------------------------------------------------------------
// <copyright file="SubnetTokenMapEntity.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Microsoft.Azure.Cosmos.Table;
    using System;

    /// <summary>
    /// Defines Table entity for SubnetTokenMap Table
    /// </summary>
    public class SubnetTokenMapEntity : TableEntity
    {
        public Guid VnetId { get; set; }

        public string PrimaryTokenId { get; set; }

        public string SecondaryTokenId { get; set; }

        public SubnetTokenMapEntity()
        {
        }

        public SubnetTokenMapEntity(string delegatedSubnetGuid, string subnetName)
        {
            PartitionKey = delegatedSubnetGuid;
            RowKey = subnetName;
        }

        public override string ToString()
        {
            return $"SubnetTokenConfigurationEntity: PartitionKey: {PartitionKey}, RowKey: {RowKey} " +
                $"PrimaryTokenId: {PrimaryTokenId}, SecondaryTokenId: {SecondaryTokenId}, VnetId :{VnetId}";
        }
    }
}
