// -----------------------------------------------------------------------------
// <copyright file="AzureADTests.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP.ServiceTests
{
    using System;
    using System.IO;
    using System.Text.RegularExpressions;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Configuration;
    using Xunit;

    public class AzureADTests
    {
        private TestConfig testConfig;
        private const string settingsFileName = "servicetests.settings.json";
        private const string connectStrRegex = "RunAs=App;AppId=(.*);TenantId=(.*);AppKey=(.*)";

        public AzureADTests()
        {
            IConfiguration Configuration;

            var builder = new ConfigurationBuilder()
              .SetBasePath(Directory.GetCurrentDirectory())
              .AddJsonFile(settingsFileName);

            Configuration = builder.Build();

            this.testConfig = Configuration.Get<TestConfig>();
            this.testConfig.ClientIdentityCertificate = Configuration.GetSection("ClientIdentityCertificate").Get<TestClientIdentityCertificate>();
        }

        [Fact]
        public async void TestGetAzureADTokenForArm()
        {
            Match match = Regex.Match(this.testConfig.KeyVaultConnectionString, connectStrRegex);
            if (!match.Success)
            {
                throw new ArgumentException($"Invalid connection string {this.testConfig.KeyVaultConnectionString}");
            }

            ClientIdentity kvIdentity = new ClientIdentity
            {
                AppId = match.Groups[1].Value,
                TenantId = match.Groups[2].Value,
                AppKey = match.Groups[3].Value,
                IdentityType = ClientIdentityType.SpClientSecret
            };

            ClientIdentity azADIdentity = new ClientIdentity
            {
                AppId = this.testConfig.ClientIdentityCertificate.AppId,
                TenantId = this.testConfig.ClientIdentityCertificate.TenantId,
                ClientCertificateName = this.testConfig.ClientIdentityCertificate.ClientCertificateName,
                IdentityType = ClientIdentityType.SpClientCertificate,
                KeyVaultUrl = this.testConfig.KeyVaultUrl
            };

            string azEnvName = this.testConfig.ClientIdentityCertificate.AzureEnvironment;
            IAzureAD azureAD = new AzureAD();
            string token = await azureAD.GetAccessTokenForArm(
                kvIdentity, azADIdentity, this.testConfig.ClientIdentityCertificate.TenantId, azEnvName);
            Assert.True(string.IsNullOrWhiteSpace(token) == false);
        }
    }
}
