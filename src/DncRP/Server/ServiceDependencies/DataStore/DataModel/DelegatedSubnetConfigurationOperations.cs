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
    using Microsoft.Azure.ContainerNetworking.CoreLib;
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
        /// <param name="resourceName">resourceName</param>
        /// <param name="delegatedSubnetGuid">DelegatedSubnet Resource Guid</param>
        /// <param name="vnetResourceGuid">VnetResourceGuid</param>
        /// <param name="DelegateSubnetRequest">DelegateSubnetRequest</param>
        /// <returns></returns>
        public async Task<DelegateSubnet> InsertOrUpdateAsync(
            string subscriptionId,
            string resourceGroupName,
            string resourceName,
            string delegatedSubnetGuid,
            string vnetResourceGuid,
            string subnetJoinState,
            DelegateSubnetRequest delegateSubnetRequest)
        {
            // Cant add with invalid delegatedSubnetId
            if (string.IsNullOrEmpty(delegatedSubnetGuid) ||
                string.Equals(delegatedSubnetGuid, Guid.Empty.ToString()))
            {
                throw new Exception($"Invalid delegatedSubnetGuid, it is either null or empty Guid, subscriptionId: {subscriptionId}, resourceGroupName :{resourceGroupName}, resourceName: {resourceName} ");
            }

            // Cant add with invalid delegatedSubnetId
            if (string.IsNullOrEmpty(vnetResourceGuid) ||
                string.Equals(vnetResourceGuid, Guid.Empty.ToString()))
            {
                // TODO: Add Exceptions for Data operations
                throw new Exception($"Invalid vnetResourceGuid, it is either null or empty Guid, subscriptionId: {subscriptionId}, resourceGroupName :{resourceGroupName}, resourceName: {resourceName}");
            }

            string associatedSubnetId = delegateSubnetRequest.Properties.Parameters.Properties.SubnetDetails.Id;
            string controllerId = delegateSubnetRequest.Properties.Parameters.Properties.ControllerDetails.Id;

            // Check if entry already exists.
            string rowKey = DelegatedSubnetResourceEntity.GetRowKey(resourceGroupName, resourceName);
            TableResult tableResult = await dataStore.GetEntityAsync<DelegatedSubnetResourceEntity>(tableName, subscriptionId, rowKey);
            DelegatedSubnetResourceEntity resourceEntity = tableResult.Result as DelegatedSubnetResourceEntity;

            if (resourceEntity == null)
            {
                logger.CLogInformation($"Creating the existing DelegatedSubnetConfigurationEntity:{resourceEntity}");
                resourceEntity = new DelegatedSubnetResourceEntity(subscriptionId, resourceGroupName, resourceName);
                resourceEntity.CreatedDateUtc = DateTime.UtcNow;
                resourceEntity.VnetResourceGuid = vnetResourceGuid;
                resourceEntity.ResourceGuid = delegatedSubnetGuid;
                resourceEntity.AssociatedSubnetId = associatedSubnetId;
                resourceEntity.ControllerId = controllerId;
            }
            else
            {
                // For now block throw in case AssociatedSubnet or ControllerId has changed
                if (!string.Equals(resourceEntity.AssociatedSubnetId, associatedSubnetId, StringComparison.InvariantCultureIgnoreCase))
                {
                    throw new ErrorResponseMessageException(
                        httpStatus: HttpStatusCode.BadRequest,
                        errorCode: ErrorResponseCode.AssociatedSubnetIdChanged,
                        errorMessage: $"DelegateSubnetResource: AssociatedSubnet is changed, existing: {resourceEntity.AssociatedSubnetId}, new: {associatedSubnetId}, " +
                        $"entity: {resourceEntity}, requestbody: {delegateSubnetRequest}.");
                }

                if (!string.Equals(resourceEntity.ControllerId, controllerId, StringComparison.InvariantCultureIgnoreCase))
                {
                    throw new ErrorResponseMessageException(
                        httpStatus: HttpStatusCode.BadRequest,
                        errorCode: ErrorResponseCode.ControllerIdChanged,
                        errorMessage: $"DelegateSubnetResource: Associated Controller id is changed, existing: {resourceEntity.ControllerId}, new: {controllerId}, " +
                        $"entity: {resourceEntity}, requestbody: {delegateSubnetRequest}.");
                }

                // TODO: Validate AssociatedSubnet is not attached to any other subnet.

                logger.CLogInformation($"Updating the existing DelegatedSubnetConfigurationEntity:{resourceEntity}");
            }

            // todo - Add this field
            resourceEntity.SubnetJoinState = subnetJoinState;
            resourceEntity.State = ProvisioningState.Succeeded.ToString();
            resourceEntity.Location = delegateSubnetRequest.Properties.Parameters.Location;

            TableResult result = await dataStore.InsertOrMergeEntityAsync(tableName, resourceEntity);
            DelegatedSubnetResourceEntity resultEntity = result.Result as DelegatedSubnetResourceEntity;

            // TODO - Add Metric event
            logger.CLogInformation($"{tableName} - InsertorUpdate entity: {resultEntity}");

            return resourceEntity.ConvertToDelegateSubnet();
        }

        /// <summary>
        /// Try get the DelegatedSubnet resource if an entity exists for a given Subnet.
        /// </summary>
        /// <param name="subscriptionId">subscription id</param>
        /// <param name="resourceGroupName">resourcegroup name</param>
        /// <param name="resourceName">resourcename</param>
        /// <returns></returns>
        public async Task<DelegateSubnet> TryGetDelegatedSubnetConfigurationAsync(
            string subscriptionId,
            string resourceGroupName,
            string resourceName)
        {
            string rowKey = DelegatedSubnetResourceEntity.GetRowKey(resourceGroupName, resourceName);
            TableResult tableResult = await dataStore.GetEntityAsync<DelegatedSubnetResourceEntity>(tableName, subscriptionId, rowKey);
            DelegatedSubnetResourceEntity resultEntity = tableResult.Result as DelegatedSubnetResourceEntity;

            if (resultEntity == null)
            {
                logger.CLogInformation($"TryGetDelegatedSubnetConfigurationAsync: ResourceNotFound, subscriptionId:{subscriptionId}, resourceGroupName: {resourceGroupName}, " +
                    $"resourceName: {resourceName}.");
                return null;
            }

            return resultEntity.ConvertToDelegateSubnet();
        }

        /// <summary>
        /// Delete DelegateSubnetResource entity if exists.
        /// </summary>
        /// <param name="delegatedSubnetId"></param>
        /// <param name="subnet"></param>
        /// <returns></returns>
        public async Task DeleteAsync(
            string subscriptionId,
            string resourceGroupName,
            string resourceName)
        {
            DelegatedSubnetResourceEntity resourceEntity = new DelegatedSubnetResourceEntity(subscriptionId, resourceGroupName, resourceName);
            resourceEntity.ETag = "*";
            var result = await this.dataStore.DeleteEntityAsync(tableName, resourceEntity);
            if (!HttpStatusCodeCheck.IsSuccess(result.HttpStatusCode) &&
                !HttpStatusCodeCheck.IsResourceNotFound(result.HttpStatusCode))
            {
                throw new ErrorResponseMessageException(
                    httpStatus: HttpStatusCode.BadRequest,
                    errorCode: ErrorResponseCode.SubnetTokenMapDeleteFailed,
                    errorMessage: $"Failed to delete DelegatedSubnetResourceEntity: {resourceEntity}, ex: {result.HttpStatusCode}.");
            }
        }
    }
}
