// -----------------------------------------------------------------------------
// <copyright file="HttpUtility.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using Newtonsoft.Json;
    using System;
    using System.IO;
    using System.Net.Http;
    using System.Net.Http.Headers;
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
        /// Deserialize it using the given settings
        /// </summary>
        /// <typeparam name="T"></typeparam>
        /// <param name="requestMessage"></param>
        /// <param name="deserializeSettings"></param>
        /// <returns></returns>
        public async Task<T> SendRequestAndGetJsonAsync<T>(HttpRequestMessage requestMessage, JsonSerializerSettings deserializeSettings)
        {
            HttpResponseMessage httpResponse = await this.httpClient.SendAsync(requestMessage);
            return await this.ParseHttpResponse<T>(httpResponse, deserializeSettings);
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

        private async Task<T> ParseHttpResponse<T>(HttpResponseMessage httpResponse, JsonSerializerSettings deserializeSettings = null)
        {
            try
            {
                httpResponse.EnsureSuccessStatusCode(); // throws if not 200-299

                if (httpResponse.Content is object && httpResponse.Content.Headers.ContentType.MediaType == "application/json")
                {
                    var contentStream = await httpResponse.Content.ReadAsStringAsync();
                    if (deserializeSettings == null)
                    {
                        // use the default settings
                        return JsonUtility.Deserialize<T>(contentStream);
                    }
                    else
                    {
                        return JsonUtility.Deserialize<T>(contentStream, deserializeSettings);
                    }
                }
            }
            catch (Exception ex)
            {
                throw new Exception($"ResponseMessage: {httpResponse.Content}, status code {httpResponse.StatusCode}, " +
                    $"Reason: {httpResponse.ReasonPhrase}, " +
                    $"Request: {httpResponse.RequestMessage}, " +
                    $"internal exception: {ex}");
            }

            throw new InvalidDataException($"Invalid content type '{httpResponse.Content.Headers.ContentType.MediaType}' returned. ");
        }

        /// <summary>
        /// Send the specified Http request and retun the HttResponseMessage
        /// </summary>
        /// <typeparam name="T"></typeparam>
        /// <param name="requestMessage"></param>
        /// <returns></returns>
        public async Task<HttpResponseMessage> SendAsync(HttpRequestMessage requestMessage)
        {
            return await httpClient.SendAsync(requestMessage);
        }
    }
}
