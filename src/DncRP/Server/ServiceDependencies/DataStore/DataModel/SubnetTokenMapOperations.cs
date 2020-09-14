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
            Guid VnetId)
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

                if (VnetId == Guid.Empty)
                {
                    throw new Exception($"Invalid VnetId, empty Guid, delegatedSubnetGuid: {delegatedSubnetGuid}, Subnet: {subnet}");
                }

                // Create the InsertOrReplace table operation
                SubnetTokenMapEntity entity = new SubnetTokenMapEntity(delegatedSubnetGuid, subnet);
                entity.PrimaryTokenId = primaryTokenId;
                entity.SecondaryTokenId = secondaryTokenId;
                entity.VnetId = VnetId;

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
        /// Check if token exists for a given DelegatedSubnet.
        /// </summary>
        /// <param name="delegatedSubnetId"></param>
        /// <param name="subnet"></param>
        /// <returns></returns>
        public async Task<bool> SubnetTokenExistsAsync(
            string delegatedSubnetId,
            string subnet)
        {
            try
            {
                TableResult tableResult = await dataStore.GetEntityAsync<DelegatedSubnetConfigurationEntity>(tableName, delegatedSubnetId, subnet);
                SubnetTokenMapEntity subnetConfigurationEntity = tableResult.Result as SubnetTokenMapEntity;

                if (subnetConfigurationEntity == null)
                {
                    logger.LogInformation($"Subnet token doesnt exist for DelegatedSubnetId: {delegatedSubnetId}, SubnetName: {subnet}");
                    return false;
                }

                return true;
            }
            catch (Exception ex)
            {
                logger.LogError($"Failed to inertorupdate entity in {tableName}, {ex}");
                throw;
            }
        }
    }
}
