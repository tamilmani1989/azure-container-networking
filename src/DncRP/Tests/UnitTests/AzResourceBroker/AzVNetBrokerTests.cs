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
    using System.Runtime.InteropServices;
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

        public AzVnetBrokerTests()
        {
            this.Configuration = new ConfigurationBuilder()
               .SetBasePath(Directory.GetCurrentDirectory())
               .AddJsonFile("appsettings.json", optional: false)
               .Build();

            this.logger = new NullLogger<AzVNetBroker>();

            CurrentContext.SetContext("dogfood", "testRegion");
            CurrentContext.SetArmAccessIdentity(new ClientIdentity());
            CurrentContext.SetManagedIdentity(new ClientIdentity());
        }

        /// <summary>
        /// Validate CreateOrUpdateServiceAssociationLink with fake the HTTP request and response.
        /// TODO: Create a TestHost which can validate the Request/Response
        /// </summary>
        [Fact]
        public async void Test_Set_Delete_ServiceAssociationLink()
        {
            Guid vnetId = Guid.NewGuid();
            const string subscriptionId = "subId";
            const string rgname = "rg";
            const string vnetName = "testVnet";
            const string subnet = "testSubnet";

            TestHttpUtility testHttpUtility = new TestHttpUtility();
            AzVNetBroker azVNetBroker = new AzVNetBroker(testHttpUtility, logger, new TestAzureAD());

            string vnetResourceId = AzVNetBroker.GetVNetResourceId(subscriptionId, rgname, vnetName);
            string subnetResourceId = $"{vnetResourceId.TrimEnd('/')}/subnets/{subnet}";
            ServiceAssociationLinkDetail sal = await azVNetBroker.GetOrCreateServiceAssociationLinkAsync(
                "useast2euap",
                vnetId.ToString(),
                subnetResourceId);

            Assert.NotNull(sal);

            // Now validate Get call itself returns the SAL detail
            testHttpUtility = new TestHttpUtility(false) ;
            azVNetBroker = new AzVNetBroker(testHttpUtility, logger, new TestAzureAD());

            sal = await azVNetBroker.GetOrCreateServiceAssociationLinkAsync(
                "useast2euap",
                vnetId.ToString(),
                subnetResourceId);

            Assert.NotNull(sal);

            await azVNetBroker.DeleteServiceAssociationLinkAsync(subscriptionId, subnetResourceId);
        }

        /// <summary>
        /// Validate CreateOrUpdateServiceAssociationLink with fake the HTTP request and response.
        /// TODO: Create a TestHost which will validate the HTTP Request/Response
        /// </summary>
        [Fact]
        public async void Test_SetResourceOwnership()
        {

            TestHttpUtility testHttpUtility = new TestHttpUtility();
            AzVNetBroker azVNetBroker = new AzVNetBroker(testHttpUtility, logger, new TestAzureAD()); 

            const string subscriptionId = "subId";
            const string rgname = "rg";
            const string vnetName = "testVnet";
            string vnetResourceId = AzVNetBroker.GetVNetResourceId(subscriptionId, rgname, vnetName);
            ResourceOwnershipResponse resourceOwnershipResponse = await azVNetBroker.SetResourceOwnershipForCustomerVnetAsync(
                "useast2euap",
                vnetResourceId);

            Assert.NotNull(resourceOwnershipResponse);
        }
    }
}
