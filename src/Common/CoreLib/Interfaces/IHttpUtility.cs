// -----------------------------------------------------------------------------
// <copyright file="IHttpUtility.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using System;
    using System.Net.Http;
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
        /// </summary>
        /// <typeparam name="T"></typeparam>
        /// <param name="requestMessage"></param>
        /// <param name="option"></param>
        /// <param name="token"></param>
        /// <returns></returns>
        Task<T> SendRequestAndGetJsonAsync<T>(
            HttpRequestMessage requestMessage, HttpCompletionOption option, CancellationToken token);

        /// <summary>
        /// Invoke REST API to send the message and deserialized response in caller specified object.
        /// </summary>
        /// <typeparam name="TRequest">TRequest type, this will be serialized</typeparam>
        /// <typeparam name="TResponse">TResponse type, this will be deserialized from response</typeparam>
        /// <param name="httpClient">http client</param>
        /// <param name="httpMethod">type of HTTPMethod (put/post)</param>
        /// <param name="requestUri">Request URI</param>
        /// <param name="requestBody">Request body of type TRequest</param>
        /// <param name="accessToken">Authorization token</param>
        /// <returns></returns>
        Task<TResponse> SendRequestAndGetJsonAsync<TRequest, TResponse>(
            HttpMethod httpMethod,
            Uri requestUri,
            TRequest requestBody,
            string accessToken) where TResponse : class where TRequest : class;
    }
}
