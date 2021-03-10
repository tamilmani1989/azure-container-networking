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
    using Microsoft.Extensions.Options;
    using Moq;
    using Newtonsoft.Json;
    using Newtonsoft.Json.Converters;
    using Newtonsoft.Json.Linq;
    using Xunit;
    using Match = System.Text.RegularExpressions.Match;

    [Collection("ServiceTestsSequential")]
    public class DelegateSubnetTests
    {
        private IHttpUtility httpUtility;
        private IKeyVaultUtility keyVaultUtility;
        private TestConfig testConfig;
        private IConfiguration configuration;
        private const string settingsFileName = "servicetests.settings.json";
        private const string connectStrRegex = "RunAs=App;AppId=(.*);TenantId=(.*);AppKey=(.*)";
        private Mock<ILogger<DelegateSubnetController>> delSubLogger;
        private Mock<ILogger<SubscriptionRegistrationController>> subscriptionLogger;
        private ITableOperations tableOperations;

        public DelegateSubnetTests()
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
                KeyVaultCertPath = this.testConfig.DelegateSubnetClientCertificate.KeyVaultCertPath,
                IdentityType = ClientIdentityType.SpClientCertificate
            };
            CurrentContext.SetArmAccessIdentity(armAccessId);

            DncSettings dncSettings = configuration.GetSection("DncSettings").Get<DncSettings>();
            CurrentContext.SetDncSettings(dncSettings);

            Mock<IOptionsMonitor<DatastoreSettings>> optionsMonitorMock = new Mock<IOptionsMonitor<DatastoreSettings>>();
            optionsMonitorMock.Setup(o => o.CurrentValue).Returns(datastoreSettings);
            IConnectionStringProvider connectionStringProvider = new ConnectionStringProvider(
                this.httpUtility,
                optionsMonitorMock.Object);

            Mock<ILogger<CosmosDbDataStore>> logger = new Mock<ILogger<CosmosDbDataStore>>();
            Mock<ILogger<TableOperations>> tableOpLogger = new Mock<ILogger<TableOperations>>();
            delSubLogger = new Mock<ILogger<DelegateSubnetController>>();
            subscriptionLogger = new Mock<ILogger<SubscriptionRegistrationController>>();

            CosmosDbDataStore dataStore = new CosmosDbDataStore(connectionStringProvider, logger.Object);
            dataStore.InitAsync().GetAwaiter().GetResult();
            tableOperations = new TableOperations(dataStore, tableOpLogger.Object, this.httpUtility, this.keyVaultUtility);
        }

        [Fact]
        public async void TestDelegateSubnetController_Get_Put_Delete()
        {
            // register Sub
            string subscriptionId = this.testConfig.SubscriptionId;
            SubscriptionRegistrationTests.RegisterSub(subscriptionId, tableOperations);

            // Pass TestHttpUtility for NRP operations.
            DelegateSubnetController delegateSubnetController = new DelegateSubnetController(delSubLogger.Object, tableOperations, new TestHttpUtility(), false);

            Guid testGuid = Guid.NewGuid();
            string resourceGroup = $"testRG_{testGuid}";
            string vnetName = $"testvnet_{testGuid}";
            string subnetName = $"testsubnet_{testGuid}";
            string resourceName = $"testresource_{testGuid}";
            string controllerName = $"testcontroller_{testGuid}";
            string location = "CentralUSEuap";

            ControllerRequest controllerRequest = new ControllerRequest
            {
                Properties = new ArmProperties
                {
                    Parameters = new ControllerRequestParameters
                    {
                        Location = location
                    }
                }
            };

            await tableOperations.ControllerConfiguration.InsertOrReplaceDncControllerAsync(
                subscriptionId,
                resourceGroup,
                controllerName,
                controllerRequest
            );

            DncRP.Controller controller = await tableOperations.ControllerConfiguration.GetDncControllerAsync($"/subscriptions/{subscriptionId}/resourceGroups/{resourceGroup}/providers/Microsoft.DelegatedNetwork/controller/{controllerName}");
            Assert.True(controller != null);
            ControllerProperties controllerProperties = (ControllerProperties)controller.Properties;
            Assert.True(controllerProperties.PropertiesList.DncEndpoint == $"https://openai-{CurrentContext.AzureRegion}.dnc.azure.com");

            // Get resource and validate results
            await GetAndValidateDelegateSubnetAsync(delegateSubnetController, 
                subscriptionId,
                resourceGroup,
                resourceName,
                HttpStatusCode.NotFound,
                DncSubnetStatus.New);

           await PutAndValidateAsync(
                delegateSubnetController,
                subscriptionId,
                resourceGroup,
                vnetName,
                subnetName,
                resourceName,
                controllerName+"bad",
                HttpStatusCode.NotFound);

           await delegateSubnetController.DeleteDelegateSubnet(
                subscriptionId,
                resourceGroup,
                resourceName);
            
			// Put resource and valudate results
            DelegateSubnetResponse response = await PutAndValidateAsync(
                delegateSubnetController,
                subscriptionId,
                resourceGroup,
                vnetName,
                subnetName,
                resourceName,
                controllerName,
                HttpStatusCode.OK);


            // Get after Put
            // Get resource and valudate results
            response = await GetAndValidateDelegateSubnetAsync(delegateSubnetController,
                subscriptionId,
                resourceGroup,
                resourceName,
                HttpStatusCode.OK,
                DncSubnetStatus.Completed,
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
                controllerName,
                HttpStatusCode.OK,
                response);

            // TODO Failed to Put DelegateSubnet again using different ResurceName
            //await PutAndValidateAsync(
            //    delegateSubnetController,
            //    subscriptionId,
            //    resourceGroup,
            //    vnetName,
            //    subnetName,
            //    "TestResource2",
            //    HttpStatusCode.BadRequest,
            //    response);

            // Delete the DelegateSubnet
            await delegateSubnetController.DeleteDelegateSubnet(
                subscriptionId,
                resourceGroup,
                resourceName);

            // Get resource and validate results
            await GetAndValidateDelegateSubnetAsync(delegateSubnetController,
                subscriptionId,
                resourceGroup,
                resourceName,
                HttpStatusCode.NotFound,
                DncSubnetStatus.New);
        }

        private async Task<DelegateSubnetResponse> GetAndValidateDelegateSubnetAsync(
            DelegateSubnetController delegateSubnetController,
            string subscriptionId,
            string resourceGroup,
            string resourceName,
            HttpStatusCode expectedStatusCode,
            string expectedSubnetState,
            DelegateSubnetResponse previousResponse = null)
        {
            this.SetHttpContext(delegateSubnetController, subscriptionId, resourceGroup, resourceName, HttpMethod.Get, null);
            ContentResult result = await delegateSubnetController.GetDelegateSubnet(
                subscriptionId,
                resourceGroup,
                resourceName);
            Assert.True(result.StatusCode == (int)expectedStatusCode,
                $"Failed GetDelegateSubnet, expectedCode: {(int)expectedStatusCode}, actual: {result.StatusCode} result: {result}");

            DelegateSubnetResponse response = null;
            if (expectedStatusCode == HttpStatusCode.OK)
            {
                response = ValidateDelegateResponse(result, previousResponse);
                DelegateSubnet delegateSubnet = await tableOperations.DelegatedSubnetConfigurationOperations.TryGetDelegatedSubnetConfigurationAsync(
                                                 subscriptionId,
                                                 resourceGroup,
                                                 resourceName);
                Assert.True(delegateSubnet.SubnetJoinStatus.Equals(expectedSubnetState));
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
            string controllerName,
            HttpStatusCode expectedStatusCode,
            DelegateSubnetResponse previousResponse = null)
        {
            DelegateSubnetRequest request = new DelegateSubnetRequest()
            {
                Properties = new DeletagetSubnetArmProperties()
                {
                    Parameters = new DelegateRequestParameters()
                    {
                        Location = CurrentContext.AzureRegion,
                        Properties = new DelegateSubnetRequestProperties()
                        {
                            SubnetDetails = new SubnetDetails()
                            {
                                Id = $"/subscriptions/{subscriptionId}/resourceGroups/{resourceGroup}/providers/Microsoft.Network/virtualNetworks/{vnetName}/subnets/{subnetName}"
                            },
                            ControllerDetails = new ControllerDetails()
                            {
                                Id = $"/subscriptions/{subscriptionId}/resourceGroups/{resourceGroup}/providers/Microsoft.DelegatedNetwork/controller/{controllerName}"
                            }
                        }
                    }
                }
            };

            this.SetHttpContext(delegateSubnetController, subscriptionId, resourceGroup, resourceName, HttpMethod.Put, request);
            ContentResult result = await delegateSubnetController.PutDelegateSubnet(
                subscriptionId,
                resourceGroup,
                resourceName);

            Assert.True(result.StatusCode == (int)expectedStatusCode,
                $"Failed PutDelegateSubnet, expectedCode: {(int)expectedStatusCode}, actual: {result.StatusCode} result: {result.Content}");
            DelegateSubnetResponse response = null;
            if (expectedStatusCode == HttpStatusCode.OK)
            {
                response = ValidateDelegateResponse(result, previousResponse, request);
            }

            return response;
        }

        private DelegateSubnetResponse ValidateDelegateResponse(ContentResult result, DelegateSubnetResponse previousResponse, DelegateSubnetRequest request = null)
        {
            DelegateSubnetResponse response = JsonUtility.Deserialize<DelegateSubnetResponse>(result.Content);
            Console.WriteLine($"DelegateSubnet Response: {response}");

            Assert.True(!string.IsNullOrWhiteSpace(response.Id));
            Assert.True(!string.IsNullOrWhiteSpace(response.Name));
            Assert.True(!string.IsNullOrWhiteSpace(response.Type));
            Assert.True(response.Location == CurrentContext.AzureRegion);

            Assert.NotNull(response.Properties);
            DelegateSubnetResponseProperties properties = response.Properties as DelegateSubnetResponseProperties;

            Assert.True(properties.State.Equals(ProvisioningState.Succeeded.ToString()));
            Assert.True(!string.IsNullOrWhiteSpace(properties.ResourceGuid));
            Assert.NotNull(properties.SubnetDetails);
            Assert.True(!string.IsNullOrWhiteSpace(properties.SubnetDetails.Id));

            Assert.NotNull(properties.ControllerDetails);
            Assert.True(!string.IsNullOrWhiteSpace(properties.ControllerDetails.Id));

            if (request != null)
            {
                Assert.True(properties.SubnetDetails.Id == request.Properties.Parameters.Properties.SubnetDetails.Id,
                    $" Latest Subnet Id: {properties.SubnetDetails.Id}, Previous SubnetId Id: {request.Properties.Parameters.Properties.SubnetDetails.Id}");

                Assert.True(properties.ControllerDetails.Id == request.Properties.Parameters.Properties.ControllerDetails.Id,
                    $" Latest Controller Id: {properties.ControllerDetails.Id}, Previous Controller Id: {request.Properties.Parameters.Properties.ControllerDetails.Id}");
            }

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
            string resourceName,
            HttpMethod method,
            DelegateSubnetRequest request)
        {
            controller.ControllerContext = new ControllerContext();
            controller.ControllerContext.HttpContext = new DefaultHttpContext();
            controller.ControllerContext.HttpContext.Request.Host = new HostString($"localhost:5002");
            controller.ControllerContext.HttpContext.Request.Path = new PathString($"/subscriptions/{subId}/resourceGroups/{resourceGroup}/providers/Microsoft.DelegatedNetwork/delegatedSubnets/{resourceName}");
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
