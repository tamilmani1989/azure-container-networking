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
            Assert.True(envs.Count() == 4);
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
    }
}
