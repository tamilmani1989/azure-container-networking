// --------------------------------------------------------------------------------------------------------------------
// <copyright file="TestHttpRequestWrapper.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.Gateway
{
    using System.Linq;
    using System.Net;
    using System.Net.Http;
    using System.Threading.Tasks;
    using Microsoft.AspNetCore.Mvc;

    public class TestHttpRequestWrapper : IHttpRequestWrapper
    {
#pragma warning disable CS1998 // Async method lacks 'await' operators and will run synchronously
        public async Task<ContentResult> SendAsync(HttpClient httpClient, HttpRequestMessage request)
#pragma warning restore CS1998 // Async method lacks 'await' operators and will run synchronously
        {
            // Return the outgoing URI
            return new ContentResult { Content = request.RequestUri.AbsoluteUri, ContentType = "text/html", StatusCode = (int)HttpStatusCode.OK };
        }
    }
}