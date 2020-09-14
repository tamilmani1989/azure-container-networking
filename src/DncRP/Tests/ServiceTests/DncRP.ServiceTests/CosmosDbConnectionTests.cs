// -----------------------------------------------------------------------------
// <copyright file="CosmosDbConnectionTests.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP.ServiceTests
{
    using System;
    using System.IO;
    using System.Net.Http;
    using System.Text.RegularExpressions;
    using System.Threading.Tasks;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Azure.Cosmos.Table;
    using Microsoft.Extensions.Configuration;
    using Microsoft.Extensions.Logging;
    using Moq;
    using Xunit;
    using Match = System.Text.RegularExpressions.Match;

    public class CosmosDbConnectionTests
    {
        private IHttpUtility httpUtility;
        private TestConfig testConfig;
        private IConfiguration configuration;
        private const string settingsFileName = "servicetests.settings.json";
        private const string connectStrRegex = "RunAs=App;AppId=(.*);TenantId=(.*);AppKey=(.*)";

        public CosmosDbConnectionTests()
        {
            this.httpUtility = new HttpUtility(new HttpClient());
            var builder = new ConfigurationBuilder()
              .SetBasePath(Directory.GetCurrentDirectory())
              .AddJsonFile(settingsFileName);

            configuration = builder.Build();
            this.testConfig = configuration.Get<TestConfig>();
        }

        [Fact]
        public async void TestCosmosDbConnection()
        {
            Match match = Regex.Match(this.testConfig.ConnectionString, connectStrRegex);
            if (!match.Success)
            {
                throw new ArgumentException($"Invalid connection string {this.testConfig.ConnectionString}");
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

            CurrentContext.SetContext(this.configuration["AzureEnvironment"], this.configuration["AzureRegion"]);
            CurrentContext.SetManagedIdentity(clientIdentity);

            IConnectionStringProvider connectionStringProvider = new ConnectionStringProvider(
                this.httpUtility,
                datastoreSettings);

            Mock<ILogger<CosmosDbDataStore>> logger = new Mock<ILogger<CosmosDbDataStore>>();
            ITableDataStore dataStore = new CosmosDbDataStore(connectionStringProvider, logger.Object);
            Task t = dataStore.InitAsync();
            t.Wait();

            const string tableName = "TestTable";
            dataStore.CreateTable(tableName);

            TableEntity tableEntity = new TableEntity("testpartition", "testrow");
            TableResult tableResult = await dataStore.InsertOrMergeEntityAsync(tableName, tableEntity);
            Assert.True(tableResult.Result != null);
        }
    }
}
