// -----------------------------------------------------------------------------
// <copyright file="AzureInfo.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using System;
    using System.Collections.Generic;
    using System.IO;
    using System.Linq;
    using Microsoft.Extensions.Configuration;

    public static class AzureInfo
    {
        private static IConfigurationRoot config;
        private const string settingsFileName = "azuresettings.json";
        private const string armEndpoints = "ArmEndpoints";
        private const string azureEnvironments = "AzureEnvironments";
        private const string azureAdInstances = "AzureAdInstances";

        static AzureInfo()
        {
            config = new ConfigurationBuilder()
               .SetBasePath(Directory.GetCurrentDirectory())
               .AddJsonFile(settingsFileName, optional: false)
               .Build();
        }

        public static IDictionary<string, string> GetArmEndpointsByAzureEnvironment()
        {
            return config.GetSection(armEndpoints).GetChildren().ToDictionary(x => x.Key, y => y.Value);
        }

        public static IEnumerable<string> GetAzureEnvironments()
        {
            return config.GetSection(azureEnvironments).Get<string[]>();
        }

        public static IDictionary<string, string> GetAzureADInstancesByAzureEnvironment()
        {
            return config.GetSection(azureAdInstances).GetChildren().ToDictionary(x => x.Key, y => y.Value);
        }

        public static string GetCosmosDbListKeyUrlFormat()
        {
            return config["CosmosDb:ListKeysUrlFormat"];
        }

        /// <summary>
        /// These environments were gotten from below azure powershell cli. When we 
        /// get instance metadata via IMDS, same environment values are returned
        /// 
        /// PS > Get-AzEnvironment
        /// Name              Resource Manager Url                  ActiveDirectory Authority
        /// ----              --------------------                  -------------------------
        /// AzureChinaCloud   https://management.chinacloudapi.cn/  https://login.chinacloudapi.cn/
        /// AzureCloud        https://management.azure.com/         https://login.microsoftonline.com/
        /// AzureGermanCloud  https://management.microsoftazure.de/ https://login.microsoftonline.de/
        /// AzureUSGovernment https://management.usgovcloudapi.net/ https://login.microsoftonline.us/
        /// PS >
        /// </summary>
        /// <returns></returns>
        public static string GetArmEndpoint(
            string azureEnvironment)
        {
            if (string.IsNullOrEmpty(azureEnvironment))
            {
                throw new ArgumentNullException("azureEnvironment is set as null or empty");
            }

            try
            {
                return config.GetSection(armEndpoints).GetSection(azureEnvironment).Value;
            }
            catch (Exception ex)
            {
                throw new InvalidAzureEnvironmentException($"Invalid environment {azureEnvironment}, exception: {ex}");
            }
        }

        /// <summary>
        /// GetAzureADInstance for a given AzureEnvironment
        /// </summary>
        /// <param name="azureEnvironment"></param>
        /// <returns></returns>
        public static string GetAzureADInstance(
            string azureEnvironment)
        {
            if (string.IsNullOrEmpty(azureEnvironment))
            {
                throw new ArgumentNullException("azureEnvironment is set as null or empty");
            }

            try
            {
                return config.GetSection(azureAdInstances).GetSection(azureEnvironment).Value;
            }
            catch (Exception ex)
            {
                throw new InvalidAzureEnvironmentException($"Invalid environment {azureEnvironment}, exception: {ex}");
            }
        }
    }
}
