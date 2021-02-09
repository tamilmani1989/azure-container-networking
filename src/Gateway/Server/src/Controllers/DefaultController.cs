// --------------------------------------------------------------------------------------------------------------------
// <copyright file="DefaultController.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.Gateway
{
    using System;
    using System.Collections.Generic;
    using System.Net.Http;
    using System.Threading.Tasks;

    using Microsoft.AspNetCore.Mvc;
    using Microsoft.AspNetCore.Mvc.WebApiCompatShim;
    using Microsoft.AspNetCore.Routing;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Logging;

    /// <summary>
    /// Controller to match all requests that come to dnc gateway
    /// </summary>
    [ApiController]
    [Route("/{**routePath}")]
    public class DefaultController : ControllerBase
    {
        private readonly IHttpClientFactory clientFactory;

        private readonly IDictionary<string, ServicePort> serviceNamesDictionary;

        private readonly ILogger<DefaultController> logger;

        private readonly IHttpRequestWrapper httpRequestWrapper;

        public DefaultController(
            IHttpClientFactory clientFactory,
            IServicePorts servicePorts,
            ILogger<DefaultController> logger,
            IHttpRequestWrapper httpRequestWrapper)
        {
            this.clientFactory = clientFactory;
            serviceNamesDictionary = servicePorts.GetServicePortsMapping();
            this.logger = logger;
            this.httpRequestWrapper = httpRequestWrapper;
        }

        [Route("/{**routePath}")]
        public async Task<ContentResult> HandleRequest()
        {
            string serviceName = GetServiceNameFromRoute(HttpContext.Request.Path);

            using (logger.BeginScope("Accessing service: {0}", serviceName))
            {
                using (HttpClient client = clientFactory.CreateClient(serviceName))
                {
                    logger.CLogInformation("Incoming request. Host: {0}, Path: {1}, https: {2} method: {3}",
                        HttpContext.Request.Host, HttpContext.Request.Path, HttpContext.Request.IsHttps, HttpContext.Request.Method);

                    HttpRequestMessageFeature hreqmf = new HttpRequestMessageFeature(HttpContext);
                    HttpRequestMessage httpRequestMessage = hreqmf.HttpRequestMessage;

                    string path = HttpContext.Request.Path.Value.Replace(string.Format("/{0}", serviceName), "");
                    UriBuilder ub = new UriBuilder(new Uri(client.BaseAddress, path));
                    ub.Query = hreqmf.HttpRequestMessage.RequestUri.Query;

                    httpRequestMessage.RequestUri = ub.Uri;

                    logger.CLogInformation("Outgoing request. Uri: {0}", httpRequestMessage.RequestUri.AbsoluteUri);

                    ContentResult contentResult = await this.httpRequestWrapper.SendAsync(client, httpRequestMessage);
                    return contentResult;
                }
            }
        }

        /// <summary>
        /// Get service name from route. In case of dnc, service name won't be in
        /// relative url. However for other services, will need to have it to differentiate between services        
        /// </summary>
        /// <param name="routePath"></param>
        /// <returns></returns>
        private string GetServiceNameFromRoute(string routePath)
        {
            const string defaultService = "dnc";

            if (string.IsNullOrEmpty(routePath))
            {
                return defaultService;
            }

            string[] parts = routePath.Split("/", StringSplitOptions.RemoveEmptyEntries);
            if (parts == null || parts.Length == 0)
            {
                return defaultService;
            }

            string sName = parts[0];
            return serviceNamesDictionary.ContainsKey(sName) ? sName : defaultService;
        }
    }
}
