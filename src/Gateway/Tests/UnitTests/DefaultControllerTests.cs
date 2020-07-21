// --------------------------------------------------------------------------------------------------------------------
// <copyright file="DefaultControllerTests.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.Gateway.UnitTests
{
    using System;
    using System.Collections.Generic;
    using System.IO;
    using System.Net.Http;
    using Microsoft.AspNetCore.Http;
    using Microsoft.AspNetCore.Http.Extensions;
    using Microsoft.AspNetCore.Mvc;
    using Microsoft.Extensions.Logging;
    using Microsoft.Extensions.Primitives;
    using Moq;
    using Xunit;

    public class ControllerDependencies : IDisposable
    {
        public ControllerDependencies()
        {
            Environment.SetEnvironmentVariable("ServiceIpAddress", "50.12.122.143");

            this.ServicesPortsCollection = ServicePorts.Load(Path.Combine(Directory.GetCurrentDirectory(), "ServicePorts.UnitTests.json"));

            Mock<ILogger<DefaultController>> logger = new Mock<ILogger<DefaultController>>();
            this.Logger = logger.Object;

            this.HttpClientsDict = new Dictionary<string, HttpClient>(StringComparer.OrdinalIgnoreCase);
            IDictionary<string, ServicePort> servicePorts = this.ServicesPortsCollection.GetServicePortsMapping();
            foreach (KeyValuePair<string, ServicePort> kvp in servicePorts)
            {
                this.HttpClientsDict.Add(
                    kvp.Key,
                    new HttpClient
                    {
                        BaseAddress = new Uri(string.Format("{0}://{1}:{2}", kvp.Value.Scheme, kvp.Value.GetHostName(), kvp.Value.Port))
                    });
            }
        }

        public void Dispose()
        {
            // No clean up for now
        }

        public ILogger<DefaultController> Logger
        {
            get; private set;
        }

        public IDictionary<string, HttpClient> HttpClientsDict
        {
            get; private set;
        }

        public ServicePorts ServicesPortsCollection
        {
            get; private set;
        }
    }

    public class DefaultControllerTests : IClassFixture<ControllerDependencies>
    {
        private ILogger<DefaultController> logger;
        private IDictionary<string, HttpClient> httpClientsDict;
        private IDictionary<string, ServicePort> servicesPortsDict;
        private ServicePorts servicesPorts;
        private string gatewayPort = "5001";

        public DefaultControllerTests(ControllerDependencies dependencyFixture)
        {
            this.logger = dependencyFixture.Logger;
            this.httpClientsDict = dependencyFixture.HttpClientsDict;
            this.servicesPorts = dependencyFixture.ServicesPortsCollection;
            this.servicesPortsDict = dependencyFixture.ServicesPortsCollection.GetServicePortsMapping();
        }

        /// <summary>
        /// Main function of DefaultController in gateway is to forward requests to services running in the backend.
        /// For example - When a request comes to gateway, it authenticates and authorizes it and then it figures
        /// to which service this request should be forwarded. This test validates the forwarding logic.
        /// </summary>
        [Fact]
        public void Validate_Forwarding_Uri()
        {
            foreach (KeyValuePair<string, ServicePort> kvp in this.servicesPortsDict)
            {
                Mock<IHttpClientFactory> clientFactory = new Mock<IHttpClientFactory>();
                clientFactory.Setup(x => x.CreateClient(It.IsAny<string>())).Returns(this.httpClientsDict[kvp.Key]);

                DefaultController controller = new DefaultController(clientFactory.Object, this.servicesPorts, this.logger, new TestHttpRequestWrapper());
                this.SetHttpContext(controller, kvp.Key);

                string inputUriStr = controller.ControllerContext.HttpContext.Request.GetEncodedUrl();
                Uri inputUri = new Uri(inputUriStr);

                UriBuilder ub = new UriBuilder(kvp.Value.Scheme, kvp.Value.GetHostName(), kvp.Value.Port, inputUri.LocalPath.Replace($"/{kvp.Key}", ""), inputUri.Query);
                Uri expectedUri = ub.Uri;

                // Invoke controller
                ContentResult result = controller.HandleRequest().GetAwaiter().GetResult();
                Uri resultUri = new Uri(result.Content);

                int compareResult = Uri.Compare(
                    expectedUri,
                    resultUri,
                    UriComponents.HostAndPort | UriComponents.PathAndQuery | UriComponents.Scheme,
                    UriFormat.SafeUnescaped, StringComparison.OrdinalIgnoreCase);

                Assert.Equal(0, compareResult);
            }
        }

        /// <summary>
        /// Validate ServicePorts.json in Gateway project
        /// </summary>
        [Fact]
        public void Validate_ServicePorts_Json_In_Gateway()
        {
            ServicePorts sp = ServicePorts.Load(Path.Combine(Directory.GetCurrentDirectory(), "ServicePorts.json"));
            sp.GetServicePortsMapping();

            // if there is an error in loading json file and getting service ports mapping, this will throw
        }

        /// <summary>
        /// Test that invalid service ports json will actually throw exception
        /// </summary>
        [Fact]
        public void Invalid_ServicePorts_Json_Throws()
        {
            ServicePorts sp = ServicePorts.Load(Path.Combine(Directory.GetCurrentDirectory(), "ServicePorts.Invalid.Host.json"));
            Assert.Throws<InvalidHostException>(() => sp.GetServicePortsMapping());

            sp = ServicePorts.Load(Path.Combine(Directory.GetCurrentDirectory(), "ServicePorts.Invalid.Scheme.json"));
            Assert.Throws<InvalidSchemeException>(() => sp.GetServicePortsMapping());

            sp = ServicePorts.Load(Path.Combine(Directory.GetCurrentDirectory(), "ServicePorts.Invalid.ServiceName.json"));
            Assert.Throws<InvalidServiceNameException>(() => sp.GetServicePortsMapping());

            sp = ServicePorts.Load(Path.Combine(Directory.GetCurrentDirectory(), "ServicePorts.Invalid.ServicePort.json"));
            Assert.Throws<InvalidServicePortException>(() => sp.GetServicePortsMapping());
        }

        private void SetHttpContext(DefaultController controller, string serviceName)
        {
            controller.ControllerContext = new ControllerContext();
            controller.ControllerContext.HttpContext = new DefaultHttpContext();
            controller.ControllerContext.HttpContext.Request.Host = new HostString($"localhost:{gatewayPort}");
            controller.ControllerContext.HttpContext.Request.Path = new PathString(string.Format("/{0}/test1/test2", serviceName));
            controller.ControllerContext.HttpContext.Request.Method = "GET";
            controller.ControllerContext.HttpContext.Request.PathBase = "";
            controller.ControllerContext.HttpContext.Request.Scheme = "https";

            Dictionary<string, StringValues> dict = new Dictionary<string, StringValues>();
            dict.Add("a", new StringValues("b"));
            dict.Add("c", new StringValues("d"));
            controller.ControllerContext.HttpContext.Request.Query = new QueryCollection(dict);
        }
    }
}
