// -----------------------------------------------------------------------------
// <copyright file="IHttpUtility.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using Newtonsoft.Json;
    using System;
    using System.Net.Http;
    using System.Net.Http.Headers;
    using System.Threading;
    using System.Threading.Tasks;

    public interface IHttpUtility
    {
        /// <summary>
        /// Invoke GET on the URI and deserialize the JSON response
        /// into user specified type (T)
        /// </summary>
        /// <typeparam name="T"></typeparam>
        /// <param name="option"></param>
        /// <param name="uri"></param>
        /// <returns></returns>
        Task<T> GetJsonAsync<T>(HttpCompletionOption option, string uri);

        /// <summary>
        /// Send the specified Http request and deserialize the JSON response
        /// into user specified type (T)
        /// </summary>
        /// <typeparam name="T"></typeparam>
        /// <param name="requestMessage"></param>
        /// <returns></returns>
        Task<T> SendRequestAndGetJsonAsync<T>(HttpRequestMessage requestMessage);

        /// <summary>
        /// Invoke Http request and deserialize its JSON result to caller specified object
        /// Deserialize it using the given settings
        /// </summary>
        /// <typeparam name="T"></typeparam>
        /// <param name="requestMessage"></param>
        /// <param name="deserializeSettings"></param>
        /// <returns></returns>
        Task<T> SendRequestAndGetJsonAsync<T>(HttpRequestMessage requestMessage, JsonSerializerSettings deserializeSettings);

        /// <summary>
        /// Invoke Http request and deserialize its JSON result to caller specified object
        /// </summary>
        /// <typeparam name="T"></typeparam>
        /// <param name="requestMessage"></param>
        /// <param name="option"></param>
        /// <param name="token"></param>
        /// <returns></returns>
        Task<T> SendRequestAndGetJsonAsync<T>(
            HttpRequestMessage requestMessage, HttpCompletionOption option, CancellationToken token);

        /// <summary>
        /// Send the specified Http request and retun the HttResponseMessage
        /// </summary>
        /// <typeparam name="T"></typeparam>
        /// <param name="requestMessage"></param>
        /// <returns></returns>
        Task<HttpResponseMessage> SendAsync(HttpRequestMessage requestMessage);

        /// <summary>
        /// Invoke Http request and deserialize its JSON result to caller specified object.
        /// It also return the response headers
        /// </summary>
        /// <typeparam name="T"></typeparam>
        /// <param name="requestMessage"></param>
        /// <param name="option"></param>
        /// <param name="token"></param>
        /// <returns></returns>
        Task<HttpResponseMessage> SendRequestAndGetHttpResponseAsync(
            HttpRequestMessage requestMessage, HttpCompletionOption option, CancellationToken token);

        /// <summary>
        /// Parse httpresponse message and deserialize its JSON result to caller specified object
        /// </summary>
        /// <typeparam name="T"></typeparam>
        /// <param name="httpResponse"></param>
        /// <param name="deserializeSettings"></param>
        /// <returns></returns>
        Task<T> ParseHttpResponse<T>(HttpResponseMessage httpResponse, JsonSerializerSettings deserializeSettings = null);
    }
}
