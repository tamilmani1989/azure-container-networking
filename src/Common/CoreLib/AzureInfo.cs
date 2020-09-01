// -----------------------------------------------------------------------------
// <copyright file="AzureInfo.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using System.Collections.Generic;
    using System.IO;
    using System.Linq;
    using Microsoft.Extensions.Configuration;

    public static class AzureInfo
    {
        private static IConfigurationRoot config;
        private const string settingsFileName = "azuresettings.json";

        static AzureInfo()
        {
            config = new ConfigurationBuilder()
               .SetBasePath(Directory.GetCurrentDirectory())
               .AddJsonFile(settingsFileName, optional: false)
               .Build();
        }

        public static IDictionary<string, string> GetArmEndpointsByAzureEnvironment()
        {
            return config.GetSection("ArmEndpoints").GetChildren().ToDictionary(x => x.Key, y => y.Value);
        }

        public static IEnumerable<string> GetAzureEnvironments()
        {
            return config.GetSection("AzureEnvironments").Get<string[]>();
        }

        public static IDictionary<string, string> GetAzureADInstancesByAzureEnvironment()
        {
            return config.GetSection("AzureAdInstances").GetChildren().ToDictionary(x => x.Key, y => y.Value);
        }

        public static string GetCosmosDbListKeyUrlFormat()
        {
            return config["CosmosDb:ListKeysUrlFormat"];
        }
    }
}
