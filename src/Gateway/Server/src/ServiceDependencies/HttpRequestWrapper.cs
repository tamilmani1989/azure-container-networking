// --------------------------------------------------------------------------------------------------------------------
// <copyright file="HttpRequestWrapper.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.Gateway
{
    using System.Linq;
    using System.Net.Http;
    using System.Threading.Tasks;
    using Microsoft.AspNetCore.Mvc;

    public class HttpRequestWrapper : IHttpRequestWrapper
    {
        public async Task<ContentResult> SendAsync(HttpClient httpClient, HttpRequestMessage request)
        {
            // Otherwise forward the request
            HttpResponseMessage response = await httpClient.SendAsync(request);
            string responseStream = await response.Content.ReadAsStringAsync();
            var contentType = response.Content.Headers.GetValues("Content-Type").FirstOrDefault();
            return new ContentResult { Content = responseStream, ContentType = contentType, StatusCode = (int)response.StatusCode };
        }
    }
}