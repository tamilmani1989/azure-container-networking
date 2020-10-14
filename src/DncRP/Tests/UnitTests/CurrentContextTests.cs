// -----------------------------------------------------------------------------
// <copyright file="CurrentContextTests.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP.Unittests
{
    using System.IO;
    using System.Net.Http;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Configuration;
    using Xunit;

    /// <summary>
    /// If your tests make use of CurrentContext class, please
    /// decorate the class with [Collection("Sequential")]
    /// </summary>
    [Collection("Sequential")]
    public class CurrentContextTests
    {
        private const string prodAppSettingsFileName = "appsettings.Development.json";
        private const string settingsFileName = "unittests.settings.json";
        private const string emptySettingsFileName = "unittests.settings.nullvalues.json";

        [Fact]
        public void TestContextIsInitializedFromSettingsFile()
        {
            var builder = new ConfigurationBuilder()
                .SetBasePath(Directory.GetCurrentDirectory())
                .AddJsonFile(settingsFileName);

            IConfiguration configuration = builder.Build();
            IImdsRetriever imdsRetriever = new TestImdsRetriever("ImdsEnv", "ImdsRegion");

            string expectedEnv = configuration["AzureEnvironment"];
            string expectedRegion = configuration["AzureRegion"];

            ContextInitializer.InitializeContext(configuration, imdsRetriever);

            Assert.True(CurrentContext.AzureEnvironment.Equals(expectedEnv));
            Assert.True(CurrentContext.AzureRegion.Equals(expectedRegion));
        }

        [Fact]
        public void TestContextIsInitializedFromImds()
        {
            var builder = new ConfigurationBuilder()
               .SetBasePath(Directory.GetCurrentDirectory())
               .AddJsonFile(emptySettingsFileName);

            IConfiguration configuration = builder.Build();

            string expectedEnv = "ImdsEnv";
            string expectedRegion = "ImdsRegion";

            IImdsRetriever imdsRetriever = new TestImdsRetriever(expectedEnv, expectedRegion);
            ContextInitializer.InitializeContext(configuration, imdsRetriever);

            Assert.True(CurrentContext.AzureEnvironment.Equals(expectedEnv.ToLower()));
            Assert.True(CurrentContext.AzureRegion.Equals(expectedRegion.ToLower()));
        }

        [Fact]
        public void Test_Production_AppSettings_Has_Identities()
        {
            var builder = new ConfigurationBuilder()
               .SetBasePath(Directory.GetCurrentDirectory())
               .AddJsonFile(prodAppSettingsFileName);

            IConfiguration configuration = builder.Build();

            string expectedEnv = "ImdsEnv";
            string expectedRegion = "ImdsRegion";

            IImdsRetriever imdsRetriever = new TestImdsRetriever(expectedEnv, expectedRegion);
            ContextInitializer.InitializeContext(configuration, imdsRetriever);

            Assert.True(CurrentContext.ManagedIdentity != null, "Managed identity not present in appsettings.json");
            Assert.True(CurrentContext.ArmAccessIdentity != null, "Arm access identity not present in appsettings.json");
        }
    }
}
