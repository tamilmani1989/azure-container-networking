// -----------------------------------------------------------------------------
// <copyright file="TestConfig.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP.ServiceTests
{
    internal class TestClientIdentityCertificate
    {
        public string AppId
        {
            get; set;
        }

        public string TenantId
        {
            get; set;
        }

        public string ClientCertificateName
        {
            get; set;
        }

        public string AzureEnvironment
        {
            get; set;
        }
    }

    internal class TestConfig
    {
        public string SubscriptionId
        {
            get; set;
        }

        public string ResourceGroup
        {
            get; set;
        }

        public string DbAccountName
        {
            get; set;
        }

        public string KeyVaultConnectionString
        {
            get; set;
        }

        public string KeyVaultUrl
        {
            get; set;
        }
        public string AzureEnvironment
        {
            get; set;
        }

        public string AzureRegion
        {
            get; set;
        }

        public TestClientIdentityCertificate ClientIdentityCertificate
        {
            get; set;
        }

        public TestClientIdentityCertificate DelegateSubnetClientCertificate
        {
            get; set;
        }
    }
}