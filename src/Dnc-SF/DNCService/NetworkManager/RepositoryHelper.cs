//-----------------------------------------------------------------------
// <copyright file="RepositoryHelper.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// 
//-----------------------------------------------------------------------

using DNCService;
using Microsoft.ServiceFabric.Data;
using Microsoft.ServiceFabric.Data.Collections;
using Newtonsoft.Json;
using System;
using System.Threading;
using System.Threading.Tasks;

namespace NetworkManager
{
    /// <summary>
    /// Repsoitory helpers.
    /// </summary>
    public class RepositoryHelper
    {
        /// <summary>
        /// Handle to the state manager.
        /// </summary>
        private static IReliableStateManager stateManager;

        /// <summary>
        /// Cached cancellation token.
        /// </summary>
        public static CancellationToken CancellationToken { get; private set; }

        /// <summary>
        /// Transaction timeout.
        /// </summary>
        private static TimeSpan transactionTimeout;

        /// <summary>
        /// True if repository helper is initialized.
        /// </summary>
        private static bool isInitialized = false;

        /// <summary>
        /// Synchronization lock.
        /// </summary>
        private static object  initLock = new Object();

        /// <summary>
        /// Helper to check if repository helper is initialized.
        /// </summary>
        private static void IsInitialized()
        {
            if (!isInitialized)
            {
                throw new Exception("Replica not initialized.");
            }
        }

        /// <summary>
        /// Initializes repository helper.
        /// </summary>
        /// <param name="stateManager">Supply the state manager.</param>
        /// <param name="cancellationToken">Supply the cancellation token.</param>
        /// <param name="transactionTimeout">Supply the default transaction timeout.</param>
        public static void Init(
            IReliableStateManager stateManager, 
            CancellationToken cancellationToken, 
            TimeSpan transactionTimeout)
        {
            lock (initLock) 
            {
                RepositoryHelper.stateManager = stateManager;
                RepositoryHelper.CancellationToken = cancellationToken;
                RepositoryHelper.transactionTimeout = transactionTimeout;
                RepositoryHelper.isInitialized = true;
            }
        }

        /// <summary>
        /// Async Helper to create or update a key value pair.
        /// </summary>
        /// <typeparam name="TKey">Supply the key type.</typeparam>
        /// <typeparam name="TValue">Supply the value type.</typeparam>
        /// <param name="tableName">Supply the table name.</param>
        /// <param name="key">Supply the key.</param>
        /// <param name="value">Supply the value.</param>
        /// <returns></returns>
        public static async Task CreateOrUpdateAsync<TKey, TValue>(
            string tableName, 
            TKey key, 
            TValue value) where TKey : IComparable<TKey>, IEquatable<TKey>
        {
            IsInitialized();
            IReliableDictionary<TKey, TValue> reliableDictionary = 
                await stateManager.GetOrAddAsync<IReliableDictionary<TKey, TValue>>(tableName);
            using (ITransaction tx = stateManager.CreateTransaction())
            {
                string serializedValue = JsonConvert.SerializeObject(value);
                ServiceEventSource.Current.Message($"Setting Table:{tableName} Key:{key} Value:{serializedValue}");

                await reliableDictionary.AddOrUpdateAsync(tx, key, value, (k,v) => value, transactionTimeout, CancellationToken);
                await tx.CommitAsync();
            }
        }

        /// <summary>
        /// Helper to create or update a key value pair.
        /// </summary>
        /// <typeparam name="TKey">Supply the key type.</typeparam>
        /// <typeparam name="TValue">Supply the value type.</typeparam>
        /// <param name="tableName">Supply the table name.</param>
        /// <param name="key">Supply the key.</param>
        /// <param name="value">Supply hte value.</param>
        /// <returns></returns>
        public static void CreateOrUpdate<TKey, TValue>(
            string tableName, 
            TKey key, 
            TValue value) where TKey : IComparable<TKey>, IEquatable<TKey>
        {
            Task createOrUpdateTask = CreateOrUpdateAsync(
                tableName, 
                key, 
                value);
            createOrUpdateTask.Wait();
        }

        /// <summary>
        /// Async Helper to retrieve value associated with the key.
        /// </summary>
        /// <typeparam name="TKey">Supply the key type.</typeparam>
        /// <typeparam name="TValue">Supply the value type.</typeparam>
        /// <param name="tableName">Supply the table name.</param>
        /// <param name="key">Supply the key.</param>
        /// <returns>Retrieved value.</returns>
        public static async Task<TValue> GetAsync<TKey, TValue>(
            string tableName, 
            TKey key) where TKey : IComparable<TKey>, IEquatable<TKey> where TValue : class, ICloneable
        {
            IsInitialized();
            IReliableDictionary<TKey, TValue> reliableDictionary = 
                await stateManager.GetOrAddAsync<IReliableDictionary<TKey, TValue>>(tableName);
            using (ITransaction tx = stateManager.CreateTransaction())
            {
                var result = await reliableDictionary.TryGetValueAsync(tx, key, LockMode.Default, transactionTimeout, CancellationToken);

                if (result.HasValue)
                {
                    ServiceEventSource.Current.Message("Retrieved data:{0}", result);
                    return result.Value.Clone() as TValue;
                }
            }

            return default(TValue);
        }

        /// <summary>
        /// Helper to retrieve value associated with the key.
        /// </summary>
        /// <typeparam name="TKey">Supply the key type.</typeparam>
        /// <typeparam name="TValue">Supply the value type.</typeparam>
        /// <param name="tableName">Supply the table name.</param>
        /// <returns>Retrieved value.</returns>
        public static TValue Get<TKey, TValue>(
            string tableName, 
            TKey key) where TKey : IComparable<TKey>, IEquatable<TKey> where TValue : class, ICloneable
        {
            Task<TValue> getTask = GetAsync<TKey, TValue>(
                tableName, 
                key);
            return getTask.Result;
        }

        /// <summary>
        /// Async Helper to delete a key.
        /// </summary>
        /// <typeparam name="TKey">Supply the key type.</typeparam>
        /// <typeparam name="TValue">Supply the value type.</typeparam>
        /// <param name="tableName">Supply the table name.</param>
        /// <param name="key">Supply the key.</param>
        public static async Task RemoveAsync<TKey, TValue>(
            string tableName,
            TKey key) where TKey : IComparable<TKey>, IEquatable<TKey>
        {
            IsInitialized();
            IReliableDictionary<TKey, TValue> reliableDictionary =
                await stateManager.GetOrAddAsync<IReliableDictionary<TKey, TValue>>(tableName);
            using (ITransaction tx = stateManager.CreateTransaction())
            {
                await reliableDictionary.TryRemoveAsync(tx, key, transactionTimeout, CancellationToken);
                await tx.CommitAsync();
            }
        }

        /// <summary>
        /// Helper to delete a key.
        /// </summary>
        /// <typeparam name="TKey">Supply the key type.</typeparam>
        /// <typeparam name="TValue">Supply the value type.</typeparam>
        /// <param name="tableName">Supply the table name.</param>
        /// <param name="key">Supply the key.</param>
        public static void Remove<TKey, TValue>(
            string tableName,
            TKey key) where TKey : IComparable<TKey>, IEquatable<TKey>
        {
            Task removeTask = RemoveAsync<TKey, TValue>(tableName, key);
            removeTask.Wait();
        }

        /// <summary>
        /// Helepr to create a transaction.
        /// </summary>
        /// <returns></returns>
        public static ITransaction GetTransaction()
        {
            IsInitialized();
            return stateManager.CreateTransaction();
        }

        /// <summary>
        /// Async Helper to retrieve a key from a reliable dictionary.
        /// </summary>
        /// <typeparam name="TKey">Supply the key type.</typeparam>
        /// <typeparam name="TValue">Supply the value type.</typeparam>
        /// <param name="tableName">Supply the table name.</param>
        /// <returns>Retrieved reliable collection.</returns>
        public static async Task<IReliableDictionary<TKey, TValue>> GetReliableDictionaryAsync<TKey, TValue>(
            string tableName) where TKey : IComparable<TKey>, IEquatable<TKey>
        {
            IReliableDictionary<TKey, TValue> reliableDictionary = await stateManager.GetOrAddAsync<IReliableDictionary<TKey, TValue>>(
                tableName);
            return reliableDictionary;
        }

        /// <summary>
        /// Async Helper to create or update a reliable dictionary.
        /// </summary>
        /// <typeparam name="TKey">Supply the key type.</typeparam>
        /// <typeparam name="TValue">Supply the value type.</typeparam>
        /// <param name="reliableDictionary">Supply the reliable dictionary.</param>
        /// <param name="key">Supply the key.</param>
        /// <param name="value">Supply the value.</param>
        public static async Task CreateOrUpdateReliableDictionary<TKey, TValue>(
            ITransaction tx,
            IReliableDictionary<TKey, TValue> reliableDictionary,
            TKey key,
            TValue value) where TKey : IComparable<TKey>, IEquatable<TKey>
        {
            IsInitialized();
            string serializedValue = JsonConvert.SerializeObject(value);
            ServiceEventSource.Current.Message($"Setting Key:{key} Value:{serializedValue}");

            await reliableDictionary.AddOrUpdateAsync(tx, key, value, (k, v) => value, transactionTimeout, CancellationToken);
        }

        /// <summary>
        /// Helper to commit a given transaction.
        /// </summary>
        /// <param name="tx">Supply the transaction.</param>
        /// <returns></returns>
        public static async Task CommitTransactionAsync(ITransaction tx)
        {
            await tx.CommitAsync();
        }

        /// <summary>
        /// Async Helper to retrieve a value associated with a key from a reliable dictionary.
        /// </summary>
        /// <typeparam name="TKey">Supply the key type.</typeparam>
        /// <typeparam name="TValue">Supply the value type.</typeparam>
        /// <param name="reliableDictionary">Supply the reliable dictionary.</param>
        /// <param name="key">Supply the key.</param>
        /// <param name="lockMode"> Supply the lock mode</param>
        /// <param name="tx">Supply the transaction.</param>
        /// <returns>Retrieved value.</returns>
        public static async Task<TValue> GetFromReliableDictionary<TKey, TValue>(
            ITransaction tx,
            IReliableDictionary<TKey, TValue> reliableDictionary,
            TKey key,
            LockMode lockMode = LockMode.Default) where TKey : IComparable<TKey>, IEquatable<TKey> where TValue : class, ICloneable
        {
            var result = await reliableDictionary.TryGetValueAsync(tx, key, lockMode, transactionTimeout, CancellationToken);

            if (result.HasValue)
            {
                return result.Value.Clone() as TValue;
            }

            return default(TValue);
        }

        /// <summary>
        /// Async Helper to remove a key from a reliable dictionary.
        /// </summary>
        /// <typeparam name="TKey">Supply the key type.</typeparam>
        /// <typeparam name="TValue">Supply the value type.</typeparam>
        /// <param name="reliableDictionary">Supply the reliable dictionary.</param>
        /// <param name="key">Supply the key.</param>
        public static async Task RemoveFromReliableDictionary<TKey, TValue>(
            ITransaction tx,
            IReliableDictionary<TKey, TValue> reliableDictionary,
            TKey key) where TKey : IComparable<TKey>, IEquatable<TKey>
        {
            IsInitialized();
            await reliableDictionary.TryRemoveAsync(tx, key, transactionTimeout, CancellationToken);
        }
    }
}