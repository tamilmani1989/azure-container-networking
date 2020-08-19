// -----------------------------------------------------------------------------
// <copyright file="HttpUtility.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using System.IO;
    using System.Net.Http;
    using System.Text.Json;
    using System.Threading.Tasks;

    public class HttpUtility : IHttpUtility
    {
        /// <summary>
        /// Invoke REST API and deserialize its JSON result to caller specified object
        /// </summary>
        /// <typeparam name="T"></typeparam>
        /// <param name="httpClient"></param>
        /// <param name="option"></param>
        /// <param name="uri"></param>
        /// <returns></returns>
        public async Task<T> GetJsonAsync<T>(HttpClient httpClient,HttpCompletionOption option,  string uri)
        {
            HttpResponseMessage httpResponse = await httpClient.GetAsync(uri, option);

            httpResponse.EnsureSuccessStatusCode(); // throws if not 200-299

            if (httpResponse.Content is object && httpResponse.Content.Headers.ContentType.MediaType == "application/json")
            {
                var contentStream = await httpResponse.Content.ReadAsStreamAsync();

                return await JsonSerializer.DeserializeAsync<T>(
                    contentStream,
                    new JsonSerializerOptions { IgnoreNullValues = true, PropertyNameCaseInsensitive = true });
            }

            throw new InvalidDataException($"Invalid content type '{httpResponse.Content.Headers.ContentType.MediaType}' returned. ");
        }    
    }
}
