// -----------------------------------------------------------------------------
// <copyright file="ControllerConfiguration.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Collections.Generic;
    using System.IO;
    using System.Net;
    using System.Threading.Tasks;
    using System.Transactions;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Azure.Cosmos.Table;
    using Microsoft.Extensions.Logging;

    public class ControllerConfiguration : IControllerConfiguration
    {
        private ITableDataStore dataStore;
        private const string resourceTableName = "ControllerResource";
        private const string DelegatedSubnetResourceTableName = "DelegatedSubnetResource";
        private const string OrchestratorResourceTableName = "OrchestratorInstanceResource";
        private const string mdncGuid = "5c7e26be-7ff9-4aaf-9e16-0bc759605aea";
        private const string mdncFirstPartyId = "a91b1853-4403-4f54-b5cb-d1ea19d90c37";
        private const string dncUrlSuffix = "dnc.azure.com";
        private const string mdncUri = "https://openai-{0}.{1}";
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

            ControllerPropertiesDetails prop = JsonUtility.Deserialize<ControllerPropertiesDetails>(resourceEntity.Properties);

            return new Controller(
                subId, resourceGroup, controllerName, resourceEntity.Location,
                new ControllerProperties
                {
                    State = TopLevelResourceProperties.ParseProvisioningState(resourceEntity.State),
                    ResourceGuid = resourceEntity.ResouceGuid,
                    PropertiesList = new ControllerPropertiesDetails
                    {
                        DncAppId = prop.DncAppId,
                        DncEndpoint = prop.DncEndpoint
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
            this.logger.CLogInformation("Starting PUT controller resource. Sub:{0}, RG:{1}, ResourceName:{2}",
                subId, resourceGroup, controllerName);

            // Create the InsertOrReplace table operation
            string rowKey = $"{resourceGroup}_{controllerName}";

            // Check if row already exists and if it does, reuse ControllerId. Once created,
            // ControllerId can't be changed.
            TableResult result = await this.dataStore.GetEntityAsync<ControllerResourceEntity>(resourceTableName, subId, rowKey);

            string resourceGuid, createdDate;
            if (HttpStatusCodeCheck.IsResourceNotFound(result.HttpStatusCode))
            {
                resourceGuid = this.GetControllerResourceGuid(controllerName);
                createdDate = DateTime.UtcNow.ToString();
            }
            else
            {
                ControllerResourceEntity entity = result.Result as ControllerResourceEntity;
                resourceGuid = entity.ResouceGuid;
                createdDate = entity.CreatedDateUtc;
            }

            string dncEndpoint = this.ConfigureManagedDNCEndpoint(
                controllerName,
                resourceGuid,
                CurrentContext.AzureRegion
            );

            ControllerPropertiesDetails list = new ControllerPropertiesDetails
            {
                DncAppId = mdncFirstPartyId,
                DncEndpoint = dncEndpoint,
            };

            string propJson = JsonUtility.Serialize(list);

            ControllerResourceEntity resourceEntity = new ControllerResourceEntity(subId, rowKey);
            resourceEntity.ResouceGuid = resourceGuid;
            // Succeeded for now. In future, it will be Provisioning in the begining and will
            // become succeeded or failed depending on if dnc deployment succeeds.
            resourceEntity.State = ProvisioningState.Succeeded.ToString();
            resourceEntity.CreatedDateUtc = createdDate;
            resourceEntity.Location = controllerRequest.Location;
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
        public async Task<Controller> MergeDncControllerAsync(
           string subId, string resourceGroup, string controllerName, ControllerRequest controllerRequest)
        {
            return new Controller(
                subId, resourceGroup, controllerName, "",
                new ControllerProperties
                {
                    State = ProvisioningState.Succeeded,
                });
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
            this.logger.CLogInformation("Starting DELETE controller resource. Sub:{0}, RG:{1}, ResourceName:{2}",
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

        private string GetControllerResourceGuid(string controllerName)
        {
            if (controllerName.StartsWith("AzureContainerNetworking_", StringComparison.OrdinalIgnoreCase))
            {
                string[] parts = controllerName.Split(new string[] { "_" }, StringSplitOptions.RemoveEmptyEntries);
                if (parts.Length == 2)
                {
                    Guid ctrlGuid;
                    if (Guid.TryParse(parts[1], out ctrlGuid))
                    {
                        // Parsing is done to make sure Guid format is correct
                        return ctrlGuid.ToString();
                    }
                }

                throw new InvalidDataException($"Invalid test controller name {controllerName}");
            }

            return mdncGuid;
        }

        private string ConfigureManagedDNCEndpoint(string controllerName, string controllerId, string location)
        {
            string region = location.Replace(" ", "");
            if (controllerName.StartsWith("AzureContainerNetworking_", StringComparison.OrdinalIgnoreCase))
            {
                return "https://" + controllerId + "-" + region + "." + dncUrlSuffix;
            }
            else
            {
                return string.Format(mdncUri, region, dncUrlSuffix);
            }
        }

        public async Task<List<DelegateSubnet>> GetDelegatedSubnetResourcesByControllerId(string armControllerId)
        {
            TableQuery<DelegatedSubnetResourceEntity> tableQuery = new TableQuery<DelegatedSubnetResourceEntity>().Where(
                    TableQuery.GenerateFilterCondition("ControllerId", QueryComparisons.Equal, armControllerId));

            ICollection<DelegatedSubnetResourceEntity> resourceEntities = await dataStore.GetEntitiesAsync(DelegatedSubnetResourceTableName, tableQuery);

            List<DelegateSubnet> delegateSubnetsList = new List<DelegateSubnet>();
            foreach(DelegatedSubnetResourceEntity dsEntity in resourceEntities)
            {
                DelegateSubnet ds = dsEntity.ConvertToDelegateSubnet();
                delegateSubnetsList.Add(ds);
            }

            return delegateSubnetsList;
        }

        public async Task<List<Orchestrator>> GetOrchestratorResourcesByControllerId(string armControllerId)
        {
            TableQuery<OrchestratorResourceEntity> tableQuery = new TableQuery<OrchestratorResourceEntity>().Where(
                    TableQuery.GenerateFilterCondition("ControllerId", QueryComparisons.Equal, armControllerId));

            ICollection<OrchestratorResourceEntity> resourceEntities = await dataStore.GetEntitiesAsync(OrchestratorResourceTableName, tableQuery);

            List<Orchestrator> orchestratorsList = new List<Orchestrator>();
            foreach (OrchestratorResourceEntity orchEntity in resourceEntities)
            {
                Orchestrator orch = orchEntity.ToOrchestrator();
                orchestratorsList.Add(orch);
            }

            return orchestratorsList;
        }
    }
}
