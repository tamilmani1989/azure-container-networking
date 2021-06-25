// -----------------------------------------------------------------------------
// <copyright file="ConfigurationTests.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP.Unittests
{
    using System;
    using System.Collections.Generic;
    using System.IO;
    using System.Text;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Configuration;
    using Xunit;

    public class ConfigurationTests
    {
        [Fact]
        public void Test_LaunchSettingsJson_For_Environment_Variables()
        {
            var config = new ConfigurationBuilder()
              .SetBasePath(Directory.GetCurrentDirectory())
              .AddJsonFile(Path.Combine(Directory.GetCurrentDirectory(), "launchSettings.json"), optional: false)
              .Build();
            
            StringBuilder sb = new StringBuilder();
            IEnumerable<IConfigurationSection> envVars = config.GetSection("profiles:DncRP:environmentVariables").GetChildren();
            foreach (IConfigurationSection ev in envVars)
            {
                if (!string.IsNullOrWhiteSpace(ev.Value))
                {
                    sb.AppendLine($"Environment variable '{ev.Key}' is set in launchSettings.json in DncRP.csproj. Should be empty before check-in");
                }
            }

            Assert.True(string.IsNullOrWhiteSpace(sb.ToString()), sb.ToString());
        }

        [Fact]
        public void Test_AppSettingsDevelopmentJson_For_Secrets()
        {
            var config = new ConfigurationBuilder()
              .SetBasePath(Directory.GetCurrentDirectory())
              .AddJsonFile(Path.Combine(Directory.GetCurrentDirectory(), "appsettings.Development.json"), optional: false)
              .Build();

            string thumbprint = config["Thumbprint"];
            Assert.True(string.IsNullOrWhiteSpace(thumbprint),
                $"Thumbprint is set in appsettings.Development.json in DncRP.csproj. Should be empty before check-in");
        }
    }
}
