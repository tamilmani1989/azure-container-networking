// -----------------------------------------------------------------------------
// <copyright file="CertificateValidator.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Collections.Generic;
    using System.Security.Cryptography.X509Certificates;
    using System.Threading.Tasks;
    using Microsoft.Extensions.Logging;

    public class CertificateValidator : ICertificateValidator
    {
        private ICertificateRetriever certsRetriever;
        private string allowedCustomThumbprint;
        private ILogger<CertificateValidator> logger;

        public CertificateValidator(
            ICertificateRetriever certificateRetriever,
            ILogger<CertificateValidator> logger,
            string thumbprint)
        {
            this.certsRetriever = certificateRetriever;
            this.allowedCustomThumbprint = thumbprint;
            this.logger = logger;
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
            bool isValid;

            if (!string.IsNullOrWhiteSpace(this.allowedCustomThumbprint))
            {
                isValid = certificate.Thumbprint.Equals(this.allowedCustomThumbprint, StringComparison.OrdinalIgnoreCase);
                if (isValid)
                {
                    return true;
                }
            }

            IEnumerable<X509Certificate2> allowedCerts = await this.certsRetriever.GetCertificates();
            if (VerifyCertificateMetadata(certificate, allowedCerts))
            {
                return true;
            }

            // If verification fails, lets fetch certificates one more time and this time
            // force it to not use cached values. This is our attempt to make sure that our cache
            // is not stale. Cache is refreshed every hour.
            allowedCerts = await this.certsRetriever.ForceGetCertificates();
            isValid = VerifyCertificateMetadata(certificate, allowedCerts);
            return this.LogIfCertValidationFailed(isValid, certificate);
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

        /// <summary>
        /// Log cert metadata if cert validation fails.
        /// </summary>
        /// <param name="isValidCert"></param>
        /// <param name="cert"></param>
        /// <returns>
        /// return if validation failed or succeeded. This value is actually passed in by caller.
        /// It is returned back to the caller
        /// </returns>
        private bool LogIfCertValidationFailed(bool isValidCert, X509Certificate2 cert)
        {
            if (!isValidCert)
            {
                this.logger.LogInformation(
                    "Cert validation failed. Thumbprint:{0}, Subject:{1}, Issuer:{2}", cert.Thumbprint, cert.Subject, cert.Issuer);
            }

            return isValidCert;
        }
    }
}