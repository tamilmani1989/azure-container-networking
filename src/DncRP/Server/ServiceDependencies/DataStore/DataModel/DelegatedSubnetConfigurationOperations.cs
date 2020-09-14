// -----------------------------------------------------------------------------
// <copyright file="DelegatedSubnetConfigurationOperations.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Threading.Tasks;
    using Microsoft.Azure.Cosmos.Table;
    using Microsoft.Azure.Documents.SystemFunctions;
    using Microsoft.Extensions.Logging;

    /// <summary>
    /// DataModel for DelegatedSubnetConfiguration operations
    /// </summary>
    public class DelegatedSubnetConfigurationOperations : IDelegatedSubnetConfigurationOperations
    {
        private ITableDataStore dataStore;
        private ILogger logger; 
        private const string tableName = "DelegatedSubnetConfiguration";

        /// <summary>
        /// Create an instance of TableOperations
        /// TODO: Pass the DataStore Settings
        /// </summary>
        /// <param name="storageConnectionString">Connection string, configuration from appsettings.Json</param>
        public DelegatedSubnetConfigurationOperations(
            ITableDataStore tableDataStore,
            ILogger logger)
        {
            this.dataStore = tableDataStore;
            this.logger = logger;

            dataStore.CreateTable(tableName);
        }

        /// <summary>
        /// Insert or update DelegatedSubnetConfigurationEntity
        /// </summary>
        /// <param name="subscriptionId">subscriptionId</param>
        /// <param name="resourceGroupName">resourceGroupName</param>
        /// <param name="vnetName">VirtualNetwork name</param>
        /// <param name="subnet">subnetname</param>
        /// <param name="delegatedSubnetName">DelegatedSubnetId</param>
        /// <param name="delegatedSubnetGuid">DelegatedSubnet Guid</param>
        /// <returns></returns>
        public async Task InsertOrUpdateAsync(
            string subscriptionId,
            string resourceGroupName,
            string vnetName,
            string subnet,
            string delegatedSubnetName,
            string delegatedSubnetGuid)
        {
            try
            {
                // Cant add with invalid delegatedSubnetId
                if (string.IsNullOrEmpty(delegatedSubnetGuid) ||
                    string.Equals(delegatedSubnetGuid, Guid.Empty.ToString()))
                {
                    // TODO: Add Exceptions for Data operations
                    throw new Exception($"Invalid PrimaryTokenId, it is either null or empty Guid, delegatedSubnetGuid: {delegatedSubnetGuid}, Subnet: {subnet}");
                }

                // Create the InsertOrReplace table operation
                DelegatedSubnetConfigurationEntity entity = new DelegatedSubnetConfigurationEntity(subscriptionId, resourceGroupName, vnetName, subnet);
                entity.ResourceName = delegatedSubnetName;
                entity.ResourceGuid = delegatedSubnetGuid;

                await dataStore.InsertOrMergeEntityAsync(tableName, entity);

                // TODO - Add Metric event
                logger.LogInformation($"{tableName} - InsertorUpdate entity: {entity}");
            }
            catch (Exception ex)
            {
                logger.LogError($"Failed to inertorupdate entity in {tableName}, {ex}");
                throw;
            }
        }

        /// <summary>
        /// Get the DelegatedSubnet Guid if an entity exists for a given Subnet.
        /// </summary>
        /// <param name="subscriptionId">subscription id</param>
        /// <param name="resourceGroupName">resourcegroup name</param>
        /// <param name="vnetName">vnetName</param>
        /// <param name="subnet">subnet</param>
        /// <returns></returns>
        public async Task<string> TryGetDelegatedSubnetGuidAsync(
            string subscriptionId,
            string resourceGroupName,
            string vnetName,
            string subnet)
        {
            try
            {
                string rowKey = DelegatedSubnetConfigurationEntity.GetSubnetConfigurationKey(resourceGroupName, vnetName, subnet);
                TableResult tableResult = await dataStore.GetEntityAsync<DelegatedSubnetConfigurationEntity>(tableName, subscriptionId, rowKey);
                DelegatedSubnetConfigurationEntity resultEntity = tableResult.Result as DelegatedSubnetConfigurationEntity;

                if (resultEntity == null)
                {
                    logger.LogInformation($"DelegatedSubnet configuration doesnt exist for subscription: {subscriptionId}, rowKey: {rowKey}");
                    return null;
                }

                return resultEntity.ResourceGuid;
            }
            catch (Exception ex)
            {
                logger.LogError($"Failed to inertorupdate entity in {tableName}, {ex}");
                throw;
            }
        }
    }
}
