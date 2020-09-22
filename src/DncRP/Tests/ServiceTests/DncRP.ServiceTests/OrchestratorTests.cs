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

    public class OrchestratorTests
    {
        private IHttpUtility httpUtility;
        private TestConfig testConfig;
        private IConfiguration configuration;
        private ClientIdentity clientIdentity;
        private DatastoreSettings datastoreSettings;
        private const string settingsFileName = "servicetests.settings.json";
        private const string connectStrRegex = "RunAs=App;AppId=(.*);TenantId=(.*);AppKey=(.*)";

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
        }

        [Fact]
        public async void TestOrchestratorResource_Get_Put_Delete()
        {
            CurrentContext.SetContext(this.configuration["AzureEnvironment"], this.configuration["AzureRegion"]);
            CurrentContext.SetManagedIdentity(clientIdentity);

            IConnectionStringProvider connectionStringProvider = new ConnectionStringProvider(
                this.httpUtility,
                datastoreSettings);

            Mock<ILogger<CosmosDbDataStore>> logger = new Mock<ILogger<CosmosDbDataStore>>();
            Mock<ILogger<TableOperations>> tableOpLogger = new Mock<ILogger<TableOperations>>();
            Mock<ILogger<OrchestratorController>> orchLogger = new Mock<ILogger<OrchestratorController>>();

            CosmosDbDataStore dataStore = new CosmosDbDataStore(connectionStringProvider, logger.Object);
            dataStore.InitAsync().GetAwaiter().GetResult();
            ITableOperations tableOperations = new TableOperations(dataStore, tableOpLogger.Object);
            OrchestratorController orchController = new OrchestratorController(tableOperations, orchLogger.Object);

            string guid = Guid.NewGuid().ToString();
            string[] strs = guid.Split(new string[] { "-" }, StringSplitOptions.RemoveEmptyEntries);
            string rgName = strs[0];
            string ctrlName = strs[1];

            // Get resource and validate results
            ContentResult result = await orchController.GetOrchestratorResource(this.testConfig.SubscriptionId, rgName, ctrlName);
            Assert.True(HttpStatusCodeCheck.IsInternalServerError((int)result.StatusCode));

            OrchestratorRequest request = new OrchestratorRequest
            {
                Properties = new OrchestratorArmProperties
                {
                    Parameters = new OrchestratorRequestParameters
                    {
                        Location = "TestLocation",
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
                                Id = ""
                            }
                        }
                    }
                }
            };

            // Put resource and validate results
            this.SetHttpContext(orchController, this.testConfig.SubscriptionId, rgName, ctrlName, HttpMethod.Put, request);
            result = await orchController.PutOrchestratorResource(this.testConfig.SubscriptionId, rgName, ctrlName);
            Assert.True(HttpStatusCodeCheck.IsInternalServerError((int)result.StatusCode));

            // Get after Put. Get resource and validate results
            result = await orchController.GetOrchestratorResource(this.testConfig.SubscriptionId, rgName, ctrlName);
            Assert.True(HttpStatusCodeCheck.IsInternalServerError((int)result.StatusCode));

            // Delete resouce and validate result
            result = await orchController.DeleteOrchestratorResource(this.testConfig.SubscriptionId, rgName, ctrlName);
            Assert.True(HttpStatusCodeCheck.IsInternalServerError((int)result.StatusCode));
        }

        private void SetHttpContext(
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
    }
}
