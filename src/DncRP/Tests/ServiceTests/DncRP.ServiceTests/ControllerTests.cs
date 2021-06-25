// -----------------------------------------------------------------------------
// <copyright file="ControllerTests.cs" company="Microsoft">
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
    using System.Threading.Tasks;
    using Microsoft.AspNetCore.Http;
    using Microsoft.AspNetCore.Mvc;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Azure.ContainerNetworking.DncRP.Unittests;
    using Microsoft.Extensions.Configuration;
    using Microsoft.Extensions.Logging;
    using Microsoft.Extensions.Options;
    using Moq;
    using Newtonsoft.Json.Linq;
    using Xunit;
    using Match = System.Text.RegularExpressions.Match;

    [Collection("ServiceTestsSequential")]
    public class ControllerTests
    {
        private IHttpUtility httpUtility;
        private IKeyVaultUtility keyVaultUtility;
        private TestConfig testConfig;
        private IConfiguration configuration;
        private ClientIdentity clientIdentity;
        private DatastoreSettings datastoreSettings;
        private const string settingsFileName = "servicetests.settings.json";
        private const string connectStrRegex = "RunAs=App;AppId=(.*);TenantId=(.*);AppKey=(.*)";
        private const string mdncGuid = "5c7e26be-7ff9-4aaf-9e16-0bc759605aea";
        private Mock<ILogger<DelegateSubnetController>> delSubLogger;
        private Mock<ILogger<SubscriptionRegistrationController>> subscriptionLogger;
        private Mock<ILogger<DncController>> dncCtrlLogger;
        private ITableOperations tableOperations;
        private CosmosDbDataStore dataStore;

        public ControllerTests()
        {
            this.httpUtility = new HttpUtility(new HttpClient());
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

            Mock<IOptionsMonitor<DatastoreSettings>> optionsMonitorMock = new Mock<IOptionsMonitor<DatastoreSettings>>();
            optionsMonitorMock.Setup(o => o.CurrentValue).Returns(datastoreSettings);
            IConnectionStringProvider connectionStringProvider = new ConnectionStringProvider(
                this.httpUtility,
                optionsMonitorMock.Object);

            Mock<ILogger<CosmosDbDataStore>> logger = new Mock<ILogger<CosmosDbDataStore>>();
            Mock<ILogger<TableOperations>> tableOpLogger = new Mock<ILogger<TableOperations>>();
            dncCtrlLogger = new Mock<ILogger<DncController>>();

            dataStore = new CosmosDbDataStore(connectionStringProvider, logger.Object);
            dataStore.InitAsync().GetAwaiter().GetResult();
            tableOperations = new TableOperations(dataStore, tableOpLogger.Object, this.httpUtility, this.keyVaultUtility);

            SubscriptionRegistrationTests.RegisterSub(this.testConfig.SubscriptionId, tableOperations).GetAwaiter().GetResult();
        }

        [Fact]
        public async void TestControllerResource_Get_Put_Delete()
        {
            string guid = Guid.NewGuid().ToString();
            string[] strs = guid.Split(new string[] { "-" }, StringSplitOptions.RemoveEmptyEntries);
            string rgName = strs[0];
            string ctrlName = strs[1];
            string location = "Test location";

            await this.TestControllerResouce_Get_Put_Delete_Internal(ctrlName, rgName, location, mdncGuid);
        }

        /// <summary>
        /// Test controller API for test scenario. For test scenario,
        /// controller name should be AzureContainerNetworking_<guid>
        /// </summary>
        [Fact]
        public async void TestControllerResource_Get_Put_Delete_CustomControllerName()
        {
            string guid = Guid.NewGuid().ToString();
            string[] strs = guid.Split(new string[] { "-" }, StringSplitOptions.RemoveEmptyEntries);
            string rgName = strs[0];
            string ctrlName = $"AzureContainerNetworking_{guid}";
            string location = "Test location";

            // Expected resource guid should be the one that is passed in the custom controller name
            await this.TestControllerResouce_Get_Put_Delete_Internal(ctrlName, rgName, location, guid);
        }

        private async Task TestControllerResouce_Get_Put_Delete_Internal(string ctrlName, string rgName, string location, string expectedCtrlResouceGuid)
        {
            DncController dncController = new DncController(tableOperations, dncCtrlLogger.Object, new MockMetricsProvider());

            string expectedId = $"/subscriptions/{this.testConfig.SubscriptionId}/resourceGroups/{rgName}/providers/Microsoft.DelegatedNetwork/controller/{ctrlName}";

            // Get resource and validate results
            ContentResult result = await dncController.GetDncResource(this.testConfig.SubscriptionId, rgName, ctrlName);
            Assert.True(result.StatusCode == (int)HttpStatusCode.NotFound, "Resource exists but expectation is that it should not");

            ControllerRequest request = new ControllerRequest
            {
                Location = location
            };

            // Put resource and validate results
            this.SetHttpContext(dncController, this.testConfig.SubscriptionId, rgName, ctrlName, HttpMethod.Put, request);
            result = await dncController.PutDncResource(this.testConfig.SubscriptionId, rgName, ctrlName);
            Assert.True(HttpStatusCodeCheck.IsSuccess((int)result.StatusCode), "PutDncResource failed");
            Assert.True(result.StatusCode == (int)HttpStatusCode.OK, "PutDncResource failed created status check");

            // We are validating raw json instead of deserializing it since Controller is derived class.
            // Deserialzing into Controller would be mean that Serialization of Controller also handles
            // derived type which will add type information to json. Since this json is supposed to be
            // returned to clients, we want to avoid adding our internal type information to json result.
            JObject jsonDeserialized = JObject.Parse(result.Content);
            Assert.True(jsonDeserialized["id"].ToString().Equals(expectedId));
            Assert.True(jsonDeserialized["name"].ToString().Equals(ctrlName));
            Assert.True(jsonDeserialized["location"].ToString().Equals(location));
            Assert.True(!string.IsNullOrWhiteSpace(jsonDeserialized["type"].ToString()));
            Assert.True(jsonDeserialized["properties"]["provisioningState"].ToString().Equals(ProvisioningState.Succeeded.ToString()));
            Assert.True(string.Equals(jsonDeserialized["properties"]["resourceGuid"].ToString(), expectedCtrlResouceGuid));
            Assert.True(!string.IsNullOrWhiteSpace(jsonDeserialized["properties"]["controllerProperties"]["dncAppId"].ToString()));
            Assert.True(!string.IsNullOrWhiteSpace(jsonDeserialized["properties"]["controllerProperties"]["dncEndpoint"].ToString()));

            // Get after Put
            // Get resource and validate results
            result = await dncController.GetDncResource(this.testConfig.SubscriptionId, rgName, ctrlName);
            Assert.True(HttpStatusCodeCheck.IsSuccess((int)result.StatusCode), "PutDncResource failed");
            Assert.True(result.StatusCode == (int)HttpStatusCode.OK, "PutDncResource failed created status check");
            jsonDeserialized = JObject.Parse(result.Content);
            Assert.True(jsonDeserialized["id"].ToString().Equals(expectedId));
            Assert.True(jsonDeserialized["name"].ToString().Equals(ctrlName));
            Assert.True(jsonDeserialized["location"].ToString().Equals(location));
            Assert.True(!string.IsNullOrWhiteSpace(jsonDeserialized["type"].ToString()));
            Assert.True(jsonDeserialized["properties"]["provisioningState"].ToString().Equals(ProvisioningState.Succeeded.ToString()));
            Assert.True(string.Equals(jsonDeserialized["properties"]["resourceGuid"].ToString(), expectedCtrlResouceGuid));
            Assert.True(!string.IsNullOrWhiteSpace(jsonDeserialized["properties"]["controllerProperties"]["dncAppId"].ToString()));
            Assert.True(!string.IsNullOrWhiteSpace(jsonDeserialized["properties"]["controllerProperties"]["dncEndpoint"].ToString()));

            // Delete resouce and validate result
            result = await dncController.DeleteDncResource(this.testConfig.SubscriptionId, rgName, ctrlName);
            Assert.True(HttpStatusCodeCheck.IsSuccess((int)result.StatusCode), "DeleteDncResource failed");
            Assert.True(result.StatusCode == (int)HttpStatusCode.OK, "DeleteDncResource failed created status check");
        }

        [Fact]
        public async void TestControllerDelete_With_OtherResources()
        {
            string guid = Guid.NewGuid().ToString();
            string[] strs = guid.Split(new string[] { "-" }, StringSplitOptions.RemoveEmptyEntries);
            string rgName = strs[0];
            string ctrlName = $"AzureContainerNetworking_{guid}";
            string vnetName = $"testvnet_{guid}";
            string subnetName = $"testsubnet_{guid}";
            string dsResourceName = $"testdsresource_{guid}";
            string orchResourceName = $"testorchresource_{guid}";
            string privateEndpointName = $"testpe_{guid}";
            string location = "Central US Euap";
            string subscriptionId = this.testConfig.SubscriptionId;

            CurrentContext.SetKeyVaultUrl(this.testConfig.KeyVaultUrl);
            DncController dncController = new DncController(tableOperations, dncCtrlLogger.Object, new MockMetricsProvider());
            ControllerRequest request = new ControllerRequest
            {
                Location = location
            };

            // Put resource and validate results
            this.SetHttpContext(dncController, subscriptionId, rgName, ctrlName, HttpMethod.Put, request);
            ContentResult result = await dncController.PutDncResource(subscriptionId, rgName, ctrlName);
            Assert.True(HttpStatusCodeCheck.IsSuccess((int)result.StatusCode), "PutDncResource failed");
            Assert.True(result.StatusCode == (int)HttpStatusCode.OK, "PutDncResource failed created status check");

            //put delegated subnet resource
            DelegateSubnetRequest dsRequest = new DelegateSubnetRequest()
            {
                Location = CurrentContext.AzureRegion,
                Properties = new DelegateSubnetRequestProperties()
                {
                    SubnetDetails = new SubnetDetails()
                    {
                        Id = $"/subscriptions/{subscriptionId}/resourceGroups/{rgName}/providers/Microsoft.Network/virtualNetworks/{vnetName}/subnets/{subnetName}"
                    },
                    ControllerDetails = new ControllerDetails()
                    {
                        Id = $"/subscriptions/{subscriptionId}/resourceGroups/{rgName}/providers/Microsoft.DelegatedNetwork/controller/{ctrlName}"
                    }
                }
            };

            await this.tableOperations.DelegatedSubnetConfigurationOperations.InsertOrUpdateAsync(
                subscriptionId,
                rgName,
                dsResourceName,
                Guid.NewGuid().ToString(),
                Guid.NewGuid().ToString(),
                DncSubnetStatus.Completed,
                ProvisioningState.Succeeded,
                dsRequest);

            // delete controller resource and it should fail
            try
            {
                result = await dncController.DeleteDncResource(subscriptionId, rgName, ctrlName);
            }
            catch (Exception ex)
            {
                int statusCode = (int)HttpStatusCode.InternalServerError;
                ErrorResponseMessageException errorException = ex as ErrorResponseMessageException;
                if (errorException != null)
                {
                    statusCode = (int)errorException.HttpStatus;
                }
                Assert.False(HttpStatusCodeCheck.IsSuccess((int)statusCode), "DeleteDncResource should fail");
                Assert.True(statusCode == (int)HttpStatusCode.BadRequest, "DeleteDncResource should fail with BadRequest");
            }

            // delete delegatedsubnet resource
            await this.tableOperations.DelegatedSubnetConfigurationOperations.DeleteAsync(
                subscriptionId,
                rgName,
                dsResourceName);
            
            // Delete controller resource and it should succeed
            result = await dncController.DeleteDncResource(subscriptionId, rgName, ctrlName);
            Assert.True(HttpStatusCodeCheck.IsSuccess((int)result.StatusCode), "DeleteDncResource failed");
            Assert.True(result.StatusCode == (int)HttpStatusCode.OK, "DeleteDncResource failed created status check");

            // put dnc controller
            this.SetHttpContext(dncController, subscriptionId, rgName, ctrlName, HttpMethod.Put, request);
            result = await dncController.PutDncResource(subscriptionId, rgName, ctrlName);
            Assert.True(HttpStatusCodeCheck.IsSuccess((int)result.StatusCode), "PutDncResource failed");
            Assert.True(result.StatusCode == (int)HttpStatusCode.OK, "PutDncResource failed created status check");

            //put orchestrator resource
            OrchestratorRequest orchRequest = new OrchestratorRequest
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
                    PrivateLinkId = $"/subscriptions/9b8218f9-902a-4d20-a65c-e98acec5362f/resourceGroups/{rgName}/providers/Microsoft.Network/privateLinkServices/pltest",
                    ControllerDetails = new ControllerDetails
                    {
                        Id = $"/subscriptions/{subscriptionId}/resourceGroups/{rgName}/providers/Microsoft.DelegatedNetwork/controller/{ctrlName}"
                    }
                }
            };
            
            await this.tableOperations.OrchestratorOperations.InsertOrReplaceOrchestratorAsync(
                subscriptionId,
                rgName,
                orchResourceName,
                privateEndpointName,
                "a43192d7-503f-477a-9cfe-4efc3ee2b6e1",
                null,
                orchRequest
                );

            try
            {
                result = await dncController.DeleteDncResource(subscriptionId, rgName, ctrlName);
            }
            catch (Exception ex)
            {
                int statusCode = (int)HttpStatusCode.InternalServerError;
                ErrorResponseMessageException errorException = ex as ErrorResponseMessageException;
                if (errorException != null)
                {
                    statusCode = (int)errorException.HttpStatus;
                }
                Assert.False(HttpStatusCodeCheck.IsSuccess((int)statusCode), "DeleteDncResource should fail");
                Assert.True(statusCode == (int)HttpStatusCode.BadRequest, "DeleteDncResource should fail with BadRequest");
            }

            // delete orchestrator resource
            await this.tableOperations.OrchestratorOperations.DeleteOrchestratorAsync(
                subscriptionId,
                rgName,
                orchResourceName);

            // Delete controller resource and it should succeed
            result = await dncController.DeleteDncResource(subscriptionId, rgName, ctrlName);
            Assert.True(HttpStatusCodeCheck.IsSuccess((int)result.StatusCode), "DeleteDncResource failed");
            Assert.True(result.StatusCode == (int)HttpStatusCode.OK, "DeleteDncResource failed created status check");
        }

        [Fact]
        public void TestController_ValidateRequestParameters()
        {
            var controller = new DncController(tableOperations, dncCtrlLogger.Object, new MockMetricsProvider());
            Assert.Throws<ErrorResponseMessageException>(() => controller.ValidateRequestParameters(
                new ControllerRequest() { Location = "" }));
        }

        /// <summary>
        /// Commented on purpose. Will be enabled in subsequent iteration
        /// </summary>
        /// <param name="controller"></param>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="controllerName"></param>
        /// <param name="method"></param>
        /// <param name="request"></param>
        //[Fact]
        //public async void TestControllerResource_ThrowsExeption()
        //{
        //    CurrentContext.SetContext(this.configuration["AzureEnvironment"], this.configuration["AzureRegion"]);
        //    CurrentContext.SetManagedIdentity(clientIdentity);

        //    IConnectionStringProvider connectionStringProvider = new ConnectionStringProvider(
        //        this.httpUtility,
        //        datastoreSettings);

        //    Mock<ILogger<CosmosDbDataStore>> logger = new Mock<ILogger<CosmosDbDataStore>>();
        //    Mock<ILogger<TableOperations>> tableOpLogger = new Mock<ILogger<TableOperations>>();
        //    Mock<ILogger<DncController>> dncCtrlLogger = new Mock<ILogger<DncController>>();

        //    CosmosDbDataStore dataStore = new CosmosDbDataStore(connectionStringProvider, logger.Object);
        //    dataStore.InitAsync().GetAwaiter().GetResult();
        //    ITableOperations tableOperations = new TableOperations(dataStore, tableOpLogger.Object);
        //    DncController dncController = new DncController(tableOperations, dncCtrlLogger.Object);

        //    string guid = Guid.NewGuid().ToString();
        //    string[] strs = guid.Split(new string[] { "-" }, StringSplitOptions.RemoveEmptyEntries);
        //    string rgName = strs[0];
        //    string ctrlName = strs[1];
        //    ControllerRequest request = new ControllerRequest
        //    {
        //        Parameters = new ControllerRequestParameters
        //        {
        //            Type = ControllerType.Kubernetes,
        //            Properties = new ControllerRequestProperties
        //            {
        //                ServerAppID = "546192d7-503f-477a-9cfe-4efc3ee2b6e1", // AAD ID for k8s apiserver auth
        //                ServerTenantID = "da6192d7-503f-477a-9cfe-4efc3ee2b6c3", // Tenant id of above AAD ID
        //                ClusterRootCA = "ddsadsad344mfdsfdl", // Root CA of k8s cluster
        //                ApiServerEndpoint = "https://testk8s.cloudapp.net"  //apiserver url that dnc needs to watch on
        //            }
        //        }
        //    };

        //    // Put resource and validate results
        //    this.SetHttpContext(dncController, this.testConfig.SubscriptionId, rgName, ctrlName, HttpMethod.Put, request);
        //    ContentResult result = await dncController.PutDncResource(this.testConfig.SubscriptionId, rgName, ctrlName);
        //    Assert.True(!HttpStatusCodeCheck.IsSuccess((int)result.StatusCode), "TestControllerResource_ThrowsExeption: PutDncResource failed");
        //    Assert.True(result.StatusCode == (int)HttpStatusCode.InternalServerError, "PutDncResource failed created status check");
        //}

        private void SetHttpContext(
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
    }
}
