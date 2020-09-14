// -----------------------------------------------------------------------------
// <copyright file="DelegatedSubnetConfigurationEntity.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Microsoft.Azure.Cosmos.Table;
    using System;

    /// <summary>
    /// Defines Table entity for DelegatedSubnetConfigurationEntity
    /// </summary>
    public class DelegatedSubnetConfigurationEntity : TopLevelResourceEntity
    {
        public string ResourceGroup { get; private set; }

        public string VirtualNetwork { get; private set; }

        public string Subnet { get; private set; }

        public DelegatedSubnetConfigurationEntity()
        {
        }

        public DelegatedSubnetConfigurationEntity(string subscriptionId, string resourceGroupName, string vnetName, string subnetName)
        {
            PartitionKey = subscriptionId;
            RowKey = DelegatedSubnetConfigurationEntity.GetSubnetConfigurationKey(resourceGroupName, vnetName, subnetName);

            ResourceGroup = resourceGroupName;
            VirtualNetwork = vnetName;
            Subnet = subnetName;
        }

        /// <summary>
        /// Generates the RowKey for a given Subnet
        /// </summary>
        /// <param name="resourceGroupName">resourceGroup name</param>
        /// <param name="vnetName">vnet name</param>
        /// <param name="subnet">subnet name</param>
        /// <returns></returns>
        public static string GetSubnetConfigurationKey(
            string resourceGroupName, 
            string vnetName, 
            string subnet)
        {
            return $"{resourceGroupName}_{vnetName}_{subnet}";
        }
    }
}
