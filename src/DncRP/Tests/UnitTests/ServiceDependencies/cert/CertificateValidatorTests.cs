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
    using Microsoft.Extensions.Options;
    using Moq;
    using Xunit;

    public class CertValidatorTestDependencies : IDisposable
    {
        public CertValidatorTestDependencies()
        {
            this.Configuration = new ConfigurationBuilder()
               .SetBasePath(Directory.GetCurrentDirectory())
               .AddJsonFile("appsettings.Development.json", optional: false)
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
    [Collection("Sequential")]
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

            CurrentContext.SetContext(azEnvName, this.dependencies.Configuration["Region"]);
            X509Certificate2 clientCert = Certificates.GenerateSelfSigned("unittest");
            //IImdsRetriever imdsRetriever = new TestImdsRetriever(azEnvName);
            TestHttpUtility testHttpUtility = new TestHttpUtility();
            testHttpUtility.SetCertificates(new List<X509Certificate2> { clientCert });
            IHttpUtility httpUtility = testHttpUtility;
            ILogger<ArmCertificateRetriever> retlogger = new NullLogger<ArmCertificateRetriever>();
            ILogger<CertificateValidator> certLogger = new NullLogger<CertificateValidator>();

            Mock<IOptionsMonitor<ArmCertificatesSettings>> optionsMonitorMock = new Mock<IOptionsMonitor<ArmCertificatesSettings>>();
            optionsMonitorMock.Setup(o => o.CurrentValue).Returns(new ArmCertificatesSettings { RefreshIntervalInSeconds = 3600 });

            Mock<IOptionsMonitor<CertificateValidatorSettings>> certValidOptions = new Mock<IOptionsMonitor<CertificateValidatorSettings>>();
            certValidOptions.Setup(o => o.CurrentValue).Returns(new CertificateValidatorSettings { Thumbprint = "" });

            ArmCertificateRetriever retriever = new ArmCertificateRetriever(httpUtility, retlogger, optionsMonitorMock.Object);
            ICertificateValidator certificateValidator = new CertificateValidator(retriever, certLogger, certValidOptions.Object);
            bool isValid = await certificateValidator.IsValid(clientCert);
            Assert.True(isValid);
        }

        [Fact]
        public async void Test_Client_Certificate_Validation_Fails()
        {
            string azEnvName = "dogfood";

            CurrentContext.SetContext(azEnvName, this.dependencies.Configuration["Region"]);
            X509Certificate2 clientCert = Certificates.GenerateSelfSigned("unittest");
            //IImdsRetriever imdsRetriever = new TestImdsRetriever(azEnvName);
            IHttpUtility httpUtility = new TestHttpUtility();
            ILogger<ArmCertificateRetriever> retlogger = new NullLogger<ArmCertificateRetriever>();
            ILogger<CertificateValidator> certLogger = new NullLogger<CertificateValidator>();

            Mock<IOptionsMonitor<ArmCertificatesSettings>> optionsMonitorMock = new Mock<IOptionsMonitor<ArmCertificatesSettings>>();
            optionsMonitorMock.Setup(o => o.CurrentValue).Returns(new ArmCertificatesSettings { RefreshIntervalInSeconds = 3600 });

            Mock<IOptionsMonitor<CertificateValidatorSettings>> certValidOptions = new Mock<IOptionsMonitor<CertificateValidatorSettings>>();
            certValidOptions.Setup(o => o.CurrentValue).Returns(new CertificateValidatorSettings { Thumbprint = "" });

            ArmCertificateRetriever retriever = new ArmCertificateRetriever(httpUtility, retlogger, optionsMonitorMock.Object);
            ICertificateValidator certificateValidator = new CertificateValidator(retriever, certLogger, certValidOptions.Object);
            bool isValid = await certificateValidator.IsValid(clientCert);
            Assert.False(isValid);
        }

        [Fact]
        public async void Test_NoCache_API_Is_Invoked_If_Cert_Validation_Fails()
        {
            string azEnvName = "dogfood";

            CurrentContext.SetContext(azEnvName, this.dependencies.Configuration["Region"]);
            X509Certificate2 clientCert = Certificates.GenerateSelfSigned("unittest");
            //IImdsRetriever imdsRetriever = new TestImdsRetriever(azEnvName);
            TestHttpUtility testHttpUtility = new TestHttpUtility();
            IHttpUtility httpUtility = testHttpUtility;
            ILogger<ArmCertificateRetriever> retlogger = new NullLogger<ArmCertificateRetriever>();
            ILogger<CertificateValidator> certLogger = new NullLogger<CertificateValidator>();

            Mock<IOptionsMonitor<ArmCertificatesSettings>> optionsMonitorMock = new Mock<IOptionsMonitor<ArmCertificatesSettings>>();
            optionsMonitorMock.Setup(o => o.CurrentValue).Returns(new ArmCertificatesSettings { RefreshIntervalInSeconds = 3600 });

            Mock<IOptionsMonitor<CertificateValidatorSettings>> certValidOptions = new Mock<IOptionsMonitor<CertificateValidatorSettings>>();
            certValidOptions.Setup(o => o.CurrentValue).Returns(new CertificateValidatorSettings { Thumbprint = "" });

            ArmCertificateRetriever retriever = new ArmCertificateRetriever(httpUtility, retlogger, optionsMonitorMock.Object);
            ICertificateValidator certificateValidator = new CertificateValidator(retriever, certLogger, certValidOptions.Object);
            bool isValid = await certificateValidator.IsValid(clientCert);
            Assert.False(isValid);

            testHttpUtility.SetCertificates(new List<X509Certificate2> { clientCert });
            isValid = await certificateValidator.IsValid(clientCert);
            Assert.True(isValid);
        }
    }
}
