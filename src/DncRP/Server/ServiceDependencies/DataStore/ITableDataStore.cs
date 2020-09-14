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
        /// Initialize the DataStore
        /// </summary>
        Task InitAsync();

        /// <summary>
        /// Creates CloudTable with TableName. Idempotent if the table is already created.
        /// </summary>
        /// <param name="tableName">table name</param>
        void CreateTable(string tableName);

        /// <summary>
        /// Inserts or Update a TableEntity into a given table. Idempotent if the entity is already updated.
        /// </summary>
        /// <param name="tableName">table name</param>
        /// <param name="entity">table entity</param>
        /// <returns>Result of an operation</returns>
        Task<TableResult> InsertOrMergeEntityAsync(string tableName, TableEntity entity);

        /// <summary>
        /// Get the Table entity of a given tablename
        /// </summary>
        /// <typeparam name="T">TableEntity type</typeparam>
        /// <param name="tableName">table name</param>
        /// <param name="partitionKey">parition Key</param>
        /// <param name="rowKey">row key</param>
        /// <returns></returns>
        Task<TableResult> GetEntityAsync<T>(string tableName, string partitionKey, string rowKey) where T : ITableEntity;
    }
}
