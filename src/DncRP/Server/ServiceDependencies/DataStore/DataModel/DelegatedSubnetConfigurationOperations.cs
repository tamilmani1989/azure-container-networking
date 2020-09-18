// -----------------------------------------------------------------------------
// <copyright file="DelegatedSubnetConfigurationOperations.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Net;
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
        private const string tableName = "DelegatedSubnetResource";

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
        /// <param name="vnetResourceGuid">VnetResourceGuid</param>
        /// <returns></returns>
        public async Task<DelegateSubnet> InsertOrUpdateAsync(
            string subscriptionId,
            string resourceGroupName,
            string vnetName,
            string subnet,
            string resourceName,
            string delegatedSubnetGuid,
            string vnetResourceGuid)
        {
            try
            {
                // Cant add with invalid delegatedSubnetId
                if (string.IsNullOrEmpty(delegatedSubnetGuid) ||
                    string.Equals(delegatedSubnetGuid, Guid.Empty.ToString()))
                {
                    throw new Exception($"Invalid delegatedSubnetGuid, it is either null or empty Guid, subscriptionId: {subscriptionId}, Subnet: {subnet}");
                }

                // Cant add with invalid delegatedSubnetId
                if (string.IsNullOrEmpty(vnetResourceGuid) ||
                    string.Equals(vnetResourceGuid, Guid.Empty.ToString()))
                {
                    // TODO: Add Exceptions for Data operations
                    throw new Exception($"Invalid vnetResourceGuid, it is either null or empty Guid, subscriptionId: {subscriptionId}, Subnet: {subnet}");
                }

                // Check if entry already exists.
                string rowKey = DelegatedSubnetConfigurationEntity.GetSubnetConfigurationKey(resourceGroupName, vnetName, subnet);
                TableResult tableResult = await dataStore.GetEntityAsync<DelegatedSubnetConfigurationEntity>(tableName, subscriptionId, rowKey);
                DelegatedSubnetConfigurationEntity delegatedSubnetConfigurationEntity = tableResult.Result as DelegatedSubnetConfigurationEntity;

                if (delegatedSubnetConfigurationEntity == null)
                {
                    delegatedSubnetConfigurationEntity = new DelegatedSubnetConfigurationEntity(subscriptionId, resourceGroupName, vnetName, subnet);
                    delegatedSubnetConfigurationEntity.CreatedDateUtc = DateTime.UtcNow;
                    delegatedSubnetConfigurationEntity.VnetResourceGuid = vnetResourceGuid;
                    delegatedSubnetConfigurationEntity.ResourceName = resourceName;
                }
                else
                {
                    if (!string.Equals(delegatedSubnetConfigurationEntity.ResourceName, resourceName, StringComparison.InvariantCultureIgnoreCase))
                    {
                        throw new ErrorResponseMessageException(
                            httpStatus: HttpStatusCode.NotFound,
                            errorCode: ErrorResponseCode.SubnetAlreadyDelegatedToDifferentResource,
                            errorMessage: $"This Subnet is already delegated to Resource: {delegatedSubnetConfigurationEntity.ResourceName}");
                    }
                    logger.LogInformation($"Updating the existing DelegatedSubnetConfigurationEntity:{delegatedSubnetConfigurationEntity}");
                }

                // todo - Add this field
                delegatedSubnetConfigurationEntity.ProvisioningStatus = "";
                delegatedSubnetConfigurationEntity.DelegatedSubnetGuid = delegatedSubnetGuid;

                // todo - This will contain the ControllerId info
                delegatedSubnetConfigurationEntity.Properties = "";

                TableResult result = await dataStore.InsertOrMergeEntityAsync(tableName, delegatedSubnetConfigurationEntity);
                DelegatedSubnetConfigurationEntity resultEntity = result.Result as DelegatedSubnetConfigurationEntity;

                // TODO - Add Metric event
                logger.LogInformation($"{tableName} - InsertorUpdate entity: {resultEntity}");

                return new DelegateSubnet(
                    resultEntity.PartitionKey,
                    resultEntity.ResourceGroup,
                    resultEntity.VnetName,
                    resultEntity.Subnet,
                    resultEntity.ResourceName,
                    resultEntity.DelegatedSubnetGuid)
                {
                    VNetResourceGuid = resultEntity.VnetResourceGuid
                };
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
        /// <param name="resourceName">resourcename</param>
        /// <param name="throwOnResourceNameComparison">set if need to compare the DelegatedResource Name as well</param>
        /// <returns></returns>
        public async Task<DelegateSubnet> TryGetDelegatedSubnetConfigurationAsync(
            string subscriptionId,
            string resourceGroupName,
            string vnetName,
            string subnet,
            string resourceName,
            bool throwOnResourceNameComparison = false)
    {
            string rowKey = DelegatedSubnetConfigurationEntity.GetSubnetConfigurationKey(resourceGroupName, vnetName, subnet);
            TableResult tableResult = await dataStore.GetEntityAsync<DelegatedSubnetConfigurationEntity>(tableName, subscriptionId, rowKey);
            DelegatedSubnetConfigurationEntity resultEntity = tableResult.Result as DelegatedSubnetConfigurationEntity;

            if (resultEntity == null)
            {
                logger.LogInformation($"DelegatedSubnet configuration doesnt exist for subscription: {subscriptionId}, rowKey: {rowKey}");
                return null;
            }

            if (!string.Equals(resultEntity.ResourceName, resourceName, StringComparison.InvariantCultureIgnoreCase))
            {
                logger.LogInformation($"DelegatedSubnet is assigned to a different resourceName: {resultEntity}, expected ResourceName: {resourceName}");
                if (throwOnResourceNameComparison)
                {
                    throw new ErrorResponseMessageException(
                        httpStatus: HttpStatusCode.BadRequest,
                        errorCode: ErrorResponseCode.SubnetAlreadyDelegatedToDifferentResource,
                        errorMessage: $"This Subnet is already delegated to Resource: {resultEntity.ResourceName}");
                }

                return null;
            }

            return new DelegateSubnet(
                resultEntity.PartitionKey,
                resultEntity.ResourceGroup,
                resultEntity.VnetName,
                resultEntity.Subnet,
                resultEntity.ResourceName,
                resultEntity.DelegatedSubnetGuid)
            {
                VNetResourceGuid = resultEntity.VnetResourceGuid
            };
        }
    }
}
