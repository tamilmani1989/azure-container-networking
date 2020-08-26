// -----------------------------------------------------------------------------
// <copyright file="CertValidatorTestDependencies.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP.Unittests
{
    using System;
    using System.Collections.Generic;
    using System.IO;
    using System.Security.Cryptography.X509Certificates;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Configuration;
    using Microsoft.Extensions.Logging;
    using Microsoft.Extensions.Logging.Abstractions;
    using Xunit;

    public class CertValidatorTestDependencies : IDisposable
    {
        public CertValidatorTestDependencies()
        {
            this.Configuration = new ConfigurationBuilder()
               .SetBasePath(Directory.GetCurrentDirectory())
               .AddJsonFile("appsettings.json", optional: false)
               .Build();
        }

        public void Dispose()
        {
            // No clean up for now
        }

        public IConfiguration Configuration
        {
            get; private set;
        }
    }

    /// <summary>
    /// Certificate validator tests. 
    /// </summary>
    public class CertificateValidatorTests : IClassFixture<CertValidatorTestDependencies>
    {
        private CertValidatorTestDependencies dependencies;

        public CertificateValidatorTests(CertValidatorTestDependencies testDependencies)
        {
            this.dependencies = testDependencies;
        }

        [Fact]
        public async void Test_Client_Certificate_Validation_Is_Successful()
        {
            string azEnvName = "dogfood";

            X509Certificate2 clientCert = Certificates.GenerateSelfSigned("unittest");
            IImdsRetriever imdsRetriever = new TestImdsRetriever(azEnvName);
            TestHttpUtility testHttpUtility = new TestHttpUtility();
            testHttpUtility.SetCertificates(new List<X509Certificate2> { clientCert });
            IHttpUtility httpUtility = testHttpUtility;
            ILogger<ArmCertificateRetriever> retlogger = new NullLogger<ArmCertificateRetriever>();
            ILogger<CertificateValidator> certLogger = new NullLogger<CertificateValidator>();

            ArmCertificateRetriever retriever = new ArmCertificateRetriever(this.dependencies.Configuration, imdsRetriever, httpUtility, retlogger);
            ICertificateValidator certificateValidator = new CertificateValidator(retriever, certLogger, null);
            bool isValid = await certificateValidator.IsValid(clientCert);
            Assert.True(isValid);
        }

        [Fact]
        public async void Test_Client_Certificate_Validation_Fails()
        {
            string azEnvName = "dogfood";

            X509Certificate2 clientCert = Certificates.GenerateSelfSigned("unittest");
            IImdsRetriever imdsRetriever = new TestImdsRetriever(azEnvName);
            IHttpUtility httpUtility = new TestHttpUtility();
            ILogger<ArmCertificateRetriever> retlogger = new NullLogger<ArmCertificateRetriever>();
            ILogger<CertificateValidator> certLogger = new NullLogger<CertificateValidator>();

            ArmCertificateRetriever retriever = new ArmCertificateRetriever(this.dependencies.Configuration, imdsRetriever, httpUtility, retlogger);
            ICertificateValidator certificateValidator = new CertificateValidator(retriever, certLogger, null);
            bool isValid = await certificateValidator.IsValid(clientCert);
            Assert.False(isValid);
        }

        [Fact]
        public async void Test_NoCache_API_Is_Invoked_If_Cert_Validation_Fails()
        {
            string azEnvName = "dogfood";

            X509Certificate2 clientCert = Certificates.GenerateSelfSigned("unittest");
            IImdsRetriever imdsRetriever = new TestImdsRetriever(azEnvName);
            TestHttpUtility testHttpUtility = new TestHttpUtility();
            IHttpUtility httpUtility = testHttpUtility;
            ILogger<ArmCertificateRetriever> retlogger = new NullLogger<ArmCertificateRetriever>();
            ILogger<CertificateValidator> certLogger = new NullLogger<CertificateValidator>();

            ArmCertificateRetriever retriever = new ArmCertificateRetriever(this.dependencies.Configuration, imdsRetriever, httpUtility, retlogger);
            ICertificateValidator certificateValidator = new CertificateValidator(retriever, certLogger, null);
            bool isValid = await certificateValidator.IsValid(clientCert);
            Assert.False(isValid);

            testHttpUtility.SetCertificates(new List<X509Certificate2> { clientCert });
            isValid = await certificateValidator.IsValid(clientCert);
            Assert.True(isValid);
        }
    }
}
