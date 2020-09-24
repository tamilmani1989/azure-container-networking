// -----------------------------------------------------------------------------
// <copyright file="OrchestratorTests.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP.ServiceTests
{
    using System;
    using System.IO;
    using System.Net;
    using System.Net.Http;
    using System.Text;
    using System.Text.RegularExpressions;
    using Microsoft.AspNetCore.Http;
    using Microsoft.AspNetCore.Mvc;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Configuration;
    using Microsoft.Extensions.Logging;
    using Moq;
    using Newtonsoft.Json.Linq;
    using Xunit;
    using Match = System.Text.RegularExpressions.Match;
    using Contracts = Microsoft.Azure.ContainerNetworking.DncRP;

    public class OrchestratorTests
    {
        private IHttpUtility httpUtility;
        private TestConfig testConfig;
        private IConfiguration configuration;
        private ClientIdentity clientIdentity;
        private DatastoreSettings datastoreSettings;
        private OrchestratorController orchController;
        private DncController dncController;
        private const string settingsFileName = "servicetests.settings.json";
        private const string connectStrRegex = "RunAs=App;AppId=(.*);TenantId=(.*);AppKey=(.*)";
        string location = "Test location";

        public OrchestratorTests()
        {
            this.httpUtility = new HttpUtility(new HttpClient());
            var builder = new ConfigurationBuilder()
              .SetBasePath(Directory.GetCurrentDirectory())
              .AddJsonFile(settingsFileName);

            configuration = builder.Build();
            this.testConfig = configuration.Get<TestConfig>();

            Match match = Regex.Match(this.testConfig.KeyVaultConnectionString, connectStrRegex);
            if (!match.Success)
            {
                throw new ArgumentException($"Invalid connection string {this.testConfig.KeyVaultConnectionString}");
            }

            this.clientIdentity = new ClientIdentity
            {
                AppId = match.Groups[1].Value,
                TenantId = match.Groups[2].Value,
                AppKey = match.Groups[3].Value,
                IdentityType = ClientIdentityType.SpClientSecret
            };

            this.datastoreSettings = new DatastoreSettings
            {
                SubscriptionId = this.testConfig.SubscriptionId,
                DBAccountName = this.testConfig.DbAccountName,
                ResourceGroup = this.testConfig.ResourceGroup,
                Region = this.testConfig.AzureRegion
            };

            CurrentContext.SetContext(this.configuration["AzureEnvironment"], this.configuration["AzureRegion"]);
            CurrentContext.SetManagedIdentity(clientIdentity);

            IConnectionStringProvider connectionStringProvider = new ConnectionStringProvider(
                this.httpUtility,
                datastoreSettings);

            Mock<ILogger<CosmosDbDataStore>> logger = new Mock<ILogger<CosmosDbDataStore>>();
            Mock<ILogger<TableOperations>> tableOpLogger = new Mock<ILogger<TableOperations>>();
            Mock<ILogger<OrchestratorController>> orchLogger = new Mock<ILogger<OrchestratorController>>();
            Mock<ILogger<DncController>> dncCtrlLogger = new Mock<ILogger<DncController>>();

            CosmosDbDataStore dataStore = new CosmosDbDataStore(connectionStringProvider, logger.Object);
            dataStore.InitAsync().GetAwaiter().GetResult();
            ITableOperations tableOperations = new TableOperations(dataStore, tableOpLogger.Object);
            this.orchController = new OrchestratorController(tableOperations, orchLogger.Object);
            this.dncController = new DncController(tableOperations, dncCtrlLogger.Object);
        }

        [Fact]
        public async void TestOrchestratorResource_Get_Put_Delete()
        {
            string guid = Guid.NewGuid().ToString();
            string[] strs = guid.Split(new string[] { "-" }, StringSplitOptions.RemoveEmptyEntries);
            string rgName = strs[0];
            string orchName = strs[1];
            string ctrlName = strs[2];
            Orchestrator expectedOrch = new Orchestrator(this.testConfig.SubscriptionId, rgName, orchName);

            // Get controller resource and validate results
            ContentResult result = await dncController.GetDncResource(this.testConfig.SubscriptionId, rgName, ctrlName);
            Assert.True(result.StatusCode == (int)HttpStatusCode.NotFound, "Resource exists but expectation is that it should not");

            ControllerRequest request = new ControllerRequest
            {
                Properties = new ArmProperties
                {
                    Parameters = new ControllerRequestParameters
                    {
                        Location = location
                    }
                }
            };

            // Put controller resource and validate results
            this.SetControllerHttpContext(dncController, this.testConfig.SubscriptionId, rgName, ctrlName, HttpMethod.Put, request);
            result = await dncController.PutDncResource(this.testConfig.SubscriptionId, rgName, ctrlName);
            Assert.True(HttpStatusCodeCheck.IsSuccess((int)result.StatusCode), "PutDncResource failed");
            Assert.True(result.StatusCode == (int)HttpStatusCode.OK, "PutDncResource failed created status check");

            // Get controller resource and validate results
            result = await orchController.GetOrchestratorResource(this.testConfig.SubscriptionId, rgName, orchName);
            Assert.True(HttpStatusCodeCheck.IsResourceNotFound((int)result.StatusCode));

            OrchestratorRequest orchRequest = this.GetOrchestratorObject(this.GetControllerArmId(this.testConfig.SubscriptionId, rgName, ctrlName));

            // Put orchestrator resource and validate results
            this.SetOrchestratorHttpContext(orchController, this.testConfig.SubscriptionId, rgName, orchName, HttpMethod.Put, orchRequest);
            result = await orchController.PutOrchestratorResource(this.testConfig.SubscriptionId, rgName, orchName);
            Assert.True(HttpStatusCodeCheck.IsSuccess((int)result.StatusCode));
            this.ValidateOrchestratorResponse(result.Content, orchRequest, expectedOrch);

            // Get after Put. Get orchestrator resource and validate results
            result = await orchController.GetOrchestratorResource(this.testConfig.SubscriptionId, rgName, orchName);
            Assert.True(HttpStatusCodeCheck.IsSuccess((int)result.StatusCode));
            this.ValidateOrchestratorResponse(result.Content, orchRequest, expectedOrch);

            // Delete orchestrator resouce and validate result
            result = await orchController.DeleteOrchestratorResource(this.testConfig.SubscriptionId, rgName, orchName);
            Assert.True(HttpStatusCodeCheck.IsSuccess((int)result.StatusCode));

            // Delete controller resouce and validate result
            result = await dncController.DeleteDncResource(this.testConfig.SubscriptionId, rgName, ctrlName);
            Assert.True(HttpStatusCodeCheck.IsSuccess((int)result.StatusCode));
        }

        [Fact]
        public async void TestOrchestratorResource_Invalid_ControllerId()
        {
            string guid = Guid.NewGuid().ToString();
            string[] strs = guid.Split(new string[] { "-" }, StringSplitOptions.RemoveEmptyEntries);
            string rgName = strs[0];
            string orchName = strs[1];
            string ctrlName = strs[2];

            // Get controller resource and validate results
            ContentResult result = await orchController.GetOrchestratorResource(this.testConfig.SubscriptionId, rgName, orchName);
            Assert.True(HttpStatusCodeCheck.IsResourceNotFound((int)result.StatusCode));

            OrchestratorRequest orchRequest = this.GetOrchestratorObject(this.GetInvalidControllerArmId());

            // Put orchestrator resource and validate results
            this.SetOrchestratorHttpContext(orchController, this.testConfig.SubscriptionId, rgName, orchName, HttpMethod.Put, orchRequest);
            result = await orchController.PutOrchestratorResource(this.testConfig.SubscriptionId, rgName, orchName);
            Assert.True(HttpStatusCodeCheck.IsBadRequest((int)result.StatusCode));

            int badRequest = (int)HttpStatusCode.BadRequest;
            ErrorDetails errorDetails = JsonUtility.Deserialize<ErrorDetails>(result.Content);
            Assert.True(errorDetails.ErrorResponse.Error.Code == badRequest.ToString());
            Assert.True(errorDetails.ErrorResponse.Error.Message.Contains(
                ErrorResponseCode.ControllerIdInvalidFormat.ToString(), StringComparison.OrdinalIgnoreCase));
        }

        private void SetOrchestratorHttpContext(
            OrchestratorController controller, string subId, string resourceGroup, string resourceName, HttpMethod method,
            OrchestratorRequest request)
        {
            controller.ControllerContext = new ControllerContext();
            controller.ControllerContext.HttpContext = new DefaultHttpContext();
            controller.ControllerContext.HttpContext.Request.Host = new HostString($"localhost:5002");
            controller.ControllerContext.HttpContext.Request.Path = new PathString($"/subscriptions/{subId}/resourceGroups/{resourceGroup}/providers/Microsoft.DelegatedNetwork/orchestratorInstances/{resourceName}");
            controller.ControllerContext.HttpContext.Request.Method = method.ToString();
            controller.ControllerContext.HttpContext.Request.PathBase = "";
            controller.ControllerContext.HttpContext.Request.Scheme = "https";

            string json = JsonUtility.Serialize(request);
            MemoryStream stream = new MemoryStream(Encoding.UTF8.GetBytes(json));
            controller.ControllerContext.HttpContext.Request.Body = stream;
        }

        private void SetControllerHttpContext(
           DncController controller, string subId, string resourceGroup, string controllerName, HttpMethod method,
           ControllerRequest request)
        {
            controller.ControllerContext = new ControllerContext();
            controller.ControllerContext.HttpContext = new DefaultHttpContext();
            controller.ControllerContext.HttpContext.Request.Host = new HostString($"localhost:5002");
            controller.ControllerContext.HttpContext.Request.Path = new PathString($"/subscriptions/{subId}/resourceGroups/{resourceGroup}/providers/Microsoft.DelegatedNetwork/controller/{controllerName}");
            controller.ControllerContext.HttpContext.Request.Method = method.ToString();
            controller.ControllerContext.HttpContext.Request.PathBase = "";
            controller.ControllerContext.HttpContext.Request.Scheme = "https";

            string json = JsonUtility.Serialize(request);
            MemoryStream stream = new MemoryStream(Encoding.UTF8.GetBytes(json));
            controller.ControllerContext.HttpContext.Request.Body = stream;
        }

        private OrchestratorRequest GetOrchestratorObject(string controllerArmId)
        {
            return new OrchestratorRequest
            {
                Properties = new OrchestratorArmProperties
                {
                    Parameters = new OrchestratorRequestParameters
                    {
                        Location = location,
                        Properties = new OrchestratorProperties
                        {
                            orchestratorInstanceDetails = new OrchestratorInstanceDetails
                            {
                                Type = OrchestratorType.Kubernetes,
                                Properties = new OrchestratorInstanceProperties
                                {
                                    AppId = "546192d7-503f-477a-9cfe-4efc3ee2b6e1", // AAD ID for k8s apiserver auth
                                    TenantId = "da6192d7-503f-477a-9cfe-4efc3ee2b6c3", // Tenant id of above AAD ID
                                    ClusterRootCA = "ddsadsad344mfdsfdl", // Root CA of k8s cluster
                                    ApiServerEndpoint = "https://testk8s.cloudapp.net"  //apiserver url that dnc needs to watch on
                                }
                            },
                            ControllerDetails = new ControllerDetails
                            {
                                Id = controllerArmId
                            }
                        }
                    }
                }
            };
        }

        private string GetControllerArmId(string subId, string rgName, string ctrlName)
        {
            Contracts.Controller ctlr = new Contracts.Controller(subId, rgName, ctrlName);
            return ctlr.Id;
        }

        private string GetInvalidControllerArmId()
        {
            return "/subscriptions/resourceGroups/Microsoft.DelegatedNetwork/controller/ctrlName";
        }

        private void ValidateOrchestratorResponse(string jsonResponse, OrchestratorRequest orchRequest, Orchestrator orch)
        {
            JObject jsonDeserialized = JObject.Parse(jsonResponse);
            Assert.True(jsonDeserialized["id"].ToString().Equals(orch.Id));
            Assert.True(jsonDeserialized["name"].ToString().Equals(orch.Name));
            Assert.True(jsonDeserialized["location"].ToString().Equals(location));
            Assert.True(jsonDeserialized["type"].ToString().Equals(orch.Type));
            Assert.True(jsonDeserialized["properties"]["state"].ToString().Equals(ProvisioningState.Succeeded.ToString()));
            Assert.True(!string.IsNullOrWhiteSpace(jsonDeserialized["properties"]["resourceGuid"].ToString()));
            Assert.True(jsonDeserialized["properties"]["orchestratorInstanceDetails"]["orchestratorType"].ToString().Equals(orchRequest.Properties.Parameters.Properties.orchestratorInstanceDetails.Type.ToString()));
            Assert.True(jsonDeserialized["properties"]["orchestratorInstanceDetails"]["properties"]["orchestratorAppID"].ToString().Equals(orchRequest.Properties.Parameters.Properties.orchestratorInstanceDetails.Properties.AppId));
            Assert.True(jsonDeserialized["properties"]["orchestratorInstanceDetails"]["properties"]["orchestratorTenantID"].ToString().Equals(orchRequest.Properties.Parameters.Properties.orchestratorInstanceDetails.Properties.TenantId));
            Assert.True(jsonDeserialized["properties"]["orchestratorInstanceDetails"]["properties"]["clusterRootCA"].ToString().Equals(orchRequest.Properties.Parameters.Properties.orchestratorInstanceDetails.Properties.ClusterRootCA));
            Assert.True(jsonDeserialized["properties"]["orchestratorInstanceDetails"]["properties"]["apiServerEndpoint"].ToString().Equals(orchRequest.Properties.Parameters.Properties.orchestratorInstanceDetails.Properties.ApiServerEndpoint));
        }
    }
}
