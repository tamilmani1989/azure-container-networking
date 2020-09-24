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
        /// <param name="resourceName">resourceName</param>
        /// <param name="delegatedSubnetGuid">DelegatedSubnet Resource Guid</param>
        /// <param name="vnetResourceGuid">VnetResourceGuid</param>
        /// <param name="DelegateSubnetRequest">DelegateSubnetRequest</param>
        /// <returns></returns>
        Task<DelegateSubnet> InsertOrUpdateAsync(
            string subscriptionId,
            string resourceGroupName,
            string resourceName,
            string delegatedSubnetGuid,
            string vnetResourceGuid,
            DelegateSubnetRequest delegateSubnetRequest);

        /// <summary>
        /// Get the DelegatedSubnet resource if an entity exists for a given Subnet.
        /// </summary>
        /// <param name="subscriptionId">subscription id</param>
        /// <param name="resourceGroupName">resourcegroup name</param>
        /// <param name="resourceName">resourcename</param>
        /// <returns></returns>
        Task<DelegateSubnet> TryGetDelegatedSubnetConfigurationAsync(
            string subscriptionId,
            string resourceGroupName,
            string resourceName);

        /// <summary>
        /// Delete DelegateSubnetResource entity if exists.
        /// </summary>
        /// <param name="delegatedSubnetId"></param>
        /// <param name="subnet"></param>
        /// <returns></returns>
        Task DeleteAsync(
            string subscriptionId,
            string resourceGroupName,
            string resourceName);
    }
}
