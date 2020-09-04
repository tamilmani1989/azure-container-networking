// -----------------------------------------------------------------------------
// <copyright file="ITableDataStore.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Threading.Tasks;
    using Microsoft.Azure.Cosmos.Table;

    /// <summary>
    /// TableDataStore interface, used for initializing the backened data storage.
    /// </summary>
    public interface ITableDataStore
    {
        /// <summary>
        /// GetTableClient
        /// </summary>
        /// <returns></returns>
        Task<CloudTableClient> GetTableClient();
    }
}
