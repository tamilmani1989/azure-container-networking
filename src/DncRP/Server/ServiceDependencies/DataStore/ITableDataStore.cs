// -----------------------------------------------------------------------------
// <copyright file="ITableDataStore.cs" company="Microsoft">
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
    /// TableDataStore interface, used for initializing the backened data storage.
    /// </summary>
    public interface ITableDataStore
    {
        /// <summary>
        /// GetTableClient
        /// </summary>
        /// <returns></returns>
        CloudTableClient GetTableClient();
    }
}
