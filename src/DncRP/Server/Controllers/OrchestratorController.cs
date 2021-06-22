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
                });
        }

        /// <summary>
        ///  PUT the Orchestrator resource. It will create Orchestrator resource if it doesn't exists
        /// </summary>
        /// <returns></returns>
        [HttpPut]
        public async Task<ContentResult> PutOrchestratorResource(string subscriptionId, string resourceGroupName, string resourceName)
        {
            if (HttpContext.Request.Body == null)
            {
                throw new EmptyRequestBody("Empty request body");
            }

            OrchestratorRequest request = null;
            using (StreamReader stream = new StreamReader(HttpContext.Request.Body))
            {
                string jsonBody = await stream.ReadToEndAsync();
                this.logger.CLogInformation("PutOrchestratorResource body: {0}", jsonBody);

                request = JsonUtility.Deserialize<OrchestratorRequest>(jsonBody);
            }

            request.Identity = this.GetIdentityFromRequestHeaders(HttpContext.Request, request);
            MSIIdentity orchIdentity = await this.GetOrchestratorIdentity(request.Identity);
            if (orchIdentity == null)
            {
                throw new ErrorResponseMessageException(
                    httpStatus: HttpStatusCode.BadRequest,
                    errorCode: ErrorResponseCode.InvalidOrchestratorRequest,
                    errorMessage: $"Failed to get orchestrator identity");
            }

            ValidateRequestParameters(request);

            OrchestratorProperties orchProps = request.Properties;

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

            // create private endpoint
            string peName = "";
            if (orchProps.PrivateLinkId != null && orchProps.PrivateLinkId != "")
            {
                UInt32 hashCode = (UInt32)ExtensionMethods.GetDeterministicHashCode(orchProps.PrivateLinkId);
                peName = GetPrivateEndpointName(hashCode);

                NetworkModel.PrivateEndpoint privateEndpoint = await azVnetBroker.GetOrCreatePrivateEndpoint(
                    orchProps.PrivateLinkId, peName);

                string privateEndpointIP = await azVnetBroker.GetPrivateEndpointIP(privateEndpoint);
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

            // insert entry in cosmos
            ContentResult result = await this.ExecuteWithRetryAsync(
                subscriptionId,
                () => this.tableOperations.OrchestratorOperations.InsertOrReplaceOrchestratorAsync(subscriptionId, resourceGroupName, resourceName, peName, resourceGuid, orchIdentity, request),
                HttpMethod.Put,
                MetricNames.PutOrchestrator,
                new List<DimensionValue>()
                {
                    new DimensionValue() { Name = DimensionNames.Subscription, Value = subscriptionId },
                    new DimensionValue() { Name = DimensionNames.ResourceGroup, Value = resourceGroupName }
                });

            //call dnc put nodeset
            ControllerProperties controllerProperties = (ControllerProperties)controller.Properties;
            DncClient dncClient = new DncClient(this.dncHttpUtility, this.logger, controllerProperties.PropertiesList.DncEndpoint);
            logger.CLogInformation("Calling dnc putnodeset for orchestrator id: {0}", resourceGuid);
            await dncClient.PutNodesetAsync(resourceGuid);

            return result;
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
            Orchestrator orch = await this.tableOperations.OrchestratorOperations.GetOrchestratorAsync(subscriptionId, resourceGroupName, resourceName);
            if (orch == null)
            {
                this.logger.LogInformation("Orchestrator resource {0} already removed", resourceName);
                return new ContentResult
                {
                    ContentType = "application/json",
                    StatusCode = 200
                };
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
                else
                {
                    response.EnsureSuccessStatusCode();
                }
            }

            // call dnc delete nodeset
            bool.TryParse(HttpContext.Request.Query["forceDelete"], out bool forceDelete);
            ControllerProperties controllerProperties = (ControllerProperties)controller.Properties;
            DncClient dncClient = new DncClient(this.dncHttpUtility, this.logger, controllerProperties.PropertiesList.DncEndpoint);
            logger.CLogInformation("Calling dnc deletenodeset for orchestrator id: {0}, forceDelete:{1}", orch.Properties.ResourceGuid, forceDelete);
            await dncClient.DeleteNodesetAsync(orch.Properties.ResourceGuid, forceDelete);

            // get dnc nodeset should return 404
            DncSettings dncSettings = CurrentContext.DncSettings;
            await AsyncRetry.Retry(() => CheckIfNodesetDeleted(dncClient, orch.Properties.ResourceGuid), dncSettings.RetryCount,
                        TimeSpan.FromSeconds(dncSettings.RetryIntervalInSeconds), TimeSpan.FromSeconds(dncSettings.RetryTimeoutInSeconds));

            // remove entries from db
            return await this.ExecuteWithRetryAsync(
                subscriptionId,
                () => this.tableOperations.OrchestratorOperations.DeleteOrchestratorAsync(subscriptionId, resourceGroupName, resourceName),
                HttpMethod.Delete,
                MetricNames.DeleteOrchestrator,
                new List<DimensionValue>()
                {
                    new DimensionValue() { Name = DimensionNames.Subscription, Value = subscriptionId },
                    new DimensionValue() { Name = DimensionNames.ResourceGroup, Value = resourceGroupName }
                });
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
                throw new ErrorResponseMessageException(HttpStatusCode.BadRequest, ErrorResponseCode.None,
                    string.Format(errorMessage, "kind"));
            }
            if (request.Properties == null)
            {
                throw new ErrorResponseMessageException(HttpStatusCode.BadRequest, ErrorResponseCode.None,
                    string.Format(errorMessage, "properties"));
            }
            if (string.IsNullOrEmpty(request.Properties.AppId))
            {
                throw new ErrorResponseMessageException(HttpStatusCode.BadRequest, ErrorResponseCode.None,
                    string.Format(errorMessage, "properties.orchestratorAppId"));
            }
            if (string.IsNullOrEmpty(request.Properties.ClusterRootCA))
            {
                throw new ErrorResponseMessageException(HttpStatusCode.BadRequest, ErrorResponseCode.None,
                    string.Format(errorMessage, "properties.clusterRootCA"));
            }
            if (string.IsNullOrEmpty(request.Properties.TenantId))
            {
                throw new ErrorResponseMessageException(HttpStatusCode.BadRequest, ErrorResponseCode.None,
                    string.Format(errorMessage, "properties.orchestratorTenantId"));
            }
            if (request.Properties.ControllerDetails == null)
            {
                throw new ErrorResponseMessageException(HttpStatusCode.BadRequest, ErrorResponseCode.None,
                    string.Format(errorMessage, "properties.controllerDetails"));
            }
            if (string.IsNullOrEmpty(request.Properties.ControllerDetails.Id))
            {
                throw new ErrorResponseMessageException(HttpStatusCode.BadRequest, ErrorResponseCode.None,
                    string.Format(errorMessage, "properties.controllerDetails.Id"));
            }
        }
    }
}
