// -----------------------------------------------------------------------------
// <copyright file="IHttpUtility.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using System.Net.Http;
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
    }
}
