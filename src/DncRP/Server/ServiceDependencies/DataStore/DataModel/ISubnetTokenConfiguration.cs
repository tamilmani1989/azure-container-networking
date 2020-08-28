// -----------------------------------------------------------------------------
// <copyright file="ISubnetTokenConfiguration.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

using System.Threading.Tasks;

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    /// <summary>
    /// Defines SubnetTokenConfiguration interface for Data operations
    /// </summary>
    public interface ISubnetTokenConfiguration
    {
        /// <summary>
        /// Persist the entry in the SubnetTokenConfiguration
        /// </summary>
        /// <param name="vnetId"></param>
        /// <param name="subnet"></param>
        /// <param name="primaryTokenId"></param>
        /// <param name="secondaryTokenId"></param>
        Task InsertOrUpdate(string vnetId, string subnet, string primaryTokenId, string secondaryTokenId);
    }
}
