// -----------------------------------------------------------------------------
// <copyright file="SubnetTokenMapOperations.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Threading.Tasks;
    using Microsoft.Azure.Cosmos.Table;
    using Microsoft.Extensions.Logging;

    /// <summary>
    /// DataModel for SubnetTokenConfiguration operations
    /// </summary>
    public class SubnetTokenMapOperations : ISubnetTokenMapOperations
    {
        private ITableDataStore dataStore;
        private ILogger logger; 
        private const string tableName = "SubnetTokenMap";

        /// <summary>
        /// Create an instance of TableOperations
        /// TODO: Pass the DataStore Settings
        /// </summary>
        /// <param name="storageConnectionString">Connection string, configuration from appsettings.Json</param>
        public SubnetTokenMapOperations(
            ITableDataStore tableDataStore,
            ILogger logger)
        {
            this.dataStore = tableDataStore;
            this.logger = logger;

            dataStore.CreateTable(tableName);
        }

        /// <summary>
        /// Insert of update into SubnetTokenConfiguration table.
        /// </summary>
        /// <param name="delegatedSubnetId">delegatedSubnet ResourceId</param>
        /// <param name="subnetName"></param>
        /// <param name="primaryTokenId"></param>
        /// <param name="secondaryTokenId"></param>
        /// <param name="VnetId"></param>
        public async Task InsertOrUpdateAsync(
            string delegatedSubnetGuid,
            string subnet,
            string primaryTokenId,
            string secondaryTokenId,
            string vnetResourceGuid)
        {
            try
            {
                // We cant add a SubnetTokenMap with empty PrimaryTokenId
                if (string.IsNullOrEmpty(primaryTokenId) ||
                    string.Equals(primaryTokenId, Guid.Empty.ToString()))
                {
                    // TODO: Add Exceptions for Data operations
                    throw new Exception($"Invalid PrimaryTokenId, it is either null or empty Guid, delegatedSubnetGuid: {delegatedSubnetGuid}, Subnet: {subnet}");
                }

                if (string.IsNullOrEmpty(vnetResourceGuid) ||
                    string.Equals(vnetResourceGuid, Guid.Empty.ToString()))
                {
                    throw new Exception($"Invalid VnetId, empty Guid, delegatedSubnetGuid: {delegatedSubnetGuid}, Subnet: {subnet}");
                }

                // Check if entry already exists.
                SubnetTokenMapEntity subnetTokenMapEntity = await TryGetSubnetTokenMapAsync(delegatedSubnetGuid, subnet);

                if (subnetTokenMapEntity == null)
                {
                    subnetTokenMapEntity = new SubnetTokenMapEntity(delegatedSubnetGuid, subnet);
                    subnetTokenMapEntity.CreatedDateUtc = DateTime.UtcNow;
                }
                else
                {
                    logger.LogInformation($"Updating the existing SubnetTokenMapEntity:{subnetTokenMapEntity}");
                }

                subnetTokenMapEntity.PrimaryTokenId = primaryTokenId;
                subnetTokenMapEntity.SecondaryTokenId = secondaryTokenId;
                subnetTokenMapEntity.VnetResourceGuid = vnetResourceGuid;

                TableResult result = await dataStore.InsertOrMergeEntityAsync(tableName, subnetTokenMapEntity);

                // TODO - Add Metric event
                logger.LogInformation($"{tableName} - InsertorUpdate entity successful: {result.Result as SubnetTokenMapEntity}");
            }
            catch (Exception ex)
            {
                logger.LogError($"Failed to inertorupdate entity in {tableName}, {ex}");
                throw;
            }
        }

        /// <summary>
        /// Check if token exists for a given DelegatedSubnet.
        /// </summary>
        /// <param name="delegatedSubnetId"></param>
        /// <param name="subnet"></param>
        /// <returns></returns>
        public async Task<bool> SubnetTokenExistsAsync(
            string delegatedSubnetGuid,
            string subnet)
        {
            try
            {
                SubnetTokenMapEntity subnetEntity = await TryGetSubnetTokenMapAsync(delegatedSubnetGuid, subnet);
                return subnetEntity != null? true:false;
            }
            catch (Exception ex)
            {
                logger.LogError($"Failed to inertorupdate entity in {tableName}, {ex}");
                throw;
            }
        }

        private async Task<SubnetTokenMapEntity> TryGetSubnetTokenMapAsync(
            string delegatedSubnetGuid,
            string subnet)
        {
            try
            {
                TableResult tableResult = await dataStore.GetEntityAsync<DelegatedSubnetConfigurationEntity>(tableName, delegatedSubnetGuid, subnet);
                SubnetTokenMapEntity subnetTokenMapEntity = tableResult.Result as SubnetTokenMapEntity;

                if (subnetTokenMapEntity == null)
                {
                    logger.LogInformation($"Subnet token doesnt exist for DelegatedSubnetId: {delegatedSubnetGuid}, SubnetName: {subnet}");
                    return null;
                }

                return subnetTokenMapEntity;
            }
            catch (Exception ex)
            {
                logger.LogError($"Failed to inertorupdate entity in {tableName}, {ex}");
                throw;
            }
        }
    }
}
