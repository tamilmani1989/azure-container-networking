// -----------------------------------------------------------------------------
// <copyright file="ControllerConfiguration.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Threading.Tasks;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Azure.Cosmos.Table;

    internal class ControllerConfiguration : IControllerConfiguration
    {
        private ITableDataStore dataStore;
        private const string resourceTableName = "ControllerResource";
        private const string configTableName = "ControllerConfiguration";
        private const string mdncId = "5c7e26be-7ff9-4aaf-9e16-0bc759605aea";
        private const string mdncUri = "https://openai.centralus.dnc.azure.com";

        /// <summary>
        /// Create an instance of TableOperations
        /// TODO: Pass the DataStore Settings
        /// </summary>
        /// <param name="storageConnectionString">Connection string, configuration from appsettings.Json</param>
        public ControllerConfiguration(
            ITableDataStore tableDataStore)
        {
            this.dataStore = tableDataStore;

            this.dataStore.CreateTable(resourceTableName);
            this.dataStore.CreateTable(configTableName);
        }

        /// <summary>
        /// Get DNC controller
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="controllerName"></param>
        /// <returns></returns>
        public async Task<Controller> GetDncControllerAsync(string subId, string resourceGroup, string controllerName)
        {
            string resourceId = $"/subscriptions/{subId}/resourceGroups/{resourceGroup}/providers/Microsoft.DelegatedNetwork/controller/{controllerName}";

            // Retrieve table operation
            string rowKey = $"{resourceGroup}_{controllerName}";
            TableResult result = await this.dataStore.GetEntityAsync<ControllerResourceEntity>(resourceTableName, subId, rowKey);
            if (!HttpStatusCodeCheck.IsSuccess(result.HttpStatusCode))
            {
                string msg = HttpStatusCodeCheck.IsResourceNotFound(result.HttpStatusCode)
                    ? $"Resource '{resourceId}' does not exist"
                    : string.Empty;

                return Controller.CreateErrorResponse(msg, result.HttpStatusCode);
            }

            ControllerResourceEntity resourceEntity = result.Result as ControllerResourceEntity;

            return new Controller(
                subId, resourceGroup, controllerName,
                new ControllerProperties
                {
                    State = this.ToRequestState(resourceEntity.State),
                    DncAppId = mdncId,
                    DncEndpoint = mdncUri
                });
        }

        /// <summary>
        /// Create a new DNC controller or replace an existing one. It works in two step.
        /// a. First insert/replace an entity into ControllerResouce table
        /// b. Next, insert/replace an entity into ControllerConfiguration table
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="controllerName"></param>
        /// <returns></returns>
        public async Task<Controller> InsertOrReplaceDncControllerAsync(
            string subId, string resourceGroup, string controllerName, ControllerRequest controllerRequest)
        {
            // Create the InsertOrReplace table operation
            string rowKey = $"{resourceGroup}_{controllerName}";

            // Check if row already exists and if it does, reuse ControllerId. Once created,
            // ControllerId can't be changed.
            TableResult result = await this.dataStore.GetEntityAsync<ControllerResourceEntity>(resourceTableName, subId, rowKey);

            string controllerGuid, createdDate;
            if (HttpStatusCodeCheck.IsResourceNotFound(result.HttpStatusCode))
            {
                controllerGuid = Guid.NewGuid().ToString();
                createdDate = DateTime.UtcNow.ToString();
            }
            else
            {
                ControllerResourceEntity cre = result.Result as ControllerResourceEntity;
                controllerGuid = cre.ControllerId;
                createdDate = cre.CreatedDateUtc;
            }

            string propJson = JsonUtility.Serialize(controllerRequest.Parameters.Properties);

            ControllerResourceEntity resourceEntity = new ControllerResourceEntity(subId, rowKey);
            resourceEntity.ControllerId = controllerGuid;
            resourceEntity.ManagedDNCId = mdncId;
            resourceEntity.ControllerType = controllerRequest.Parameters.Type.ToString();
            resourceEntity.Properties = propJson;
            resourceEntity.State = ProvisioningState.Succeeded.ToString();
            resourceEntity.CreatedDateUtc = createdDate;

            // Execute the operation.
            result = await this.dataStore.InsertOrReplaceEntityAsync(resourceTableName, resourceEntity);
            if (!HttpStatusCodeCheck.IsSuccess(result.HttpStatusCode))
            {
                return Controller.CreateErrorResponse("Error in insert controller resource", result.HttpStatusCode);
            }

            resourceEntity = result.Result as ControllerResourceEntity;

            // insert entity into controller config table
            int httpCode = await this.InsertOrReplaceControllerConfigAsync(mdncId, controllerGuid, controllerRequest.Parameters.Type, propJson);
            if (!HttpStatusCodeCheck.IsSuccess(httpCode))
            {
                return Controller.CreateErrorResponse("Error in insert controller config", httpCode);
            }

            return new Controller(
               subId, resourceGroup, controllerName,
               new ControllerProperties
               {
                   State = this.ToRequestState(resourceEntity.State),
                   DncAppId = mdncId,
                   DncEndpoint = mdncUri
               });
        }

        /// <summary>
        /// Modify an existing controller
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="controllerName"></param>
        /// <returns></returns>
        public Task<Controller> MergeDncControllerAsync(
           string subId, string resourceGroup, string controllerName, ControllerRequest controllerRequest)
        {
            throw new NotSupportedException();
        }

        /// <summary>
        /// Delete works in two step.
        /// a. First state of controller is set to Deleting and response is returned to caller as Accepted(202)
        /// b. A background worker pick up deleting entries and first deletes them from ControllerConfiguration
        /// table and then from ControllerResouce table
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="controllerName"></param>
        /// <returns></returns>
        public async Task<Controller> DeleteDncControllerAsync(string subId, string resourceGroup, string controllerName)
        {
            string resourceId = $"/subscriptions/{subId}/resourceGroups/{resourceGroup}/providers/Microsoft.DelegatedNetwork/controller/{controllerName}";

            // Create Retrieve table operation
            string rowKey = $"{resourceGroup}_{controllerName}";
            TableResult result = await this.dataStore.GetEntityAsync<ControllerResourceEntity>(resourceTableName, subId, rowKey);
            if (!HttpStatusCodeCheck.IsSuccess(result.HttpStatusCode))
            {
                string msg = HttpStatusCodeCheck.IsResourceNotFound(result.HttpStatusCode)
                    ? $"Resource '{resourceId}' does not exist"
                    : string.Empty;

                return Controller.CreateErrorResponse(msg, result.HttpStatusCode);
            }

            // First delete entry in config table. If resource not found in config table, that is not treated as error
            ControllerResourceEntity resourceEntity = result.Result as ControllerResourceEntity;
            await this.DeleteControllerConfigAsync(resourceEntity.ManagedDNCId, resourceEntity.ControllerId);
            if (!HttpStatusCodeCheck.IsSuccess(result.HttpStatusCode) &&
                !HttpStatusCodeCheck.IsResourceNotFound(result.HttpStatusCode))
            {
                return Controller.CreateErrorResponse("Error in delete controller config", result.HttpStatusCode);
            }

            // Now, delete entry in resource table
            result = await this.dataStore.DeleteEntityAsync(resourceTableName, resourceEntity);
            if (!HttpStatusCodeCheck.IsSuccess(result.HttpStatusCode))
            {
                return Controller.CreateErrorResponse("Error in delete controller resource", result.HttpStatusCode);
            }

            return new Controller(
                subId, resourceGroup, controllerName,
                new ControllerProperties
                {
                    State = ProvisioningState.Succeeded,
                    DncAppId = mdncId,
                    DncEndpoint = mdncUri
                });
        }

        /// <summary>
        /// Insert or replace an entity in Controller config table
        /// </summary>
        /// <param name="mdncId"></param>
        /// <param name="controllerId"></param>
        /// <param name="type"></param>
        /// <param name="propertiesJson"></param>
        /// <returns></returns>
        private async Task<int> InsertOrReplaceControllerConfigAsync(
            string mdncId, string controllerId, ControllerType type, string propertiesJson)
        {
            ControllerConfigurationEntity entity = new ControllerConfigurationEntity(mdncId, controllerId);
            entity.ControllerType = type.ToString();
            entity.Properties = propertiesJson;

            // Execute Delete table operation
            TableResult result = await this.dataStore.InsertOrReplaceEntityAsync(configTableName, entity);
            return result.HttpStatusCode;
        }

        /// <summary>
        /// Insert or replace an entity in Controller config table
        /// </summary>
        /// <param name="mdncId"></param>
        /// <param name="controllerId"></param>
        /// <param name="type"></param>
        /// <param name="propertiesJson"></param>
        /// <returns></returns>
        private async Task<int> DeleteControllerConfigAsync(
            string mdncId, string controllerId)
        {
            TableResult result = await this.dataStore.GetEntityAsync<ControllerConfigurationEntity>(configTableName, mdncId, controllerId);
            if (!HttpStatusCodeCheck.IsSuccess(result.HttpStatusCode))
            {
                return result.HttpStatusCode;
            }

            ControllerConfigurationEntity entity = new ControllerConfigurationEntity(mdncId, controllerId);
            entity.ETag = "*";

            // Execute Delete table operation
            result = await this.dataStore.DeleteEntityAsync(configTableName, entity);
            return result.HttpStatusCode;
        }

        /// <summary>
        /// Convert state to enum
        /// </summary>
        /// <param name="state"></param>
        /// <returns></returns>
        private ProvisioningState ToRequestState(string state)
        {
            return (ProvisioningState)Enum.Parse(typeof(ProvisioningState), state, true);
        }
    }
}
