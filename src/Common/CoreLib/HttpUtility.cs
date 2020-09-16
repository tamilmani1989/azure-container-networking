// -----------------------------------------------------------------------------
// <copyright file="HttpUtility.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using System;
    using System.IO;
    using System.Net.Http;
    using System.Net.Http.Headers;
    using System.Text;
    using System.Threading;
    using System.Threading.Tasks;

    public class HttpUtility : IHttpUtility
    {
        private HttpClient httpClient;

        public HttpUtility(HttpClient httpClient)
        {
            this.httpClient = httpClient;
        }

        /// <summary>
        /// The header client request identifier.
        /// </summary>
        public const string HeaderClientRequestId = "x-ms-client-request-id";

        /// <summary>
        /// Invoke REST API and deserialize its JSON result to caller specified object
        /// </summary>
        /// <typeparam name="T"></typeparam>
        /// <param name="httpClient"></param>
        /// <param name="option"></param>
        /// <param name="uri"></param>
        /// <returns></returns>
        public async Task<T> GetJsonAsync<T>(HttpCompletionOption option, string uri)
        {
            HttpResponseMessage httpResponse = await this.httpClient.GetAsync(uri, option);
            return await this.ParseHttpResponse<T>(httpResponse);
        }

        /// <summary>
        /// Invoke Http request and deserialize its JSON result to caller specified object
        /// </summary>
        /// <typeparam name="T"></typeparam>
        /// <param name="requestMessage"></param>
        /// <returns></returns>
        public async Task<T> SendRequestAndGetJsonAsync<T>(HttpRequestMessage requestMessage)
        {
            HttpResponseMessage httpResponse = await this.httpClient.SendAsync(requestMessage);
            return await this.ParseHttpResponse<T>(httpResponse);
        }

        /// <summary>
        /// Invoke Http request and deserialize its JSON result to caller specified object
        /// </summary>
        /// <typeparam name="T"></typeparam>
        /// <param name="requestMessage"></param>
        /// <param name="option"></param>
        /// <param name="token"></param>
        /// <returns></returns>
        public async Task<T> SendRequestAndGetJsonAsync<T>(
            HttpRequestMessage requestMessage, HttpCompletionOption option, CancellationToken token)
        {
            HttpResponseMessage httpResponse = await this.httpClient.SendAsync(requestMessage, option, token);
            return await this.ParseHttpResponse<T>(httpResponse);
        }

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
        public async Task<TResponse> SendRequestAndGetJsonAsync<TRequest, TResponse>(
            HttpMethod httpMethod,
            Uri requestUri,
            TRequest requestBody,
            string accessToken) where TResponse : class where TRequest : class
        {
            using (var request = new HttpRequestMessage(httpMethod, requestUri))
            {
                if (!string.IsNullOrWhiteSpace(accessToken))
                {
                    request.Headers.Authorization = new AuthenticationHeaderValue(scheme: "Bearer", parameter: accessToken);
                }

                request.Headers.Add(name: HeaderClientRequestId, value: Guid.NewGuid().ToString());
                request.Headers.Add(name: "User-Agent", value: "cloud-console-rp/1.0");

                if (requestBody != null)
                {
                    request.Content = new StringContent(JsonUtility.Serialize(requestBody), Encoding.UTF8, "application/json");
                }

                var response = await httpClient.SendAsync(request);

                return await ParseHttpResponse<TResponse>(response);
            }
        }

        private async Task<T> ParseHttpResponse<T>(HttpResponseMessage httpResponse)
        {
            httpResponse.EnsureSuccessStatusCode(); // throws if not 200-299

            if (httpResponse.Content is object && httpResponse.Content.Headers.ContentType.MediaType == "application/json")
            {
                var contentStream = await httpResponse.Content.ReadAsStreamAsync();

                return JsonUtility.Deserialize<T>(contentStream);
            }

            throw new InvalidDataException($"Invalid content type '{httpResponse.Content.Headers.ContentType.MediaType}' returned. ");
        }
    }
}
