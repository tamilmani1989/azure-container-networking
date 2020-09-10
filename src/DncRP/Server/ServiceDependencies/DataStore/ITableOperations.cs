// -----------------------------------------------------------------------------
// <copyright file="ITableOperations.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Microsoft.Azure.Cosmos.Table;
    using System.Collections.Generic;
    using System.Security.Cryptography.X509Certificates;
    using System.Threading.Tasks;

    /// <summary>
    /// Defines interfaces for all the table operations supported.
    /// </summary>
    public interface ITableOperations
    {
        /// <summary>
        /// SubnetTokenConfiguration operations
        /// </summary>
        /// <returns></returns>
        ISubnetTokenConfiguration SubnetTokenConfiguration { get; }

        /// <summary>
        /// ControllerConfiguration operations
        /// </summary>
        /// <returns></returns>
        IControllerConfiguration ControllerConfiguration
        {
            get;
        }
    }
}
