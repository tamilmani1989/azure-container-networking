// -----------------------------------------------------------------------------
// <copyright file="CosmosDbKeysTests.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP.ServiceTests
{
    using System.IO;
    using System.Net.Http;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Configuration;
    using Xunit;

    public class CosmosDbKeysTests
    {
        private IHttpUtility httpUtility;
        private TestConfig testConfig;
        private const string settingsFileName = "servicetests.settings.json";

        public CosmosDbKeysTests()
        {
            this.httpUtility = new HttpUtility(new HttpClient());

            IConfiguration Configuration;

            var builder = new ConfigurationBuilder()
              .SetBasePath(Directory.GetCurrentDirectory())
              .AddJsonFile(settingsFileName);

            Configuration = builder.Build();

            this.testConfig = Configuration.Get<TestConfig>();
        }

        [Fact]
        public async void TestGetCosmosDbKeys()
        {
            string azEnvName = "azurecloud";
            string connectionString = this.testConfig.KeyVaultConnectionString;

            IAzureAD azureAD = new AzureAD();
            string armEndpoint = AzureInfo.GetArmEndpoint(azEnvName);
            string token = await azureAD.GetAccessToken(connectionString, armEndpoint, azEnvName);
            Assert.True(string.IsNullOrWhiteSpace(token) == false);


            string subscriptionId = this.testConfig.SubscriptionId;
            string resourceGroup = this.testConfig.ResourceGroup;
            string dbAccountName = this.testConfig.DbAccountName;

            ICosmosDb cosmosDb = new CosmosDb(this.httpUtility);
            CosmosDbRWKeys keys = await cosmosDb.GetReadWriteKeys(subscriptionId, resourceGroup, dbAccountName, token, azEnvName);

            Assert.True(string.IsNullOrWhiteSpace(keys.PrimaryMasterKey) == false);
            Assert.True(string.IsNullOrWhiteSpace(keys.SecondaryMasterKey) == false);
        }
    }
}
