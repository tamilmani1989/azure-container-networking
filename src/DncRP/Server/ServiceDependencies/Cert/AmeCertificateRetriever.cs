// -----------------------------------------------------------------------------
// <copyright file="AmeCertificateRetriever.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Collections.Generic;
    using System.Net.Http;
    using System.Security.Cryptography.X509Certificates;
    using System.Threading;
    using System.Threading.Tasks;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Logging;

    public class AmeCertificateRetriever : ICertificateRetriever
    {
        private IHttpUtility httpUtility;
        private readonly List<X509Certificate2> certificateList;
        private readonly SemaphoreSlim semaphoreSlim;
        private const string dSMSUri = "https://ussouth-dsms.dsms.core.windows.net/dsms/issuercertificates?getissuersv3&caName=ame";

        /// <summary>
        /// Create an instance of CertificateRetriever
        /// </summary>
        public AmeCertificateRetriever(
            IHttpUtility httpUtility)
        {
            this.httpUtility = httpUtility;
            this.certificateList = new List<X509Certificate2>();
            this.semaphoreSlim = new SemaphoreSlim(1, 1);
        }

        /// <summary>
        /// Retrieve certificate dSMS endpoint
        /// </summary>
        /// <returns></returns>
        public async Task<IEnumerable<X509Certificate2>> GetCertificates()
        {
            return await this.GetCertificatesInternal(false);
        }

        /// <summary>
        /// Get certificates and fetch them directly from public dSMS endpoint.
        /// Do not use cached values
        /// </summary>
        /// <returns></returns>
        public async Task<IEnumerable<X509Certificate2>> ForceGetCertificates()
        {
            return await this.GetCertificatesInternal(true);
        }

        private async Task<IEnumerable<X509Certificate2>> GetCertificatesInternal(bool doNotUseCache)
        {
            if (doNotUseCache || this.certificateList.Count == 0)
            {
                await semaphoreSlim.WaitAsync();

                try
                {
                    AmeCertMetadata metadata = await this.httpUtility.GetJsonAsync<AmeCertMetadata>(
                        HttpCompletionOption.ResponseHeadersRead, dSMSUri);
                       
                    List<X509Certificate2> certs = new List<X509Certificate2>();
                    foreach (RootsInfos root in metadata.RootsInfos)
                    {
                        if (root.CaName == "AME")
                        {
                            var serverCert = new X509Certificate2(Convert.FromBase64String(root.Body));
                            certs.Add(serverCert);
                        }
                        foreach(Intermediates im in root.Intermediates)
                        {
                            var serverCert = new X509Certificate2(Convert.FromBase64String(im.Body));
                            certs.Add(serverCert);
                        }
                    }
                    
                    this.certificateList.Clear();
                    this.certificateList.AddRange(certs);
                }
                catch(Exception ex)
                {
                    CustomLog.WriteToConsole("Exception caught while retreiving ame CA: {0}", ex);
                }
                finally
                {
                    semaphoreSlim.Release();
                }
            }

            return this.certificateList;
        }
    }
}
