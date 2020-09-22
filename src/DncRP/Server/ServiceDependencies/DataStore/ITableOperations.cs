// -----------------------------------------------------------------------------
// <copyright file="ITableOperations.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    /// <summary>
    /// Defines interfaces for all the table operations supported.
    /// </summary>
    public interface ITableOperations
    {
        /// <summary>
        /// SubnetTokenMap operations
        /// </summary>
        /// <returns></returns>
        ISubnetTokenMapOperations SubnetTokenMapOperations { get; }

        /// <summary>
        /// DelegatedSubnetConfiguration operations
        /// </summary>
        /// <returns></returns>
        IDelegatedSubnetConfigurationOperations DelegatedSubnetConfigurationOperations { get; }

        /// <summary>
        /// ControllerConfiguration operations
        /// </summary>
        /// <returns></returns>
        IControllerConfiguration ControllerConfiguration { get; }

        IOrchestratorOperations OrchestratorOperations { get; }
    }
}
