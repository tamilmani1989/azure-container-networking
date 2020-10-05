// --------------------------------------------------------------------------------------------------------------------
// <copyright file="SubscriptionOperation.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// ---------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Azure.Cosmos.Table;
    using Microsoft.Extensions.Logging;
    using System;
    using System.Collections;
    using System.Collections.Generic;
    using System.Linq;
    using System.Net;
    using System.Threading.Tasks;

    public class SubscriptionOperation : ISubscriptionOperation
    {
        private ITableDataStore dataStore;
        private ILogger logger;
        private const string tableName = "SubscriptionResource";

        /// <summary>
        /// Create an instance of TableOperations
        /// TODO: Pass the DataStore Settings
        /// </summary>
        /// <param name="storageConnectionString">Connection string, configuration from appsettings.Json</param>
        public SubscriptionOperation(
            ITableDataStore tableDataStore,
            ILogger logger)
        {
            this.dataStore = tableDataStore;
            this.logger = logger;

            dataStore.CreateTable(tableName);
        }

        /// <summary>
        /// GetTenantInfo
        /// </summary>
        /// <param name="subscriptionId"></param>
        /// <returns></returns>
        public async Task<string> GetTenantIdAsync(string subscriptionId)
        {
            SubscriptionResourceEntity resourceEntity = await GetSubscriptionResourceAsync(subscriptionId);

            // TODO - Add Metric event
            logger.LogInformation($"Subscription: {resourceEntity.PartitionKey}, TenantId: {resourceEntity.RowKey}");
            return resourceEntity.RowKey;
        }

        /// <summary>
        /// Insert of update into SubscriptionResource table.
        /// </summary>
        /// <param name="subscriptionId">subscriptionId</param>
        /// <returns></returns>
        public async Task<bool> IsSubscriptionRegisteredAsync(
            string subscriptionId)
        {
            SubscriptionResourceEntity resourceEntity = await TryGetSubscriptionResourceAsync(subscriptionId);

            // TODO - Add Metric event
            if (resourceEntity == null)
            {
                return false;
            }

            return true;
        }

        /// <summary>
        /// Validate Subscription is registered
        /// </summary>
        /// <param name="subscriptionId"></param>
        /// <returns></returns>
        public async void ValidateSubscriptionRegistrationAsync(string subscriptionId)
        {
            await GetSubscriptionResourceAsync(subscriptionId);
        }

        /// <summary>
        /// Insert of update into SubscriptionResource table.
        /// </summary>
        /// <param name="subscriptionId">subscriptionId</param>
        /// <param name="tenantId">tenantId</param>
        /// <param name="state">state</param>
        /// <returns></returns>
        public async Task RegisterSubAsync(
            string subscriptionId,
            string tenantId,
            string requestBody)
        {
            // Check if entry already exists.
            SubscriptionResourceEntity resourceEntity = await TryGetSubscriptionResourceAsync(subscriptionId);

            if (resourceEntity == null)
            {
                resourceEntity = new SubscriptionResourceEntity(subscriptionId, tenantId);
                resourceEntity.CreatedDateUtc = DateTime.UtcNow;
                resourceEntity.RequestBody = requestBody;
                resourceEntity.RegistrationState = SubscriptionRegistrationState.Registered.ToString();
            }
            else
            {
                logger.LogInformation($"Subscription is already registered:{resourceEntity}");
                // Validate Tenant id is same
                if (!string.Equals(resourceEntity.RowKey, tenantId))
                {
                    throw new ErrorResponseMessageException(
                        HttpStatusCode.BadRequest,
                        ErrorResponseCode.TenantIdUpdateNotSupported,
                        $"Subscription registered already exists, sub: {subscriptionId}, " +
                        $"tenant: {resourceEntity.RowKey}, cant update tenantId: {tenantId}");
                }

                return;
            }

            TableResult result = await dataStore.InsertOrMergeEntityAsync(tableName, resourceEntity);
            if (!HttpStatusCodeCheck.IsSuccess(result.HttpStatusCode))
            {
                throw new ErrorResponseMessageException(
                        (HttpStatusCode)result.HttpStatusCode,
                        ErrorResponseCode.SubscriptionRegistrationFails,
                        $"Subscription registered failed, sub: {subscriptionId}, tenantId: {tenantId}");
            }

            // TODO - Add Metric event
            logger.LogInformation($"{tableName} - InsertorUpdate entity successful: {result.Result as SubscriptionResourceEntity}");
        }

        /// <summary>
        /// Insert of update into SubscriptionResource table.
        /// </summary>
        /// <param name="subscriptionId">subscriptionId</param>
        /// <param name="tenantId">tenantId</param>
        /// <param name="state">state</param>
        /// <returns></returns>
        public async Task UnregisterSubAsync(
            string subscriptionId)
        {
            // Now, delete entry in resource table
            // TODO check notables have entries related for this subscription.

            SubscriptionResourceEntity resourceEntity = await TryGetSubscriptionResourceAsync(subscriptionId);
            if (resourceEntity == null)
            {
                logger.LogInformation($"Subsctiption :{subscriptionId} doesnt exist, skip unregister");
                return;
            }

            resourceEntity.ETag = "*";
            var result = await this.dataStore.DeleteEntityAsync(tableName, resourceEntity);
            if (!HttpStatusCodeCheck.IsSuccess(result.HttpStatusCode) &&
                !HttpStatusCodeCheck.IsResourceNotFound(result.HttpStatusCode))
            {
                throw new ErrorResponseMessageException(
                    httpStatus: HttpStatusCode.BadRequest,
                    errorCode: ErrorResponseCode.SubscriptionResourceDeleteFailed,
                    errorMessage: $"Failed to delete SubscriptionResourceEntity: {resourceEntity}, ex: {result.HttpStatusCode}.");
            }
        }

        private async Task<SubscriptionResourceEntity> GetSubscriptionResourceAsync(
            string subscriptionId)
        {
            SubscriptionResourceEntity resourceEntity = await TryGetSubscriptionResourceAsync(subscriptionId);
            if (resourceEntity == null)
            {
                throw new ErrorResponseMessageException(
                    HttpStatusCode.BadRequest,
                    ErrorResponseCode.SubscriptionNotRegistered,
                    $"Subscription not registered: {subscriptionId}");
            }

            return resourceEntity;
        }

        private async Task<SubscriptionResourceEntity> TryGetSubscriptionResourceAsync(
            string subscriptionId)
        {
            TableQuery<SubscriptionResourceEntity> tableQuery = new TableQuery<SubscriptionResourceEntity>().Where(
                TableQuery.GenerateFilterCondition(
                    "PartitionKey", QueryComparisons.Equal, subscriptionId));

            ICollection<SubscriptionResourceEntity> resourceEntities = await dataStore.GetEntitiesAsync(tableName, tableQuery);

            if (resourceEntities.Count > 2)
            {
                throw new Exception($"Unexpected SubscriptionResource entries");
            }

            return resourceEntities.FirstOrDefault();
        }
    }
}
