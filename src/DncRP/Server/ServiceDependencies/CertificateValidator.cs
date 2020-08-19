// -----------------------------------------------------------------------------
// <copyright file="CertificateValidator.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Collections.Generic;
    using System.Security.Cryptography.X509Certificates;
    using System.Threading.Tasks;

    public class CertificateValidator : ICertificateValidator
    {
        private ICertificateRetriever certsRetriever;

        public CertificateValidator(ICertificateRetriever certificateRetriever)
        {
            this.certsRetriever = certificateRetriever;
        }

        /// <summary>
        /// Validate that certificate is valid and present in the allowed
        /// list of certificates (fetched using certificate retriever).
        /// For now, only thumbprint is matched
        /// </summary>
        /// <param name="certificate"></param>
        /// <returns></returns>
        public async Task<bool> IsValid(X509Certificate2 certificate)
        {
            IEnumerable<X509Certificate2> allowedCerts = await this.certsRetriever.GetCertificates();
            if (VerifyCertificateMetadata(certificate, allowedCerts))
            {
                return true;
            }

            // If verification fails, lets fetch certificates one more time and this time
            // force it to not use cached values. This is our attempt to make sure that our cache
            // is not stale. Cache is refreshed every hour.
            allowedCerts = await this.certsRetriever.ForceGetCertificates();
            return VerifyCertificateMetadata(certificate, allowedCerts);
        }

        /// <summary>
        /// Verify metadata of certificate
        /// </summary>
        /// <param name="callerCert"></param>
        /// <param name="allowedCerts"></param>
        /// <returns></returns>
        private bool VerifyCertificateMetadata(X509Certificate2 callerCert, IEnumerable<X509Certificate2> allowedCerts)
        {
            foreach (X509Certificate2 cert in allowedCerts)
            {
                if (callerCert.Thumbprint == cert.Thumbprint)
                {
                    return true;
                }
            }

            return false;
        }
    }
}
