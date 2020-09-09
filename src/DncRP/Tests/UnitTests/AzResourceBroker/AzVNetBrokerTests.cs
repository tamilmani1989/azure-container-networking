// -----------------------------------------------------------------------------
// <copyright file="ArmCertificateRetrieverTests.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP.Unittests
{
    using System;
    using System.Collections.Generic;
    using System.IO;
    using System.Linq;
    using System.Security.Cryptography.X509Certificates;
    using System.Threading.Tasks;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Configuration;
    using Microsoft.Extensions.Logging;
    using Microsoft.Extensions.Logging.Abstractions;
    using Xunit;

    /// <summary>
    /// If your tests make use of CurrentContext class, please
    /// decorate the class with [Collection("Sequential")]
    /// </summary>
    [Collection("Sequential")]
    public class AzVnetBrokerTests
    {
        private IConfiguration Configuration;
        private ILogger<AzVNetBroker> logger;
        private IHttpUtility testHttpUtility;

        public AzVnetBrokerTests()
        {
            this.Configuration = new ConfigurationBuilder()
               .SetBasePath(Directory.GetCurrentDirectory())
               .AddJsonFile("appsettings.json", optional: false)
               .Build();

            this.logger = new NullLogger<AzVNetBroker>();
            this.testHttpUtility = new TestHttpUtility();
        }

        /// <summary>
        /// Validate CreateOrUpdateServiceAssociationLink with fake the HTTP request and response.
        /// TODO: Create a TestHost which can validate the Request/Response
        /// </summary>
        [Fact]
        public async void Test_SetServiceAssociationLink()
        {
            CurrentContext.SetContext("dogfood", "testRegion");
            AzVNetBroker vnetBroker = new AzVNetBroker(testHttpUtility, logger);
            Guid vnetId = Guid.NewGuid();
            ServiceAssociationLink sal = await vnetBroker.CreateOrUpdateServiceAssociationLink(
                "useast2euap",
                "testSub",
                vnetId,
                "delegatedSubnetName",
                Guid.NewGuid().ToString(),
                Guid.NewGuid().ToString());

            Assert.NotNull(sal);
        }

        /// <summary>
        /// Validate CreateOrUpdateServiceAssociationLink with fake the HTTP request and response.
        /// TODO: Create a TestHost which will validate the HTTP Request/Response
        /// </summary>
        [Fact]
        public async void Test_SetResourceOwnership()
        {
            CurrentContext.SetContext("dogfood", "testRegion");
            AzVNetBroker vnetBroker = new AzVNetBroker(testHttpUtility, logger);
            Guid vnetId = Guid.NewGuid();
            ResourceOwnershipResponse resourceOwnershipResponse = await vnetBroker.SetResourceOwnershipForCustomerVnet(
                "useast2euap",
                "testSub",
                "testRG",
                "testVnet",
                VirtualNetworkOwnershipAction.Set);

            Assert.NotNull(resourceOwnershipResponse);
        }
    }
}
