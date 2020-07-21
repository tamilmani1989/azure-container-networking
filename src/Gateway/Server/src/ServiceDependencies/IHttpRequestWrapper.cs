// --------------------------------------------------------------------------------------------------------------------
// <copyright file="IHttpRequestWrapper.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.Gateway
{
    using System.Net.Http;
    using System.Threading.Tasks;
    using Microsoft.AspNetCore.Mvc;

    public interface IHttpRequestWrapper
    {
        Task<ContentResult> SendAsync(HttpClient httpClient, HttpRequestMessage request);
    }
}
