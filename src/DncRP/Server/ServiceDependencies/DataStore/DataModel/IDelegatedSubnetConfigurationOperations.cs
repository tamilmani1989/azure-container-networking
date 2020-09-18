// -----------------------------------------------------------------------------
// <copyright file="IDelegatedSubnetConfigurationOperations.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

using System;
using System.Threading.Tasks;

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    /// <summary>
    /// Defines SubnetTokenConfiguration interface for Data operations
    /// </summary>
    public interface IDelegatedSubnetConfigurationOperations
    {
        /// <summary>
        /// Insert or update DelegatedSubnetConfigurationEntity
        /// </summary>
        /// <param name="subscriptionId">subscriptionId</param>
        /// <param name="resourceGroupName">resourceGroupName</param>
        /// <param name="vnetName">VirtualNetwork name</param>
        /// <param name="subnet">subnetname</param>
        /// <param name="delegatedSubnetName">DelegatedSubnetId</param>
        /// <param name="delegatedSubnetGuid">DelegatedSubnet Guid</param>
        /// <param name="vnetResourceGuid">VnetResource Guid</param>
        /// <returns></returns>
        Task<DelegateSubnet> InsertOrUpdateAsync(
            string subscriptionId,
            string resourceGroupName,
            string vnetName,
            string subnet,
            string delegatedSubnetName,
            string delegatedSubnetGuid,
            string vnetResourceGuid);

        /// <summary>
        /// Get the DelegatedSubnet Guid if an entity exists for a given Subnet.
        /// </summary>
        /// <param name="subscriptionId">subscription id</param>
        /// <param name="resourceGroupName">resourcegroup name</param>
        /// <param name="vnetName">vnetName</param>
        /// <param name="subnet">subnet</param>
        /// <returns></returns>
        Task<DelegateSubnet> TryGetDelegatedSubnetConfigurationAsync(
            string subscriptionId,
            string resourceGroupName,
            string vnetName,
            string subnet,
            string resourceName,
            bool throwOnResourceNameComparison = false);
    }
}
