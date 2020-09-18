// -----------------------------------------------------------------------------
// <copyright file="DelegateSubnetTests.cs" company="Microsoft">
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
    using Moq;
    using Newtonsoft.Json;
    using Newtonsoft.Json.Converters;
    using Newtonsoft.Json.Linq;
    using Xunit;
    using Match = System.Text.RegularExpressions.Match;

    public class DelegateSubnetTests
    {
        private IHttpUtility httpUtility;
        private TestConfig testConfig;
        private IConfiguration configuration;
        private const string settingsFileName = "servicetests.settings.json";
        private const string connectStrRegex = "RunAs=App;AppId=(.*);TenantId=(.*);AppKey=(.*)";
        private Mock<ILogger<DelegateSubnetController>> delSubLogger;
        private ITableOperations tableOperations;

        public DelegateSubnetTests()
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
                KeyVaultCertPath = this.testConfig.DelegateSubnetClientCertificate.KeyVaultCertPath,
                IdentityType = ClientIdentityType.SpClientCertificate
            };
            CurrentContext.SetArmAccessIdentity(armAccessId);

            IConnectionStringProvider connectionStringProvider = new ConnectionStringProvider(
                this.httpUtility,
                datastoreSettings);

            Mock<ILogger<CosmosDbDataStore>> logger = new Mock<ILogger<CosmosDbDataStore>>();
            Mock<ILogger<TableOperations>> tableOpLogger = new Mock<ILogger<TableOperations>>();
            delSubLogger = new Mock<ILogger<DelegateSubnetController>>();

            CosmosDbDataStore dataStore = new CosmosDbDataStore(connectionStringProvider, logger.Object);
            dataStore.InitAsync().GetAwaiter().GetResult();
            tableOperations = new TableOperations(dataStore, tableOpLogger.Object);
        }

        [Fact]
        public async void TestDelegateSubnetController_Get_Put_Delete()
        {
            // Pass TestHttpUtility for NRP operations.
            DelegateSubnetController delegateSubnetController = new DelegateSubnetController(delSubLogger.Object, tableOperations, new TestHttpUtility());

            Guid testGuid = Guid.NewGuid();
            string subscriptionId = this.testConfig.SubscriptionId;
            string resourceGroup = $"testRG_{testGuid}";
            string vnetName = $"testvnet_{testGuid}";
            string subnetName = $"testsubnet_{testGuid}";
            string resourceName = $"testresource_{testGuid}";

            // Get resource and validate results
            await GetAndValidateDelegateSubnetAsync(delegateSubnetController, 
                subscriptionId,
                resourceGroup,
                vnetName,
                subnetName,
                resourceName,
                HttpStatusCode.NotFound);

            // Put resource and valudate results
            DelegateSubnetResponse response = await PutAndValidateAsync(
                delegateSubnetController,
                subscriptionId,
                resourceGroup,
                vnetName,
                subnetName,
                resourceName,
                HttpStatusCode.OK);


            // Get after Put
            // Get resource and valudate results
            response = await GetAndValidateDelegateSubnetAsync(delegateSubnetController,
                subscriptionId,
                resourceGroup,
                vnetName,
                subnetName,
                resourceName,
                HttpStatusCode.OK,
                response);

            // Idempotent Put
            // Put resource and valudate results
            response = await PutAndValidateAsync(
                delegateSubnetController,
                subscriptionId,
                resourceGroup,
                vnetName,
                subnetName,
                resourceName,
                HttpStatusCode.OK,
                response);

            // Failed to Put DelegateSubnet again using different ResurceName
            await PutAndValidateAsync(
                delegateSubnetController,
                subscriptionId,
                resourceGroup,
                vnetName,
                subnetName,
                "TestResource2",
                HttpStatusCode.BadRequest,
                response);
        }

        private async Task<DelegateSubnetResponse> GetAndValidateDelegateSubnetAsync(
            DelegateSubnetController delegateSubnetController,
            string subscriptionId,
            string resourceGroup,
            string vnetName,
            string subnetName,
            string resourceName,
            HttpStatusCode expectedStatusCode,
            DelegateSubnetResponse previousResponse = null)
        {
            this.SetHttpContext(delegateSubnetController, subscriptionId, resourceGroup, vnetName, subnetName, resourceName, HttpMethod.Get, null);
            ContentResult result = await delegateSubnetController.GetDelegateSubnet(
                subscriptionId,
                resourceGroup,
                vnetName,
                subnetName,
                resourceName);
            Assert.True(result.StatusCode == (int)expectedStatusCode,
                $"Failed GetDelegateSubnet, expectedCode: {(int)expectedStatusCode}, actual: {result.StatusCode} result: {result}");

            DelegateSubnetResponse response = null;
            if (expectedStatusCode == HttpStatusCode.OK)
            {
                response = ValidateDelegateResponse(result, previousResponse);
            }

            return response;
        }

        private async Task<DelegateSubnetResponse> PutAndValidateAsync(
            DelegateSubnetController delegateSubnetController,
            string subscriptionId, 
            string resourceGroup, 
            string vnetName, 
            string subnetName, 
            string resourceName, 
            HttpStatusCode expectedStatusCode,
            DelegateSubnetResponse previousResponse = null)
        {
            this.SetHttpContext(delegateSubnetController, subscriptionId, resourceGroup, vnetName, subnetName, resourceName, HttpMethod.Put, null);
            ContentResult result = await delegateSubnetController.PutDelegateSubnet(
                subscriptionId,
                resourceGroup,
                vnetName,
                subnetName,
                resourceName);

            Assert.True(result.StatusCode == (int)expectedStatusCode,
                $"Failed PutDelegateSubnet, expectedCode: {(int)expectedStatusCode}, actual: {result.StatusCode} result: {result}");
            DelegateSubnetResponse response = null;
            if (expectedStatusCode == HttpStatusCode.OK)
            {
                response = ValidateDelegateResponse(result, previousResponse);
            }

            return response;
        }

        private DelegateSubnetResponse ValidateDelegateResponse(ContentResult result, DelegateSubnetResponse previousResponse)
        {
            DelegateSubnetResponse response = JsonUtility.Deserialize<DelegateSubnetResponse>(result.Content);
            Console.WriteLine($"DelegateSubnet Response: {response}");

            Assert.True(!string.IsNullOrWhiteSpace(response.Id));
            Assert.True(!string.IsNullOrWhiteSpace(response.Name));
            Assert.True(!string.IsNullOrWhiteSpace(response.Location));
            Assert.True(!string.IsNullOrWhiteSpace(response.Type));
            Assert.NotNull(response.Properties);
            DelegateSubnetResponseProperties properties = response.Properties as DelegateSubnetResponseProperties;

            //Assert.True(properties.State.Equals(ProvisioningState.Succeeded));
            Assert.True(!string.IsNullOrWhiteSpace(properties.ResourceGuid));

            if (previousResponse != null)
            {
                Assert.True(response.Id == previousResponse.Id,
                    $" Latest Id: {response.Id}, Previous Id: {previousResponse.Id}");
                Assert.True(response.Name == previousResponse.Name,
                    $" Latest Name: {response.Name}, Previous Name: {previousResponse.Name}");
                Assert.True(response.Location == previousResponse.Location,
                    $" Latest Location: {response.Location}, Previous Location: {previousResponse.Location}");
                Assert.True(response.Type == previousResponse.Type,
                    $" Latest Type: {response.Type}, Previous Type: {previousResponse.Type}");

                DelegateSubnetResponseProperties prevProperties = previousResponse.Properties as DelegateSubnetResponseProperties;
                Assert.True(prevProperties.ResourceGuid == properties.ResourceGuid,
                    $"ResourceGuid changed unexpectedly. Latest ResourceGuid: {properties.ResourceGuid}, Previous ResourceGuid: {prevProperties.ResourceGuid}");
            }

            return response;
        }

        private void SetHttpContext(
            DelegateSubnetController controller, 
            string subId, 
            string resourceGroup, 
            string vnetName,
            string subnet,
            string resourceName,
            HttpMethod method,
            DelegateSubnetRequest request)
        {
            controller.ControllerContext = new ControllerContext();
            controller.ControllerContext.HttpContext = new DefaultHttpContext();
            controller.ControllerContext.HttpContext.Request.Host = new HostString($"localhost:5002");
            controller.ControllerContext.HttpContext.Request.Path = new PathString($"/subscriptions/{subId}/resourceGroups/{resourceGroup}/providers/Microsoft.Network/virtualNetworks/{vnetName}/subnets/{subnet}/providers/Microsoft.DelegatedNetwork/delegatedSubnets/{resourceName}");
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
    }
}
