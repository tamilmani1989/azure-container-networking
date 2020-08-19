// -----------------------------------------------------------------------------
// <copyright file="ImdsRetriever.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using System.Net.Http;
    using System.Threading.Tasks;

    /// <summary>
    /// Class to invoke IMDS REST API and deserialize its results
    /// </summary>
    public class ImdsRetriever : IImdsRetriever
    {
        private IHttpUtility httpUtility;

        public ImdsRetriever(IHttpUtility httpUtility)
        {
            this.httpUtility = httpUtility;
        }

        /// <summary>
        /// Fetch IMDS Instance metadata
        /// http://169.254.169.254/metadata/instance?api-version=2019-08-15&format=json
        /// </summary>
        /// <param name="httpClient"></param>
        /// <param name="imdsInstanceUri"></param>
        /// <returns></returns>
        public async Task<InstanceResponse> GetInstanceMetadata(HttpClient httpClient, string imdsInstanceUri)
        {
            return await this.httpUtility.GetJsonAsync<InstanceResponse>(httpClient, HttpCompletionOption.ResponseHeadersRead, imdsInstanceUri);
        }
    }
}
