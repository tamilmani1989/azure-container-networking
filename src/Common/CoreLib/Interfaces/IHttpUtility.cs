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
        /// Invoke GET on the URI and deserialize the JSON resonse 
        /// into user specified type (T)
        /// </summary>
        /// <typeparam name="T"></typeparam>
        /// <param name="httpClient"></param>
        /// <param name="option"></param>
        /// <param name="uri"></param>
        /// <returns></returns>
        Task<T> GetJsonAsync<T>(HttpClient httpClient, HttpCompletionOption option, string uri);
    }
}
