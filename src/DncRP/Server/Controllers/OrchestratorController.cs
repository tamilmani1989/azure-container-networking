// --------------------------------------------------------------------------------------------------------------------
// <copyright file="OrchestratorController.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Collections.Generic;
    using System.IO;
    using System.Net;
    using System.Net.Http;
    using System.Net.Http.Headers;
    using System.Threading;
    using System.Threading.Tasks;
    using Microsoft.AspNetCore.Mvc;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Azure.KeyVault;
    using Microsoft.Extensions.Logging;
    using NetworkModel = Microsoft.Azure.Management.Network.Models;

    /// <summary>
    /// Class to handle CRUD operations for Orchestrator
    /// </summary>
    [Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DelegatedNetwork/orchestrators/{resourceName}")]
    public class OrchestratorController : DncRpControllerBase<OrchestratorController>
    {
        private AzVNetBroker azVnetBroker;
        private IHttpUtility httpUtility;
        private IHttpUtility dncHttpUtility;

        private const string privateEndpointName = "privateendpoint-{0}";

        public OrchestratorController(
            ITableOperations tableOperations,
            IHttpUtilityFactory httpUtilityFactory,
            ILogger<OrchestratorController> logger,
            IMetricsProvider metricsProvider) : base(logger, tableOperations, metricsProvider)
        {
            IAzureAD azureAD = new AzureAD();
            this.httpUtility = httpUtilityFactory.Resolve(Constants.HttpUtility);
            this.dncHttpUtility = httpUtilityFactory.Resolve(Constants.DncHttpUtility);
            this.azVnetBroker = new AzVNetBroker(httpUtility, logger, azureAD);
            logger.CLogInformation("OrchestratorController initialized");
        }

        /// <summary>
        /// Get the Orchestrator resource
        /// </summary>
        /// <returns></returns>
        [HttpGet]
        public async Task<ContentResult> GetOrchestratorResource(string subscriptionId, string resourceGroupName, string resourceName)
        {
            return await this.ExecuteWithRetryAsync(
                subscriptionId,
                () => this.tableOperations.OrchestratorOperations.GetOrchestratorAsync(subscriptionId, resourceGroupName, resourceName),
                HttpMethod.Get,
                MetricNames.GetOrchestrator,
                new List<DimensionValue>()
                {
                    new DimensionValue() { Name = DimensionNames.Subscription, Value = subscriptionId },
                    new DimensionValue() { Name = DimensionNames.ResourceGroup, Value = resourceGroupName }
                },
                1);
        }

        /// <summary>
        ///  PUT the Orchestrator resource. It will create Orchestrator resource if it doesn't exists
        /// </summary>
        /// <returns></returns>
        [HttpPut]
        public async Task<ContentResult> PutOrchestratorResource(string subscriptionId, string resourceGroupName, string resourceName)
        {
            return await this.ExecuteWithRetryAsync(subscriptionId, () =>
                 ExecutePutOrchestratorAsync(
                   subscriptionId,
                   resourceGroupName,
                   resourceName),
                 HttpMethod.Put,
                 MetricNames.PutOrchestrator,
                 new List<DimensionValue>()
                 {
                    new DimensionValue() { Name = DimensionNames.Subscription, Value = subscriptionId },
                    new DimensionValue() { Name = DimensionNames.ResourceGroup, Value = resourceGroupName }
                 },
                1);
        }

        private async Task<Orchestrator> ExecutePutOrchestratorAsync(
            string subscriptionId,
            string resourceGroupName,
            string resourceName)
        {
            if (HttpContext.Request.Body == null)
            {
                throw new ErrorResponseMessageException(
                    httpStatus: HttpStatusCode.BadRequest,
                    errorCode: ErrorResponseCode.InvalidRequestBody,
                    errorMessage: "Empty request body");
            }

            // deserialize request body to orchestrator request
            OrchestratorRequest request = null;
            using (StreamReader stream = new StreamReader(HttpContext.Request.Body))
            {
                string jsonBody = await stream.ReadToEndAsync();
                this.logger.CLogInformation("PutOrchestratorResource body: {0}", jsonBody);
                request = JsonUtility.Deserialize<OrchestratorRequest>(jsonBody);
            }

            // validate orchestrator request parameters
            ValidateRequestParameters(request);

            // Get controller
            OrchestratorProperties orchProps = request.Properties;
            Controller controller = await tableOperations.ControllerConfiguration.GetDncControllerAsync(
                orchProps.ControllerDetails.Id);
            if (controller.Error != null)
            {
                throw new ErrorResponseMessageException(
                       httpStatus: HttpStatusCode.BadRequest,
                       errorCode: ErrorResponseCode.ControllerIdNotFound,
                       errorMessage: controller.Error.Error);
            }

            // Get MSI
            MSIIdentity orchIdentity = null;
            try
            {
                request.Identity = this.GetIdentityFromRequestHeaders(HttpContext.Request, request);
                orchIdentity = await ExtensionMethods.GetMSIIdentity(httpUtility, request.Identity);
            }
            catch(Exception  e)
            {
                throw new ErrorResponseMessageException(
                    httpStatus: HttpStatusCode.BadRequest,
                    errorCode: ErrorResponseCode.MSIRetrievalFailed,
                    errorMessage: $"Failed to get orchestrator identity: {e.ToString()}");
            }


            // create private endpoint if privatelinkId specified
            string peName = "";
            if (orchProps.PrivateLinkId != null && orchProps.PrivateLinkId != "")
            {
                UInt32 hashCode = (UInt32)ExtensionMethods.GetDeterministicHashCode(orchProps.PrivateLinkId);
                peName = GetPrivateEndpointName(hashCode);
                string privateEndpointIP = null;
                try
                {
                    NetworkModel.PrivateEndpoint privateEndpoint = await azVnetBroker.GetOrCreatePrivateEndpoint(
                        orchProps.PrivateLinkId, peName);

                    privateEndpointIP = await azVnetBroker.GetPrivateEndpointIP(privateEndpoint);
                }
                catch (Exception e)
                {
                    throw new ErrorResponseMessageException(
                        httpStatus: HttpStatusCode.InternalServerError,
                        errorCode: ErrorResponseCode.PrivateEndpointCreateFailed,
                        errorMessage: $"Private endpoint creation failed: {e.ToString()}");
                }

                this.logger.CLogInformation("Private Endpoint IP: {0}", privateEndpointIP);
                orchProps.ApiServerEndpoint = "https://" + privateEndpointIP;
            }
            else if (orchProps.ApiServerEndpoint == null || orchProps.ApiServerEndpoint.Length == 0)
            {
                throw new ErrorResponseMessageException(
                        httpStatus: HttpStatusCode.BadRequest,
                        errorCode: ErrorResponseCode.InvalidOrchestratorRequest,
                        errorMessage: $"Neither privateLinkResourceId nor apiserverEndpoint specified");
            }

            // Get orchestrator
            string resourceGuid = null;
            Orchestrator orch = await this.tableOperations.OrchestratorOperations.GetOrchestratorAsync(
                            subscriptionId, resourceGroupName, resourceName);
            if (orch.Error == null)
            {
                resourceGuid = orch.Properties.ResourceGuid;
                logger.CLogInformation("Orchestrator Exists. {0}", resourceGuid);
            }
            else
            {
                resourceGuid = Guid.NewGuid().ToString();
            }

            // set state to provisioning
            request.Properties.State = ProvisioningState.Provisioning;
            await this.tableOperations.OrchestratorOperations.InsertOrReplaceOrchestratorAsync(subscriptionId, resourceGroupName, resourceName, peName, resourceGuid, orchIdentity, request);
            
            //call dnc put nodeset
            ControllerProperties controllerProperties = (ControllerProperties)controller.Properties;
            DncClient dncClient = new DncClient(this.dncHttpUtility, this.logger, controllerProperties.PropertiesList.DncEndpoint);
            logger.CLogInformation("Calling dnc putnodeset for orchestrator id: {0}", resourceGuid);
            DncSettings dncSettings = CurrentContext.DncSettings;
            await AsyncRetry.Retry(() => dncClient.PutNodesetAsync(resourceGuid), dncSettings.RetryCount,
                TimeSpan.FromSeconds(dncSettings.RetryIntervalInSeconds), TimeSpan.FromSeconds(dncSettings.RetryTimeoutInSeconds));

            // set state to succeeded
            request.Properties.State = ProvisioningState.Succeeded;
            return await this.tableOperations.OrchestratorOperations.InsertOrReplaceOrchestratorAsync(subscriptionId, resourceGroupName, resourceName, peName, resourceGuid, orchIdentity, request);
        }

        /// <summary>
        ///  Patch the Orchestrator resource
        /// </summary>
        /// <returns></returns>
        [HttpPatch]
#pragma warning disable CS1998 // Async method lacks 'await' operators and will run synchronously. NotSupported yet
        public async Task<ContentResult> PatchOrchestratorResource(string subscriptionId, string resourceGroupName, string resourceName)
#pragma warning restore CS1998 // Async method lacks 'await' operators and will run synchronously. NotSupported yet
        {
            throw new NotSupportedException();
        }

        /// <summary>
        ///  Delete the Orchestrator resource
        /// </summary>
        /// <returns></returns>
        [HttpDelete]
        public async Task<ContentResult> DeleteOrchestratorResource(string subscriptionId, string resourceGroupName, string resourceName)
        {
            // Get orchestrator
            return await this.ExecuteWithRetryAsync(
                subscriptionId,
                () => ExecuteDeleteOrchestratorAsync(subscriptionId, resourceGroupName, resourceName),
                HttpMethod.Delete,
                MetricNames.DeleteOrchestrator,
                new List<DimensionValue>()
                {
                    new DimensionValue() { Name = DimensionNames.Subscription, Value = subscriptionId },
                    new DimensionValue() { Name = DimensionNames.ResourceGroup, Value = resourceGroupName }
                },
                1);
        }

        private async Task<Orchestrator> ExecuteDeleteOrchestratorAsync(string subscriptionId, string resourceGroupName, string resourceName)
        {
            Orchestrator orch = await this.tableOperations.OrchestratorOperations.GetOrchestratorAsync(subscriptionId, resourceGroupName, resourceName);
            if (orch.Error != null)
            {
                this.logger.LogInformation("Orchestrator resource {0} already removed", resourceName);
                return new Orchestrator();
            }

            OrchestratorProperties orchProps = (OrchestratorProperties)orch.Properties;

            // Get controller
            Controller controller = await tableOperations.ControllerConfiguration.GetDncControllerAsync(
                                                    orchProps.ControllerDetails.Id);
            if (controller.Error != null)
            {
                throw new ErrorResponseMessageException(
                       httpStatus: HttpStatusCode.BadRequest,
                       errorCode: ErrorResponseCode.ControllerIdNotFound,
                       errorMessage: controller.Error.Error);
            }

            // Delete private endpoint
            if (orchProps.PrivateLinkId != null && orchProps.PrivateLinkId != "")
            {
                UInt32 hashCode = (UInt32)ExtensionMethods.GetDeterministicHashCode(orchProps.PrivateLinkId);
                string privateEndpointName = GetPrivateEndpointName(hashCode);
                this.logger.LogInformation("Deleting private endpoint {0} for orchestrator {1}", privateEndpointName, resourceName);
                HttpResponseMessage response = await azVnetBroker.DeletePrivateEndpoint(privateEndpointName);
                if (response.StatusCode == HttpStatusCode.NotFound)
                {
                    this.logger.LogWarning("PrivateEndpoint resource {0} already removed", privateEndpointName);
                }
                else if (!response.IsSuccessStatusCode)
                {
                    throw new ErrorResponseMessageException(
                        httpStatus: HttpStatusCode.InternalServerError,
                        errorCode: ErrorResponseCode.PrivateEndpointDeleteFailed,
                        errorMessage: $"private endpoint delete failed with code {response.StatusCode} content: {response.Content.ToString()}");
                }
            }

            // call dnc delete nodeset
            bool.TryParse(HttpContext.Request.Query["forceDelete"], out bool forceDelete);
            ControllerProperties controllerProperties = (ControllerProperties)controller.Properties;
            DncClient dncClient = new DncClient(this.dncHttpUtility, this.logger, controllerProperties.PropertiesList.DncEndpoint);
            logger.CLogInformation("Calling dnc deletenodeset for orchestrator id: {0}, forceDelete:{1}", orch.Properties.ResourceGuid, forceDelete);
            
            DncSettings dncSettings = CurrentContext.DncSettings;
            await AsyncRetry.Retry(() => dncClient.DeleteNodesetAsync(orch.Properties.ResourceGuid, forceDelete), dncSettings.RetryCount,
                        TimeSpan.FromSeconds(dncSettings.RetryIntervalInSeconds), TimeSpan.FromSeconds(dncSettings.RetryTimeoutInSeconds));

            // get dnc nodeset should return 404
            await AsyncRetry.Retry(() => CheckIfNodesetDeleted(dncClient, orch.Properties.ResourceGuid), dncSettings.RetryCount,
                        TimeSpan.FromSeconds(dncSettings.RetryIntervalInSeconds), TimeSpan.FromSeconds(dncSettings.RetryTimeoutInSeconds));

            // remove entries from db
            return await this.tableOperations.OrchestratorOperations.DeleteOrchestratorAsync(subscriptionId, resourceGroupName, resourceName);
        }

        private async Task CheckIfNodesetDeleted(
            IDncClient dncClient,
            string orchestratorId)
        {
            HttpResponseMessage response = await dncClient.GetNodesetAsync(orchestratorId);
            if (response.StatusCode != HttpStatusCode.NotFound)
            {
                logger.CLogError("DNC Nodeset Pending delete. status code:{0}", response.StatusCode);
                throw new ErrorResponseMessageException(
                    httpStatus: HttpStatusCode.InternalServerError,
                    errorCode: ErrorResponseCode.NodesetDeleteFailed,
                    $"DNC Nodeset Pending delete. status code:{response.StatusCode}");
            }
        }

        private string GetPrivateEndpointName(UInt32 hashCode)
        {
            return string.Format(privateEndpointName, hashCode);
        }

        public void ValidateRequestParameters(OrchestratorRequest request)
        {
            string errorMessage = "{0} parameter not present";
            if (string.IsNullOrEmpty(request.Location))
            {
                throw new ErrorResponseMessageException(HttpStatusCode.BadRequest, ErrorResponseCode.None,
                    string.Format(errorMessage, "location"));
            }
            if (string.IsNullOrEmpty(request.Kind.ToString()) || request.Kind == OrchestratorKind.Unknown)
            {
                throw new ErrorResponseMessageException(HttpStatusCode.BadRequest, ErrorResponseCode.InvalidRequestBody,
                    string.Format(errorMessage, "kind"));
            }
            if (request.Properties == null)
            {
                throw new ErrorResponseMessageException(HttpStatusCode.BadRequest, ErrorResponseCode.InvalidRequestBody,
                    string.Format(errorMessage, "properties"));
            }
            if (string.IsNullOrEmpty(request.Properties.AppId))
            {
                throw new ErrorResponseMessageException(HttpStatusCode.BadRequest, ErrorResponseCode.InvalidRequestBody,
                    string.Format(errorMessage, "properties.orchestratorAppId"));
            }
            if (string.IsNullOrEmpty(request.Properties.ClusterRootCA))
            {
                throw new ErrorResponseMessageException(HttpStatusCode.BadRequest, ErrorResponseCode.InvalidRequestBody,
                    string.Format(errorMessage, "properties.clusterRootCA"));
            }
            if (string.IsNullOrEmpty(request.Properties.TenantId))
            {
                throw new ErrorResponseMessageException(HttpStatusCode.BadRequest, ErrorResponseCode.InvalidRequestBody,
                    string.Format(errorMessage, "properties.orchestratorTenantId"));
            }
            if (request.Properties.ControllerDetails == null)
            {
                throw new ErrorResponseMessageException(HttpStatusCode.BadRequest, ErrorResponseCode.InvalidRequestBody,
                    string.Format(errorMessage, "properties.controllerDetails"));
            }
            if (string.IsNullOrEmpty(request.Properties.ControllerDetails.Id))
            {
                throw new ErrorResponseMessageException(HttpStatusCode.BadRequest, ErrorResponseCode.InvalidRequestBody,
                    string.Format(errorMessage, "properties.controllerDetails.Id"));
            }
        }
    }
}
