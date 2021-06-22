// -----------------------------------------------------------------------------
// <copyright file="SubscriptionRegistrationTests.cs" company="Microsoft">
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
    using System.Net.Http.Formatting;
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
    using Newtonsoft.Json;
    using Newtonsoft.Json.Converters;
    using Newtonsoft.Json.Linq;
    using Xunit;
    using Match = System.Text.RegularExpressions.Match;

    [Collection("ServiceTestsSequential")]
    public class SubscriptionRegistrationTests
    {
        private IHttpUtility httpUtility;
        private IKeyVaultUtility keyVaultUtility;
        private TestConfig testConfig;
        private IConfiguration configuration;
        private const string settingsFileName = "servicetests.settings.json";
        private const string connectStrRegex = "RunAs=App;AppId=(.*);TenantId=(.*);AppKey=(.*)";
        private Mock<ILogger<SubscriptionRegistrationController>> subscriptionLogger;
        private ITableOperations tableOperations;

        public SubscriptionRegistrationTests()
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

            ClientIdentity clientIdentity = new ClientIdentity
            {
                AppId = match.Groups[1].Value,
                TenantId = match.Groups[2].Value,
                AppKey = match.Groups[3].Value,
                IdentityType = ClientIdentityType.SpClientSecret
            };

            DatastoreSettings datastoreSettings = new DatastoreSettings
            {
                SubscriptionId = this.testConfig.SubscriptionId,
                DBAccountName = this.testConfig.DbAccountName,
                ResourceGroup = this.testConfig.ResourceGroup,
                Region = this.testConfig.AzureRegion
            };

            string curDir = AppInfo.FolderPath;
            string cirFile = AppInfo.FilePath;

            CurrentContext.SetContext(this.configuration["AzureEnvironment"], this.configuration["AzureRegion"]);
            CurrentContext.SetManagedIdentity(clientIdentity);

            ClientIdentity armAccessId = new ClientIdentity
            {
                AppId = this.testConfig.DelegateSubnetClientCertificate.AppId,
                ClientCertificateName = this.testConfig.DelegateSubnetClientCertificate.ClientCertificateName,
                IdentityType = ClientIdentityType.SpClientCertificate
            };
            CurrentContext.SetArmAccessIdentity(armAccessId);

            Mock<IOptionsMonitor<DatastoreSettings>> optionsMonitorMock = new Mock<IOptionsMonitor<DatastoreSettings>>();
            optionsMonitorMock.Setup(o => o.CurrentValue).Returns(datastoreSettings);
            IConnectionStringProvider connectionStringProvider = new ConnectionStringProvider(
                this.httpUtility,
                optionsMonitorMock.Object);

            Mock<ILogger<CosmosDbDataStore>> logger = new Mock<ILogger<CosmosDbDataStore>>();
            Mock<ILogger<TableOperations>> tableOpLogger = new Mock<ILogger<TableOperations>>();
            subscriptionLogger = new Mock<ILogger<SubscriptionRegistrationController>>();

            CosmosDbDataStore dataStore = new CosmosDbDataStore(connectionStringProvider, logger.Object);
            dataStore.InitAsync().GetAwaiter().GetResult();
            tableOperations = new TableOperations(dataStore, tableOpLogger.Object, this.httpUtility, this.keyVaultUtility);
        }

        [Fact]
        public async void TestRegister_UnregisterSub()
        {
            // register Sub
            string subscriptionId = this.testConfig.SubscriptionId;
            SubscriptionRegistrationController subController = new SubscriptionRegistrationController(subscriptionLogger.Object, tableOperations, new TestHttpUtility(), new MockMetricsProvider());

            SubscriptionRegistrationRequest request = new SubscriptionRegistrationRequest()
            {
                State = SubscriptionRegistrationState.Registered.ToString(),
                RegistrationDate = DateTime.UtcNow
            };

            SetSubscriptionRegistrationHttpContext(subController, subscriptionId, HttpMethod.Put, request);
            ContentResult result = await subController.PutSubscriptionRegistration(
                subscriptionId);

            Assert.True(result.StatusCode == (int)HttpStatusCode.OK,
                $"Sub registration failed: {subscriptionId}");

            // Re-Register
            SetSubscriptionRegistrationHttpContext(subController, subscriptionId, HttpMethod.Put, request);
            result = await subController.PutSubscriptionRegistration(
                subscriptionId);

            Assert.True(result.StatusCode == (int)HttpStatusCode.OK,
                $"Sub registration failed: {subscriptionId}");


            // unregister
            request.State = SubscriptionRegistrationState.Unregistered.ToString();
            SetSubscriptionRegistrationHttpContext(subController, subscriptionId, HttpMethod.Put, request);
            result = await subController.PutSubscriptionRegistration(
                subscriptionId);

            Assert.True(result.StatusCode == (int)HttpStatusCode.OK,
                $"Sub unregistration failed: {subscriptionId}");
        }

        private static void SetSubscriptionRegistrationHttpContext(
            SubscriptionRegistrationController controller, 
            string subId, 
            HttpMethod method,
            SubscriptionRegistrationRequest request)
        {
            controller.ControllerContext = new ControllerContext();
            controller.ControllerContext.HttpContext = new DefaultHttpContext();
            controller.ControllerContext.HttpContext.Request.Host = new HostString($"localhost:5002");
            controller.ControllerContext.HttpContext.Request.Path = new PathString($"/subscriptions/{subId}");
            controller.ControllerContext.HttpContext.Request.Method = method.ToString();
            controller.ControllerContext.HttpContext.Request.PathBase = "";
            controller.ControllerContext.HttpContext.Request.Scheme = "https";

            if (request != null)
            {
                string json = JsonUtility.Serialize(request);
                MemoryStream stream = new MemoryStream(Encoding.UTF8.GetBytes(json));
                controller.ControllerContext.HttpContext.Request.Body = stream;
            }
        }

        internal static async Task RegisterSub(
            string subscriptionId,
            ITableOperations tableOperations)
        {
            Mock<ILogger<SubscriptionRegistrationController>> logger = new Mock<ILogger<SubscriptionRegistrationController>>();
            SubscriptionRegistrationController subController = new SubscriptionRegistrationController(logger.Object, tableOperations, new TestHttpUtility(), new MockMetricsProvider());

            SubscriptionRegistrationRequest request = new SubscriptionRegistrationRequest()
            {
                State = SubscriptionRegistrationState.Registered.ToString(),
                RegistrationDate = DateTime.UtcNow
            };

            SetSubscriptionRegistrationHttpContext(subController, subscriptionId, HttpMethod.Put, request);
            ContentResult result = await subController.PutSubscriptionRegistration(
                subscriptionId);

            Assert.True(result.StatusCode == (int)HttpStatusCode.OK, 
                $"Sub registration failed: {subscriptionId}");
        }
    }
}
