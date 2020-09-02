// -----------------------------------------------------------------------------
// <copyright file="IImdsRetriever.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using System.Net.Http;
    using System.Threading.Tasks;

    public interface IImdsRetriever
    {
        /// <summary>
        /// Fetch IMDS Instance metadata
        /// </summary>
        /// <param name="httpClient"></param>
        /// <param name="imdsInstanceUri"></param>
        /// <returns></returns>
        Task<InstanceResponse> GetInstanceMetadata(string imdsInstanceUri);
    }
}
