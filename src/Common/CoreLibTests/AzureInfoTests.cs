// -----------------------------------------------------------------------------
// <copyright file="AzureInfoTests.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib.Unittests
{
    using System;
    using System.Collections.Generic;
    using System.Linq;
    using Xunit;

    public class AzureInfoTests
    {
        [Fact]
        public void TestAzureEnvironments()
        {
            IEnumerable<string> envs = AzureInfo.GetAzureEnvironments();
            Assert.True(envs.Count() == 5);
        }

        [Fact]
        public void TestArmEndpointsByAzureEnvironment()
        {
            IEnumerable<string> envs = AzureInfo.GetAzureEnvironments();

            IDictionary<string, string> dict = AzureInfo.GetArmEndpointsByAzureEnvironment();
            foreach (string e in envs)
            {
                Assert.True(dict.ContainsKey(e));
            }
        }

        [Fact]
        public void TestAzureADInstancesByAzureEnvironment()
        {
            IEnumerable<string> envs = AzureInfo.GetAzureEnvironments();

            IDictionary<string, string> dict = AzureInfo.GetAzureADInstancesByAzureEnvironment();
            foreach (string e in envs)
            {
                Assert.True(dict.ContainsKey(e));
            }
        }

        [Fact]
        public void TestGetArmEndpoint()
        {
            IEnumerable<string> envs = AzureInfo.GetAzureEnvironments();

            foreach (string e in envs)
            {
                string actualEndpoint = AzureInfo.GetArmEndpoint(e);
                string expectedEndpoint = "";
                switch (e)
                {
                    case "azurecloud":
                        expectedEndpoint = "https://management.azure.com";
                        break;
                    case "azurechinacloud":
                        expectedEndpoint = "https://management.chinacloudapi.cn";
                        break;
                    case "azureusgovernment":
                        expectedEndpoint = "https://management.usgovcloudapi.net";
                        break;
                    case "azuregermancloud":
                        expectedEndpoint = "https://management.microsoftazure.de";
                        break;
                    case "dogfood":
                        expectedEndpoint = "https://api-dogfood-admin.resources.windows-int.net";
                        break;
                    default:
                        throw new InvalidOperationException($"Invalid environment {e}");
                }

                Assert.True(string.Equals(actualEndpoint, expectedEndpoint));
            }
        }


        [Fact]
        public void TestGetAadInstance()
        {
            IEnumerable<string> envs = AzureInfo.GetAzureEnvironments();

            foreach (string e in envs)
            {
                string actualAadInstance = AzureInfo.GetAzureADInstance(e);
                string expectedlAadInstance = "";
                switch (e)
                {
                    case "azurecloud":
                        expectedlAadInstance = "https://login.microsoftonline.com";
                        break;
                    case "azurechinacloud":
                        expectedlAadInstance = "https://login.chinacloudapi.cn";
                        break;
                    case "azureusgovernment":
                        expectedlAadInstance = "https://login.microsoftonline.us";
                        break;
                    case "azuregermancloud":
                        expectedlAadInstance = "https://login.microsoftonline.de";
                        break;
                    case "dogfood":
                        expectedlAadInstance = "";
                        break;
                    default:
                        throw new InvalidOperationException($"Invalid environment {e}");
                }

                Assert.True(string.Equals(actualAadInstance, expectedlAadInstance));
            }
        }

        [Fact]
        public void TestGetAadInstanceWithTrailingSlash()
        {
            IEnumerable<string> envs = AzureInfo.GetAzureEnvironments();

            foreach (string e in envs)
            {
                string actualAadInstance = AzureInfo.GetAzureADInstance(e, true);
                string expectedlAadInstance = "";
                switch (e)
                {
                    case "azurecloud":
                        expectedlAadInstance = "https://login.microsoftonline.com/";
                        break;
                    case "azurechinacloud":
                        expectedlAadInstance = "https://login.chinacloudapi.cn/";
                        break;
                    case "azureusgovernment":
                        expectedlAadInstance = "https://login.microsoftonline.us/";
                        break;
                    case "azuregermancloud":
                        expectedlAadInstance = "https://login.microsoftonline.de/";
                        break;
                    case "dogfood":
                        expectedlAadInstance = "";
                        break;
                    default:
                        throw new InvalidOperationException($"Invalid environment {e}");
                }

                Assert.True(string.Equals(actualAadInstance, expectedlAadInstance));
            }
        }
    }
}
