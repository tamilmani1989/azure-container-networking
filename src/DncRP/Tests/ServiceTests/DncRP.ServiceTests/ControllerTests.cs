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
    using Microsoft.AspNetCore.Http;
    using Microsoft.AspNetCore.Mvc;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Configuration;
    using Microsoft.Extensions.Logging;
    using Moq;
    using Newtonsoft.Json.Linq;
    using Xunit;
    using Match = System.Text.RegularExpressions.Match;

    public class ControllerTests
    {
        private IHttpUtility httpUtility;
        private TestConfig testConfig;
        private IConfiguration configuration;
        private ClientIdentity clientIdentity;
        private DatastoreSettings datastoreSettings;
        private const string settingsFileName = "servicetests.settings.json";
        private const string connectStrRegex = "RunAs=App;AppId=(.*);TenantId=(.*);AppKey=(.*)";

        public ControllerTests()
        {
            this.httpUtility = new HttpUtility(new HttpClient());
            var builder = new ConfigurationBuilder()
              .SetBasePath(Directory.GetCurrentDirectory())
              .AddJsonFile(settingsFileName);

            configuration = builder.Build();
            this.testConfig = configuration.Get<TestConfig>();

            Match match = Regex.Match(this.testConfig.ConnectionString, connectStrRegex);
            if (!match.Success)
            {
                throw new ArgumentException($"Invalid connection string {this.testConfig.ConnectionString}");
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
        }

        [Fact]
        public async void TestControllerResource_Get_Put_Delete()
        {
            CurrentContext.SetContext(this.configuration["AzureEnvironment"], this.configuration["AzureRegion"]);
            CurrentContext.SetManagedIdentity(clientIdentity);

            IConnectionStringProvider connectionStringProvider = new ConnectionStringProvider(
                this.httpUtility,
                datastoreSettings);

            Mock<ILogger<CosmosDbDataStore>> logger = new Mock<ILogger<CosmosDbDataStore>>();
            Mock<ILogger<TableOperations>> tableOpLogger = new Mock<ILogger<TableOperations>>();
            Mock<ILogger<DncController>> dncCtrlLogger = new Mock<ILogger<DncController>>();

            CosmosDbDataStore dataStore = new CosmosDbDataStore(connectionStringProvider, logger.Object);
            dataStore.InitAsync().GetAwaiter().GetResult();
            ITableOperations tableOperations = new TableOperations(dataStore, tableOpLogger.Object);
            DncController dncController = new DncController(tableOperations, dncCtrlLogger.Object);

            string guid = Guid.NewGuid().ToString();
            string[] strs = guid.Split(new string[] { "-" }, StringSplitOptions.RemoveEmptyEntries);
            string rgName = strs[0];
            string ctrlName = strs[1];

            // Get resource and validate results
            ContentResult result = await dncController.GetDncResource(this.testConfig.SubscriptionId, rgName, ctrlName);
            Assert.True(result.StatusCode == (int)HttpStatusCode.NotFound, "Resource exists but expectation is that it should not");

            ControllerRequest request = new ControllerRequest
            {
                Parameters = new ControllerRequestParameters
                {
                    Type = ControllerType.Kubernetes,
                    Properties = new ControllerRequestProperties
                    {
                        ServerAppID = "546192d7-503f-477a-9cfe-4efc3ee2b6e1", // AAD ID for k8s apiserver auth
                        ServerTenantID = "da6192d7-503f-477a-9cfe-4efc3ee2b6c3", // Tenant id of above AAD ID
                        ClusterRootCA = "ddsadsad344mfdsfdl", // Root CA of k8s cluster
                        ApiServerEndpoint = "https://testk8s.cloudapp.net"  //apiserver url that dnc needs to watch on
                    }
                }
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
            Assert.True(!string.IsNullOrWhiteSpace(jsonDeserialized["id"].ToString()));
            Assert.True(!string.IsNullOrWhiteSpace(jsonDeserialized["name"].ToString()));
            Assert.True(!string.IsNullOrWhiteSpace(jsonDeserialized["location"].ToString()));
            Assert.True(!string.IsNullOrWhiteSpace(jsonDeserialized["type"].ToString()));
            Assert.True(jsonDeserialized["properties"]["state"].ToString().Equals(ProvisioningState.Succeeded.ToString()));
            Assert.True(!string.IsNullOrWhiteSpace(jsonDeserialized["properties"]["dncAppId"].ToString()));
            Assert.True(!string.IsNullOrWhiteSpace(jsonDeserialized["properties"]["dncEndpoint"].ToString()));

            // Get after Put
            // Get resource and validate results
            result = await dncController.GetDncResource(this.testConfig.SubscriptionId, rgName, ctrlName);
            Assert.True(HttpStatusCodeCheck.IsSuccess((int)result.StatusCode), "PutDncResource failed");
            Assert.True(result.StatusCode == (int)HttpStatusCode.OK, "PutDncResource failed created status check");
            jsonDeserialized = JObject.Parse(result.Content);
            Assert.True(!string.IsNullOrWhiteSpace(jsonDeserialized["id"].ToString()));
            Assert.True(!string.IsNullOrWhiteSpace(jsonDeserialized["name"].ToString()));
            Assert.True(!string.IsNullOrWhiteSpace(jsonDeserialized["location"].ToString()));
            Assert.True(!string.IsNullOrWhiteSpace(jsonDeserialized["type"].ToString()));
            Assert.True(jsonDeserialized["properties"]["state"].ToString().Equals(ProvisioningState.Succeeded.ToString()));
            Assert.True(!string.IsNullOrWhiteSpace(jsonDeserialized["properties"]["dncAppId"].ToString()));
            Assert.True(!string.IsNullOrWhiteSpace(jsonDeserialized["properties"]["dncEndpoint"].ToString()));

            // Delete resouce and validate result
            result = await dncController.DeleteDncResource(this.testConfig.SubscriptionId, rgName, ctrlName);
            Assert.True(HttpStatusCodeCheck.IsSuccess((int)result.StatusCode), "DeleteDncResource failed");
            Assert.True(result.StatusCode == (int)HttpStatusCode.OK, "DeleteDncResource failed created status check");
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
