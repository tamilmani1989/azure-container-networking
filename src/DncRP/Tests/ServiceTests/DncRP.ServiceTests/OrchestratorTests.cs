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
    using Microsoft.Azure.ContainerNetworking.DncRP.Unittests;
    using global::Azure.Security.KeyVault.Secrets;
    using Microsoft.Extensions.Options;

    [Collection("ServiceTestsSequential")]
    public class OrchestratorTests
    {
        private IHttpUtility httpUtility;
        private IHttpUtilityFactory httpUtilityFactory;
        private IKeyVaultUtility keyVaultUtility;
        private TestConfig testConfig;
        private IConfiguration configuration;
        private ClientIdentity clientIdentity;
        private DatastoreSettings datastoreSettings;
        private OrchestratorController orchController;
        private DncController dncController;
        private ITableOperations tableOperations;
        private const string settingsFileName = "servicetests.settings.json";
        private const string connectStrRegex = "RunAs=App;AppId=(.*);TenantId=(.*);AppKey=(.*)";
        string location = "Test location";

        public OrchestratorTests()
        {
            this.httpUtility = new HttpUtility(new HttpClient());
            this.httpUtilityFactory = new HttpUtilityFactory();
            this.httpUtilityFactory.Register(Constants.HttpUtility, httpUtility);
            this.keyVaultUtility = new KeyVaultUtility();
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

            CurrentContext.SetKeyVaultUrl(this.testConfig.KeyVaultUrl);

            this.clientIdentity = new ClientIdentity
            {
                AppId = match.Groups[1].Value,
                TenantId = match.Groups[2].Value,
                AppKey = match.Groups[3].Value,
                IdentityType = ClientIdentityType.SpClientSecret
            };


            ClientIdentity azADIdentity = new ClientIdentity
            {
                AppId = "a91b1853-4403-4f54-b5cb-d1ea19d90c37", //DNC-RP APP ID
                TenantId = this.testConfig.ClientIdentityCertificate.TenantId,
                ClientCertificateName = this.testConfig.ClientIdentityCertificate.ClientCertificateName,
                IdentityType = ClientIdentityType.SpClientCertificate,
                KeyVaultUrl = this.configuration["KeyVaultUrl"]
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

            CurrentContext.SetArmAccessIdentity(azADIdentity);
            CurrentContext.SetKeyVaultUrl(this.configuration["KeyVaultUrl"]);

            DncSettings dncSettings = configuration.GetSection("DncSettings").Get<DncSettings>();
            if (dncSettings == null)
            {
                Console.WriteLine("DncSettings is null. Initialize to default values");
                dncSettings = new DncSettings();
            }
            CurrentContext.SetDncSettings(dncSettings);

            Mock<IOptionsMonitor<DatastoreSettings>> optionsMonitorMock = new Mock<IOptionsMonitor<DatastoreSettings>>();
            optionsMonitorMock.Setup(o => o.CurrentValue).Returns(datastoreSettings);
            IConnectionStringProvider connectionStringProvider = new ConnectionStringProvider(
                this.httpUtility,
                optionsMonitorMock.Object);

            Mock<ILogger<CosmosDbDataStore>> logger = new Mock<ILogger<CosmosDbDataStore>>();
            Mock<ILogger<TableOperations>> tableOpLogger = new Mock<ILogger<TableOperations>>();
            Mock<ILogger<OrchestratorController>> orchLogger = new Mock<ILogger<OrchestratorController>>();
            Mock<ILogger<DncController>> dncCtrlLogger = new Mock<ILogger<DncController>>();

            CosmosDbDataStore dataStore = new CosmosDbDataStore(connectionStringProvider, logger.Object);
            dataStore.InitAsync().GetAwaiter().GetResult();
            this.tableOperations = new TableOperations(dataStore, tableOpLogger.Object, this.httpUtility, this.keyVaultUtility);
            this.orchController = new OrchestratorController(tableOperations, httpUtilityFactory, orchLogger.Object);
            this.dncController = new DncController(tableOperations, dncCtrlLogger.Object);
        }

        [Fact]
        public async void TestOrchestratorResource_Get_Put_Delete()
        {
            try
            {
                // Change environment to PPE before test and restore it back after test finishes. THis is needed
                // since to access DB, we need to be in public cloud environment but to test orchestrator identity
                // we need access to PPE environment
                CurrentContext.SetContext("azurecloud", this.configuration["AzureRegion"]);

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
                    Location = location
                };

                await SubscriptionRegistrationTests.RegisterSub(this.testConfig.SubscriptionId, this.tableOperations);

                // Put controller resource and validate results
                this.SetControllerHttpContext(dncController, this.testConfig.SubscriptionId, rgName, ctrlName, HttpMethod.Put, request);
                result = await dncController.PutDncResource(this.testConfig.SubscriptionId, rgName, ctrlName);
                Assert.True(HttpStatusCodeCheck.IsSuccess((int)result.StatusCode), $"PutDncResource failed. {result.Content}");
                Assert.True(result.StatusCode == (int)HttpStatusCode.OK, "PutDncResource failed created status check");

                // Get controller resource and validate results
                result = await orchController.GetOrchestratorResource(this.testConfig.SubscriptionId, rgName, orchName);
                Assert.True(HttpStatusCodeCheck.IsResourceNotFound((int)result.StatusCode));

                OrchestratorRequest orchRequest = this.GetOrchestratorObject(this.GetControllerArmId(this.testConfig.SubscriptionId, rgName, ctrlName), this.testConfig.ResourceGroup);

                // Put orchestrator resource and validate results
                this.SetOrchestratorHttpContext(orchController, this.testConfig.SubscriptionId, rgName, orchName, HttpMethod.Put, orchRequest);
                result = await orchController.PutOrchestratorResource(this.testConfig.SubscriptionId, rgName, orchName);
                Assert.True(HttpStatusCodeCheck.IsSuccess((int)result.StatusCode), $"PutOrchestratorResource failed. {result.Content}");
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
            finally
            {
                CurrentContext.SetContext(this.configuration["AzureEnvironment"], this.configuration["AzureRegion"]);
            }
        }

        [Fact]
        public async void TestOrchestratorResource_Invalid_ControllerId()
        {
            string guid = Guid.NewGuid().ToString();
            string[] strs = guid.Split(new string[] { "-" }, StringSplitOptions.RemoveEmptyEntries);
            string rgName = strs[0];
            string orchName = strs[1];
            string ctrlName = strs[2];

            await SubscriptionRegistrationTests.RegisterSub(this.testConfig.SubscriptionId, this.tableOperations);

            // Get controller resource and validate results
            ContentResult result = await orchController.GetOrchestratorResource(this.testConfig.SubscriptionId, rgName, orchName);
            Assert.True(HttpStatusCodeCheck.IsResourceNotFound((int)result.StatusCode));

            OrchestratorRequest orchRequest = this.GetOrchestratorObject(this.GetInvalidControllerArmId(), this.testConfig.ResourceGroup);

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

        [Fact]
        public async void TestOrchestratorResource_Put_Delete_And_Validate_KeyVault()
        {
            try
            {
                // Change environment to PPE before test and restore it back after test finishes. THis is needed
                // since to access DB, we need to be in public cloud environment but to test orchestrator identity
                // we need access to PPE environment
                // CurrentContext.SetContext("ppe", this.configuration["AzureRegion"]);

                string guid = Guid.NewGuid().ToString();
                string[] strs = guid.Split(new string[] { "-" }, StringSplitOptions.RemoveEmptyEntries);
                string rgName = strs[0];
                string orchName = strs[1];
                string ctrlName = strs[2];
                Orchestrator expectedOrch = new Orchestrator(this.testConfig.SubscriptionId, rgName, orchName);

                await SubscriptionRegistrationTests.RegisterSub(this.testConfig.SubscriptionId, this.tableOperations);

                // Get controller resource and validate results
                ContentResult result = await dncController.GetDncResource(this.testConfig.SubscriptionId, rgName, ctrlName);
                Assert.True(result.StatusCode == (int)HttpStatusCode.NotFound, "Resource exists but expectation is that it should not");

                ControllerRequest request = new ControllerRequest
                {
                    Location = location
                };

                // Put controller resource and validate results
                this.SetControllerHttpContext(dncController, this.testConfig.SubscriptionId, rgName, ctrlName, HttpMethod.Put, request);
                result = await dncController.PutDncResource(this.testConfig.SubscriptionId, rgName, ctrlName);
                Assert.True(HttpStatusCodeCheck.IsSuccess((int)result.StatusCode), $"PutDncResource failed. {result.Content}");
                Assert.True(result.StatusCode == (int)HttpStatusCode.OK, "PutDncResource failed created status check");

                // Get controller resource and validate results
                result = await orchController.GetOrchestratorResource(this.testConfig.SubscriptionId, rgName, orchName);
                Assert.True(HttpStatusCodeCheck.IsResourceNotFound((int)result.StatusCode));

                OrchestratorRequest orchRequest = this.GetOrchestratorObject(this.GetControllerArmId(this.testConfig.SubscriptionId, rgName, ctrlName), this.testConfig.ResourceGroup);

                // Put orchestrator resource and validate results
                this.SetOrchestratorHttpContext(orchController, this.testConfig.SubscriptionId, rgName, orchName, HttpMethod.Put, orchRequest);
                result = await orchController.PutOrchestratorResource(this.testConfig.SubscriptionId, rgName, orchName);
                Assert.True(HttpStatusCodeCheck.IsSuccess((int)result.StatusCode), $"PutOrchestratorResource failed. {result.Content}");
                this.ValidateOrchestratorResponse(result.Content, orchRequest, expectedOrch);
                string resouceGuid = GetOrchestratorResouceGuid(result.Content);

                KeyVaultSecret secret = this.keyVaultUtility.GetSecret(CurrentContext.KeyVaultUrl, CurrentContext.ManagedIdentity, resouceGuid, null);
                Assert.True(string.IsNullOrWhiteSpace(secret.Value) == false);

                // Do another PUT on the same resouce
                this.SetOrchestratorHttpContext(orchController, this.testConfig.SubscriptionId, rgName, orchName, HttpMethod.Put, orchRequest);
                result = await orchController.PutOrchestratorResource(this.testConfig.SubscriptionId, rgName, orchName);
                Assert.True(HttpStatusCodeCheck.IsSuccess((int)result.StatusCode), $"PutOrchestratorResource failed. {result.Content}");
                this.ValidateOrchestratorResponse(result.Content, orchRequest, expectedOrch);
                string resouceGuid1 = GetOrchestratorResouceGuid(result.Content);

                // Validate resouce ID hasn't changed since partitionKey and rowkey are same due to same resouce
                Assert.True(resouceGuid == resouceGuid1);

                KeyVaultSecret secret1 = this.keyVaultUtility.GetSecret(CurrentContext.KeyVaultUrl, CurrentContext.ManagedIdentity, resouceGuid, null);
                Assert.True(string.IsNullOrWhiteSpace(secret.Value) == false);

                // Vaidate that KV has the same secret even after another PUT as it wasn't updated.
                Assert.True(secret.Properties.Version == secret1.Properties.Version);

                // Delete orchestrator resouce and validate result
                result = await orchController.DeleteOrchestratorResource(this.testConfig.SubscriptionId, rgName, orchName);
                Assert.True(HttpStatusCodeCheck.IsSuccess((int)result.StatusCode));

                // Check if secret exists in KV
                Assert.ThrowsAny<Exception>(() => this.keyVaultUtility.GetSecret(CurrentContext.KeyVaultUrl, CurrentContext.ManagedIdentity, resouceGuid, null));
                //Assert.True(string.IsNullOrWhiteSpace(secret.Value) == false);

                // Delete controller resouce and validate result
                result = await dncController.DeleteDncResource(this.testConfig.SubscriptionId, rgName, ctrlName);
                Assert.True(HttpStatusCodeCheck.IsSuccess((int)result.StatusCode));
            }
            finally
            {
                CurrentContext.SetContext(this.configuration["AzureEnvironment"], this.configuration["AzureRegion"]);
            }
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
            controller.ControllerContext.HttpContext.Request.Headers.Add("x-ms-client-tenant-id", "563c161a-f814-45eb-bac4-9d780e7f3ee6");
            controller.ControllerContext.HttpContext.Request.Headers.Add("x-ms-home-tenant-id", "563c161a-f814-45eb-bac4-9d780e7f3ee6");
            controller.ControllerContext.HttpContext.Request.Headers.Add("x-ms-identity-principal-id", "fa441e00-aa7d-44d1-8a4a-ee0b2af05bd7");
            controller.ControllerContext.HttpContext.Request.Headers.Add("x-ms-identity-url", "https://control-centralus.identity.azure-int.net/subscriptions/1e76d7b2-c2e3-44f1-b8a9-dc4a2264cae7/resourcegroups/rgopenaippe/providers/Microsoft.DelegatedNetwork/orchestratorInstances/orch1/credentials/v2/systemassigned?arpid=5ca39f01-a2ae-42e6-b1b0-9c133bd4c461&said=dc1798db-db55-4d74-81f4-748771171cd5");


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

        private OrchestratorRequest GetOrchestratorObject(string controllerArmId, string resourceGroup)
        {
            return new OrchestratorRequest
            {
                Location = location,
                Kind = OrchestratorKind.Kubernetes,
                IdentityParameter = new OrchestratorIdentityParameter
                {
                    IdentityType = MSIIdentityType.SystemAssigned
                },
                Properties = new OrchestratorProperties
                {
                    AppId = "546192d7-503f-477a-9cfe-4efc3ee2b6e1", // AAD ID for k8s apiserver auth
                    TenantId = "da6192d7-503f-477a-9cfe-4efc3ee2b6c3", // Tenant id of above AAD ID
                    ClusterRootCA = "ddsadsad344mfdsfdl", // Root CA of k8s cluster
                    ApiServerEndpoint = "https://testk8s.cloudapp.net",  //apiserver url that dnc needs to watch on
                    PrivateLinkId = $"/subscriptions/9b8218f9-902a-4d20-a65c-e98acec5362f/resourceGroups/{resourceGroup}/providers/Microsoft.Network/privateLinkServices/pltest",
                    ControllerDetails = new ControllerDetails
                    {
                        Id = controllerArmId
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
            Assert.True(jsonDeserialized["kind"].ToString().Equals(orchRequest.Kind.ToString()));
            Assert.True(jsonDeserialized["properties"]["provisioningState"].ToString().Equals(ProvisioningState.Succeeded.ToString()));
            Assert.True(!string.IsNullOrWhiteSpace(jsonDeserialized["properties"]["resourceGuid"].ToString()));
            Assert.True(jsonDeserialized["properties"]["orchestratorAppId"].ToString().Equals(orchRequest.Properties.AppId));
            Assert.True(jsonDeserialized["properties"]["orchestratorTenantId"].ToString().Equals(orchRequest.Properties.TenantId));
            Assert.True(jsonDeserialized["properties"]["clusterRootCA"].ToString().Equals(orchRequest.Properties.ClusterRootCA));
            if (orchRequest.Properties.PrivateLinkId == "")
            {
                Assert.True(jsonDeserialized["properties"]["apiServerEndpoint"].ToString().Equals(orchRequest.Properties.ApiServerEndpoint));
            }
            else
            {
                Assert.True(!string.IsNullOrWhiteSpace(jsonDeserialized["properties"]["apiServerEndpoint"].ToString()));
            }
            Assert.True(jsonDeserialized["properties"]["controllerDetails"]["id"].ToString().Equals(orchRequest.Properties.ControllerDetails.Id));
        }

        private string GetOrchestratorResouceGuid(string jsonResponse)
        {
            JObject jsonDeserialized = JObject.Parse(jsonResponse);
            return jsonDeserialized["properties"]["resourceGuid"].ToString();
        }
    }
}
