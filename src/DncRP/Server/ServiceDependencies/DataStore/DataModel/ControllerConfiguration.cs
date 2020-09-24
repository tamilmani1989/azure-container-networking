// -----------------------------------------------------------------------------
// <copyright file="ControllerConfiguration.cs" company="Microsoft">
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
    using Microsoft.Extensions.Logging;

    public class ControllerConfiguration : IControllerConfiguration
    {
        private ITableDataStore dataStore;
        private const string resourceTableName = "ControllerResource";
        private const string mdncGuid = "5c7e26be-7ff9-4aaf-9e16-0bc759605aea";
        private const string mdncFirstPartyId = "1efe5bbf-d5b1-4fe9-99fa-f55ce1c88679";
        private const string mdncUri = "https://openai.centralus.dnc.azure.com";
        private ILogger logger;

        /// <summary>
        /// Create an instance of TableOperations
        /// TODO: Pass the DataStore Settings
        /// </summary>
        /// <param name="storageConnectionString">Connection string, configuration from appsettings.Json</param>
        public ControllerConfiguration(
            ITableDataStore tableDataStore,
            ILogger logger)
        {
            this.dataStore = tableDataStore;
            this.logger = logger;

            this.dataStore.CreateTable(resourceTableName);
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
                subId, resourceGroup, controllerName, resourceEntity.Location,
                new ControllerProperties
                {
                    State = TopLevelResourceProperties.ParseProvisioningState(resourceEntity.State),
                    ResourceGuid = resourceEntity.ResouceGuid,
                    PropertiesList = new ControllerPropertiesDetails
                    {
                        DncAppId = mdncFirstPartyId,
                        DncEndpoint = mdncUri
                    }
                });
        }

        /// <summary>
        /// Get DNC controller using ARM controller ID as input
        /// </summary>
        /// <param name="armControllerId"></param>
        /// <returns></returns>
        public async Task<Controller> GetDncControllerAsync(string armControllerId)
        {
            Controller controller = null;
            if (!Controller.TryParseController(armControllerId, out controller))
            {
                return Controller.CreateErrorResponse(
                    "Invalid format for controller id", (int)HttpStatusCode.BadRequest, ErrorResponseCode.ControllerIdInvalidFormat);
            }

            return await this.GetDncControllerAsync(controller.SubscriptionId, controller.ResourceGroup, controller.Name);
        }

        /// <summary>
        /// Create a new DNC controller or replace an existing one.
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="controllerName"></param>
        /// <returns></returns>
        public async Task<Controller> InsertOrReplaceDncControllerAsync(
            string subId, string resourceGroup, string controllerName, ControllerRequest controllerRequest)
        {
            this.logger.LogInformation("Starting PUT controller resource. Sub:{0}, RG:{1}, ResourceName:{2}",
                subId, resourceGroup, controllerName);

            // Create the InsertOrReplace table operation
            string rowKey = $"{resourceGroup}_{controllerName}";

            // Check if row already exists and if it does, reuse ControllerId. Once created,
            // ControllerId can't be changed.
            TableResult result = await this.dataStore.GetEntityAsync<ControllerResourceEntity>(resourceTableName, subId, rowKey);

            string resourceGuid, createdDate;
            if (HttpStatusCodeCheck.IsResourceNotFound(result.HttpStatusCode))
            {
                resourceGuid = Guid.NewGuid().ToString();
                createdDate = DateTime.UtcNow.ToString();
            }
            else
            {
                ControllerResourceEntity entity = result.Result as ControllerResourceEntity;
                resourceGuid = entity.ResouceGuid;
                createdDate = entity.CreatedDateUtc;
            }

            ControllerPropertiesDetails list = new ControllerPropertiesDetails
            {
                DncAppId = mdncFirstPartyId,
                DncEndpoint = mdncUri
            };

            string propJson = JsonUtility.Serialize(list);

            ControllerResourceEntity resourceEntity = new ControllerResourceEntity(subId, rowKey);
            resourceEntity.ResouceGuid = resourceGuid;
            // Succeeded for now. In future, it will be Provisioning in the begining and will
            // become succeeded or failed depending on if dnc deployment succeeds.
            resourceEntity.State = ProvisioningState.Succeeded.ToString();
            resourceEntity.CreatedDateUtc = createdDate;
            resourceEntity.Location = controllerRequest.Properties.Parameters.Location;
            resourceEntity.Properties = propJson;

            // Execute the operation.
            result = await this.dataStore.InsertOrReplaceEntityAsync(resourceTableName, resourceEntity);
            if (!HttpStatusCodeCheck.IsSuccess(result.HttpStatusCode))
            {
                return Controller.CreateErrorResponse("Error in put controller resource", result.HttpStatusCode);
            }

            resourceEntity = result.Result as ControllerResourceEntity;

            return new Controller(
                subId, resourceGroup, controllerName, resourceEntity.Location,
                new ControllerProperties
                {
                    State = TopLevelResourceProperties.ParseProvisioningState(resourceEntity.State),
                    ResourceGuid = resourceEntity.ResouceGuid,
                    PropertiesList = JsonUtility.Deserialize<ControllerPropertiesDetails>(resourceEntity.Properties)
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
            this.logger.LogInformation("Starting DELETE controller resource. Sub:{0}, RG:{1}, ResourceName:{2}",
                subId, resourceGroup, controllerName);

            string resourceId = $"/subscriptions/{subId}/resourceGroups/{resourceGroup}/providers/Microsoft.DelegatedNetwork/controller/{controllerName}";

            // Create Retrieve table operation
            string rowKey = $"{resourceGroup}_{controllerName}";
            TableResult result = await this.dataStore.GetEntityAsync<ControllerResourceEntity>(resourceTableName, subId, rowKey);
            if (HttpStatusCodeCheck.IsResourceNotFound(result.HttpStatusCode))
            {
                return new Controller(subId, resourceGroup, controllerName);
            }

            // Now, delete entry in resource table
            ControllerResourceEntity resourceEntity = result.Result as ControllerResourceEntity;
            result = await this.dataStore.DeleteEntityAsync(resourceTableName, resourceEntity);
            if (!HttpStatusCodeCheck.IsSuccess(result.HttpStatusCode))
            {
                return Controller.CreateErrorResponse("Error in delete controller resource", result.HttpStatusCode);
            }

            return new Controller(subId, resourceGroup, controllerName);
        }
    }
}
