namespace Microsoft.Azure.ContainerNetworking.DncRP.Unittests
{
    using System.Collections.Generic;
    using System.Net.Http;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Logging;
    using Microsoft.Extensions.Logging.Abstractions;
    using Xunit;
    using System.Security.Cryptography.X509Certificates;
    using System.Linq;

    public class AmeRetrieverTests
    {
        private ILogger<AmeCertificateRetriever> logger;
        private IHttpUtility httpUtility;
        private const string expectedSubNamePrefix = "CN=ameroot";

        public AmeRetrieverTests()
        {
            this.logger = new NullLogger<AmeCertificateRetriever>();
            this.httpUtility = new HttpUtility(new HttpClient());
        }

        [Fact]
        public async void GetAmeCertTest()
        {
            AmeCertificateRetriever certRetriever = new AmeCertificateRetriever(
                this.httpUtility);
            IEnumerable<X509Certificate2> certs = await certRetriever.GetCertificates();

            Assert.True(certs.Count() != 0);
            foreach (X509Certificate2 c in certs)
            {
                Assert.StartsWith(expectedSubNamePrefix, c.Issuer);
            }
        }
    }
}
