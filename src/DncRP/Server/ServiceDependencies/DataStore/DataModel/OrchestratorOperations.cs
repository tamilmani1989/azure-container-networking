// -----------------------------------------------------------------------------
// <copyright file="OrchestratorOperations.cs" company="Microsoft">
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

    public class OrchestratorOperations : IOrchestratorOperations
    {
        private ITableDataStore dataStore;
        private const string resourceTableName = "OrchestratorInstanceResource";
        private const string configTableName = "OrchestratorInstancesConfiguration";
        private ILogger logger;
        private IControllerConfiguration controllerOperations;

        /// <summary>
        /// Create an instance of OrchestratorOperations
        /// </summary>
        /// <param name="tableDataStore"></param>
        /// <param name="logger"></param>
        public OrchestratorOperations(
            ITableDataStore tableDataStore,
            ILogger logger,
            IControllerConfiguration controllerConfiguration)
        {
            this.dataStore = tableDataStore;
            this.logger = logger;
            this.controllerOperations = controllerConfiguration;

            this.dataStore.CreateTable(resourceTableName);
            this.dataStore.CreateTable(configTableName);
        }

        /// <summary>
        /// Get Orchestrator
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="resourceName"></param>
        /// <returns></returns>
        public async Task<Orchestrator> GetOrchestratorAsync(string subId, string resourceGroup, string resourceName)
        {
            // Retrieve table operation
            string rowKey = $"{resourceGroup}_{resourceName}";
            TableResult result = await this.dataStore.GetEntityAsync<OrchestratorResourceEntity>(resourceTableName, subId, rowKey);
            if (!HttpStatusCodeCheck.IsSuccess(result.HttpStatusCode))
            {
                Orchestrator orchestrator = new Orchestrator(subId, resourceGroup, resourceName);
                string msg = HttpStatusCodeCheck.IsResourceNotFound(result.HttpStatusCode)
                    ? $"Resource '{orchestrator.Id}' does not exist"
                    : string.Empty;

                return Orchestrator.CreateErrorResponse(msg, result.HttpStatusCode);
            }

            OrchestratorResourceEntity resourceEntity = result.Result as OrchestratorResourceEntity;
            return resourceEntity.ToOrchestrator();
        }

        /// <summary>
        /// Create a new Orchestrator or replace an existing one
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="resourceName"></param>
        /// <param name="orchestratorRequest"></param>
        /// <returns></returns>
        public async Task<Orchestrator> InsertOrReplaceOrchestratorAsync(
            string subId, string resourceGroup, string resourceName, OrchestratorRequest orchestratorRequest)
        {
            this.logger.LogInformation("Starting PUT orchestrator resource. Sub:{0}, RG:{1}, ResourceName:{2}",
                subId, resourceGroup, resourceName);

            string controllerArmResouceId = orchestratorRequest.Properties.Parameters.Properties.ControllerDetails.Id;
            Controller controller = await this.controllerOperations.GetDncControllerAsync(controllerArmResouceId);
            if (controller.Error != null)
            {
                return Orchestrator.CreateErrorResponse(
                    controller.Error.Error, controller.Error.StatusCode, controller.Error.ErrorResponseCode);
            }

            // Check if row already exists and if it does, reuse resourceGuid.
            string rowKey = $"{resourceGroup}_{resourceName}";
            TableResult result = await this.dataStore.GetEntityAsync<OrchestratorResourceEntity>(resourceTableName, subId, rowKey);

            string resourceGuid, createdDate;
            if (HttpStatusCodeCheck.IsResourceNotFound(result.HttpStatusCode))
            {
                resourceGuid = Guid.NewGuid().ToString();
                createdDate = DateTime.UtcNow.ToString();
            }
            else
            {
                OrchestratorResourceEntity entity = result.Result as OrchestratorResourceEntity;
                resourceGuid = entity.ResouceGuid;
                createdDate = entity.CreatedDateUtc;
            }

            string propJson = JsonUtility.Serialize(orchestratorRequest.Properties.Parameters.Properties.orchestratorInstanceDetails.Properties);
            OrchestratorType orchestratorType = orchestratorRequest.Properties.Parameters.Properties.orchestratorInstanceDetails.Type;
            ControllerProperties controllerProps = (ControllerProperties)controller.Properties;

            OrchestratorResourceEntity resourceEntity = new OrchestratorResourceEntity(subId, rowKey);
            resourceEntity.ResouceGuid = resourceGuid;
            resourceEntity.OrchestratorType = orchestratorType.ToString();
            resourceEntity.Properties = propJson;
            resourceEntity.State = ProvisioningState.Succeeded.ToString();
            resourceEntity.CreatedDateUtc = createdDate;
            resourceEntity.ControllerId = controllerArmResouceId;
            resourceEntity.Location = orchestratorRequest.Properties.Parameters.Location;
            resourceEntity.ControllerProperties = JsonUtility.Serialize<ControllerProperties>(controllerProps);

            // Execute the operation.
            result = await this.dataStore.InsertOrReplaceEntityAsync(resourceTableName, resourceEntity);
            if (!HttpStatusCodeCheck.IsSuccess(result.HttpStatusCode))
            {
                return Orchestrator.CreateErrorResponse("Error in put orchestrator resource", result.HttpStatusCode);
            }

            resourceEntity = result.Result as OrchestratorResourceEntity;

            // insert entity into controller config table
            ControllerProperties ctrlProps = (ControllerProperties)controller.Properties;
            int httpCode = await this.InsertOrReplaceOchestratorConfigAsync(ctrlProps.PropertiesList.DncAppId, resourceGuid, orchestratorType, propJson);
            if (!HttpStatusCodeCheck.IsSuccess(httpCode))
            {
                return Orchestrator.CreateErrorResponse("Error in put orchestrator config", httpCode);
            }

            return resourceEntity.ToOrchestrator();
        }

        /// <summary>
        /// Modify an existing Orchestrator
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="resourceName"></param>
        /// <param name="orchestratorRequest"></param>
        /// <returns></returns>
        public Task<Orchestrator> MergeOrchestratorAsync(
            string subId, string resourceGroup, string resourceName, OrchestratorRequest orchestratorRequest)
        {
            throw new NotSupportedException();
        }

        /// <summary>
        /// Delete Orchestrator
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="resourceName"></param>
        /// <returns></returns>
        public async Task<Orchestrator> DeleteOrchestratorAsync(string subId, string resourceGroup, string resourceName)
        {
            this.logger.LogInformation("Starting DELETE orchestrator resource. Sub:{0}, RG:{1}, ResourceName:{2}",
                subId, resourceGroup, resourceName);

            // Create Retrieve table operation
            string rowKey = $"{resourceGroup}_{resourceName}";
            TableResult result = await this.dataStore.GetEntityAsync<OrchestratorResourceEntity>(resourceTableName, subId, rowKey);
            if (!HttpStatusCodeCheck.IsSuccess(result.HttpStatusCode))
            {
                if (HttpStatusCodeCheck.IsResourceNotFound(result.HttpStatusCode))
                {
                    return new Orchestrator(subId, resourceGroup, resourceName);
                }

                return Orchestrator.CreateErrorResponse("Error in delete orchestrator. GetEntity failed.", result.HttpStatusCode);
            }

            OrchestratorResourceEntity resourceEntity = result.Result as OrchestratorResourceEntity;
            ControllerProperties controllerProps = JsonUtility.Deserialize<ControllerProperties>(resourceEntity.ControllerProperties);

            // First delete entry in config table
            int statusCode = await this.DeleteOchestratorConfigAsync(controllerProps.PropertiesList.DncAppId, resourceEntity.ResouceGuid);
            if (!HttpStatusCodeCheck.IsSuccess(statusCode))
            {
                return Orchestrator.CreateErrorResponse("Error in delete orchestrator config", result.HttpStatusCode);
            }

            // Now, delete entry in resource table
            result = await this.dataStore.DeleteEntityAsync(resourceTableName, resourceEntity);
            if (!HttpStatusCodeCheck.IsSuccess(result.HttpStatusCode))
            {
                return Orchestrator.CreateErrorResponse("Error in delete orchestrator resource", result.HttpStatusCode);
            }

            return resourceEntity.ToOrchestrator();
        }

        /// <summary>
        /// Insert or replace an entity in Controller config table
        /// </summary>
        /// <param name="mdncId"></param>
        /// <param name="orchestratorInstanceId"></param>
        /// <param name="type"></param>
        /// <param name="propertiesJson"></param>
        /// <returns></returns>
        private async Task<int> InsertOrReplaceOchestratorConfigAsync(
            string mdncId, string orchestratorResourceGuid, OrchestratorType type, string propertiesJson)
        {
            OrchestratorConfigEntity entity = new OrchestratorConfigEntity(mdncId, orchestratorResourceGuid);
            entity.OrchestratorType = type.ToString();
            entity.Properties = propertiesJson;

            TableResult result = await this.dataStore.InsertOrReplaceEntityAsync(configTableName, entity);
            return result.HttpStatusCode;
        }

        /// <summary>
        /// Delete orchestrator config table
        /// </summary>
        /// <param name="mdncId"></param>
        /// <param name="controllerId"></param>
        /// <param name="type"></param>
        /// <param name="propertiesJson"></param>
        /// <returns></returns>
        private async Task<int> DeleteOchestratorConfigAsync(
            string mdncId, string orchestratorResourceGuid)
        {
            TableResult result = await this.dataStore.GetEntityAsync<OrchestratorConfigEntity>(
                configTableName, mdncId, orchestratorResourceGuid);
            if (!HttpStatusCodeCheck.IsSuccess(result.HttpStatusCode))
            {
                // If resource not found in config table, that is not treated as error
                if (HttpStatusCodeCheck.IsResourceNotFound(result.HttpStatusCode))
                {
                    return (int)HttpStatusCode.OK;
                }

                return (int)result.HttpStatusCode;
            }

            OrchestratorConfigEntity entity = new OrchestratorConfigEntity(mdncId, orchestratorResourceGuid);
            entity.ETag = "*";

            // Execute Delete table operation
            result = await this.dataStore.DeleteEntityAsync(configTableName, entity);
            return result.HttpStatusCode;
        }
    }
}
