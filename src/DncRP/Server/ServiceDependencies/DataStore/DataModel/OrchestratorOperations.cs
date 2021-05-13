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
    using System.Net.Http;
    using System.Net.Http.Headers;
    using System.Threading;
    using System.Threading.Tasks;
    using global::Azure.Security.KeyVault.Secrets;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Azure.Cosmos.Table;
    using Microsoft.Azure.KeyVault;
    using Microsoft.Extensions.Logging;

    public class OrchestratorOperations : IOrchestratorOperations
    {
        private ITableDataStore dataStore;
        private const string resourceTableName = "OrchestratorInstanceResource";
        private const string configTableName = "OrchestratorInstancesConfiguration";
        private ILogger logger;
        private IControllerConfiguration controllerOperations;
        private IHttpUtility httpUtility;
        private IKeyVaultUtility keyVaultUtility;
        private bool deleteNotSupported = false;

        /// <summary>
        /// Create an instance of OrchestratorOperations
        /// </summary>
        /// <param name="tableDataStore"></param>
        /// <param name="logger"></param>
        public OrchestratorOperations(
            ITableDataStore tableDataStore,
            ILogger logger,
            IControllerConfiguration controllerConfiguration,
            IHttpUtility httpUtility,
            IKeyVaultUtility keyVaultUtility)
        {
            this.dataStore = tableDataStore;
            this.logger = logger;
            this.controllerOperations = controllerConfiguration;
            this.httpUtility = httpUtility;
            this.keyVaultUtility = keyVaultUtility;

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
            this.logger.CLogInformation("Starting PUT orchestrator resource. Sub:{0}, RG:{1}, ResourceName:{2}",
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

            OrchestratorProperties orchProps = orchestratorRequest.Properties.Parameters.Properties;
            string propJson = JsonUtility.Serialize(orchProps);
            OrchestratorKind orchestratorType = orchestratorRequest.Properties.Parameters.Kind;
            ControllerProperties controllerProps = (ControllerProperties)controller.Properties;
            string identityMetadataJson = JsonUtility.Serialize<MSIIdentityMetadata>(orchestratorRequest.Identity);

            OrchestratorResourceEntity resourceEntity = new OrchestratorResourceEntity(subId, rowKey);
            resourceEntity.ResouceGuid = resourceGuid;
            resourceEntity.OrchestratorType = orchestratorType.ToString();
            resourceEntity.Properties = propJson;
            resourceEntity.State = ProvisioningState.Succeeded.ToString();
            resourceEntity.CreatedDateUtc = createdDate;
            resourceEntity.ControllerId = controllerArmResouceId;
            resourceEntity.Location = orchestratorRequest.Properties.Parameters.Location;
            resourceEntity.ControllerProperties = JsonUtility.Serialize<ControllerProperties>(controllerProps);
            resourceEntity.IdentityMetadata = identityMetadataJson;

            // Execute the operation.
            result = await this.dataStore.InsertOrReplaceEntityAsync(resourceTableName, resourceEntity);
            if (!HttpStatusCodeCheck.IsSuccess(result.HttpStatusCode))
            {
                return Orchestrator.CreateErrorResponse("Error in put orchestrator resource", result.HttpStatusCode);
            }

            resourceEntity = result.Result as OrchestratorResourceEntity;

            // insert entity into controller config table
            int httpCode = await this.InsertOrReplaceOchestratorConfigAsync(
                controller.Properties.ResourceGuid, resourceGuid, orchestratorType, orchProps, orchestratorRequest.Identity);
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
            this.logger.CLogInformation("Starting DELETE orchestrator resource. Sub:{0}, RG:{1}, ResourceName:{2}",
                subId, resourceGroup, resourceName);

            if (this.deleteNotSupported)
            {
                throw new NotSupportedException("Delete Orchestrator API is not supported yet");
            }

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
            int statusCode = await this.DeleteOchestratorConfigAsync(controllerProps.ResourceGuid, resourceEntity.ResouceGuid);
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
        /// <param name="orchPropertiesJson"></param>
        /// <param name="newOrchResouce">
        /// </param>
        /// <returns></returns>
        private async Task<int> InsertOrReplaceOchestratorConfigAsync(
            string mdncId,
            string orchestratorResourceGuid,
            OrchestratorKind type,
            OrchestratorProperties orchRequestProperties,
            MSIIdentityMetadata metadata)
        {
            string clientsecret = null, identityProps;
            OrchestratorPropertiesInternal orchProperties = orchRequestProperties.ToPropertiesInternal();

            TableResult result = await this.dataStore.GetEntityAsync<OrchestratorConfigEntity>(
                configTableName, mdncId, orchestratorResourceGuid);

            // client ID and client secret are RP generated/retrieved data that is stored
            // in orchestrator properties. It can't be changed by user on subsequent PUT request
            if (HttpStatusCodeCheck.IsResourceNotFound(result.HttpStatusCode))
            {
                MSIIdentity orchIdentity = await this.GetOrchestratorIdentity(metadata);

                orchProperties.ClientId = orchIdentity.ClientId;
                orchProperties.ClientSecretName = orchestratorResourceGuid;

                clientsecret = orchIdentity.ClientSecret;
                orchIdentity.ClientSecret = null; // We don't want to store client secret in Cosmos. It will be stored in KV
                identityProps = JsonUtility.Serialize<MSIIdentity>(orchIdentity);
            }
            else if (!HttpStatusCodeCheck.IsSuccess(result.HttpStatusCode))
            {
                this.logger.CLogError("Error in GET Orchestrator config. StatusCode:{0}", result.HttpStatusCode);
                return result.HttpStatusCode;
            }
            else
            {
                OrchestratorConfigEntity configEntity = result.Result as OrchestratorConfigEntity;
                OrchestratorPropertiesInternal props = JsonUtility.Deserialize<OrchestratorPropertiesInternal>(configEntity.Properties);
                orchProperties.ClientId = props.ClientId;
                orchProperties.ClientSecretName = props.ClientSecretName;
                identityProps = configEntity.IdentityProperties;
            }

            OrchestratorConfigEntity entity = new OrchestratorConfigEntity(mdncId, orchestratorResourceGuid);
            entity.OrchestratorType = type.ToString();
            entity.Properties = JsonUtility.Serialize<OrchestratorPropertiesInternal>(orchProperties);
            entity.IdentityProperties = identityProps;

            result = await this.dataStore.InsertOrReplaceEntityAsync(configTableName, entity);
            if (!HttpStatusCodeCheck.IsSuccess(result.HttpStatusCode))
            {
                this.logger.CLogError("InsertOrReplaceOchestratorConfigAsync: Error in PUT Orchestrator config. StatusCode:{0}", result.HttpStatusCode);
                return result.HttpStatusCode;
            }

            // First store the secret in key vault and then update table. We want to make sure
            // secret is persisted before we mark Ochestrator PUT as success
            if (!string.IsNullOrWhiteSpace(clientsecret))
            {
                this.keyVaultUtility.SetSecret(
                    CurrentContext.KeyVaultUrl, CurrentContext.ManagedIdentity, orchProperties.ClientSecretName, clientsecret);
            }

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

            OrchestratorConfigEntity entity = result.Result as OrchestratorConfigEntity;
            OrchestratorPropertiesInternal props = JsonUtility.Deserialize<OrchestratorPropertiesInternal>(entity.Properties);

            // Delete secret from key vault
            DeleteSecretOperation secret = this.keyVaultUtility.DeleteSecret(
                CurrentContext.KeyVaultUrl, CurrentContext.ManagedIdentity, props.ClientSecretName);

            entity.ETag = "*";

            // Execute Delete table operation
            result = await this.dataStore.DeleteEntityAsync(configTableName, entity);
            return result.HttpStatusCode;
        }

        private async Task<MSIIdentity> GetOrchestratorIdentity(MSIIdentityMetadata identity)
        {
            HttpRequestMessage requestMessage = new HttpRequestMessage
            {
                Method = HttpMethod.Get,
                RequestUri = new Uri(identity.Url),
            };

            HttpResponseMessage response = await this.httpUtility.SendRequestAndGetHttpResponseAsync(
                requestMessage, HttpCompletionOption.ResponseHeadersRead, CancellationToken.None);

            if (response.StatusCode == HttpStatusCode.Unauthorized)
            {
                var authenticateHeaders = response.Headers.WwwAuthenticate;

                foreach (var authenticateHeader in authenticateHeaders)
                {
                    // Sample authentication header - Bearer authorization = "https://login.windows-ppe.net/5D929AE3-B37C-46AA-A3C8-C1558902F101"
                    // This check basically checks that header starts with keyword Bearer
                    if (HttpBearerChallenge.IsBearerChallenge(authenticateHeader.ToString()))
                    {
                        var challenge = new HttpBearerChallenge(requestMessage.RequestUri, authenticateHeader.ToString());
                        this.logger.CLogInformation("AuthorizationServer: {0}", challenge.AuthorizationServer);
                        this.logger.CLogInformation("Resource: {0}", challenge.Resource);
                        this.logger.CLogInformation("Scope: {0}", challenge.Scope);

                        string resource = $"{challenge.Resource}/.default";
                        IAzureAD azureAD = new AzureAD();
                        string token = await azureAD.GetAccessToken(
                            CurrentContext.ManagedIdentity,
                            CurrentContext.ArmAccessIdentity,
                            resource,
                            challenge.AuthorizationServer,
                            CurrentContext.AzureEnvironment).ConfigureAwait(false);

                        requestMessage = new HttpRequestMessage
                        {
                            Method = HttpMethod.Get,
                            RequestUri = new Uri($"{identity.Url}?{ResourceConstants.IdentityUrlApiVersion}"),
                        };
                        requestMessage.Headers.Authorization = new AuthenticationHeaderValue("Bearer", token);

                        return await this.httpUtility.SendRequestAndGetJsonAsync<MSIIdentity>(
                            requestMessage, HttpCompletionOption.ResponseHeadersRead, CancellationToken.None);
                    }
                }
            }

            string msg = string.Format("GetOrchestratorIdentity: Unexpected response code: {0}. {1}",
                response.StatusCode,
                response.Content != null ? response.Content.ToString() : string.Empty);
            this.logger.CLogError(msg);

            throw new Exception(msg);
        }
    }
}
