// -----------------------------------------------------------------------------
// <copyright file="ISubnetTokenMapOperations.cs" company="Microsoft">
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
    public interface ISubnetTokenMapOperations
    {
        /// <summary>
        /// Insert of update into SubnetTokenConfiguration table.
        /// </summary>
        /// <param name="delegatedSubnetId">delegatedSubnet ResourceId</param>
        /// <param name="subnetName"></param>
        /// <param name="primaryTokenId"></param>
        /// <param name="secondaryTokenId"></param>
        /// <param name="vnetResourceGuid"></param>
        Task InsertOrUpdateAsync(
            string delegatedSubnetId,
            string subnet,
            string primaryTokenId,
            string secondaryTokenId,
            string vnetResourceGuid);

        /// <summary>
        /// Check if token exists for a given DelegatedSubnet.
        /// </summary>
        /// <param name="delegatedSubnetId"></param>
        /// <param name="subnet"></param>
        /// <returns></returns>
        Task<bool> SubnetTokenExistsAsync(
            string delegatedSubnetId,
            string subnet);

        /// <summary>
        /// Delete the SubnetTokenMap entity if exists.
        /// </summary>
        /// <param name="delegatedSubnetId"></param>
        /// <param name="subnet"></param>
        /// <returns></returns>
        Task DeleteAsync(
            string delegatedSubnetId,
            string subnet);
    }
}
