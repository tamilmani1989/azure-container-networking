// -----------------------------------------------------------------------------
// <copyright file="ICertificateRetriever.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Collections.Generic;
    using System.Security.Cryptography.X509Certificates;
    using System.Threading.Tasks;

    /// <summary>
    /// Certificate retrieval interface. If implementation does not
    /// maintain cache internally, both APIs will be same in that case.
    /// </summary>
    public interface ICertificateRetriever
    {
        /// <summary>
        /// Get certificates. If cache is valid and not expired,
        /// certificates will be returned from cache
        /// </summary>
        /// <returns></returns>
        Task<IEnumerable<X509Certificate2>> GetCertificates();

        /// <summary>
        /// Get certificates and do not use cached values
        /// </summary>
        /// <returns></returns>
        Task<IEnumerable<X509Certificate2>> ForceGetCertificates();
    }
}
