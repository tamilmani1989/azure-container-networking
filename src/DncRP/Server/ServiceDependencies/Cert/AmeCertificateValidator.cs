// -----------------------------------------------------------------------------
// <copyright file="AmeCertificateValidator.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Collections.Generic;
    using System.Net.Http;
    using System.Net.Security;
    using System.Runtime.CompilerServices;
    using System.Security.Cryptography.X509Certificates;
    using Microsoft.Azure.ContainerNetworking.CoreLib;

    public static class AmeCertificateValidator
    {
        private static string allowedSubjectNames;
        private static bool skipAmeCertValidation;
        private static IHttpUtility ameHttpUtility;

        private static bool VerifyIssuer(X509Certificate2 serverCert, IEnumerable<X509Certificate2> rootCaCerts)
        {
            string serverIssuerName = serverCert.GetNameInfo(X509NameType.SimpleName, true);
            foreach (X509Certificate2 cert in rootCaCerts)
            {
                string simpleName = cert.GetNameInfo(X509NameType.SimpleName, false);
                string dnsName = cert.GetNameInfo(X509NameType.DnsName, false);
                if (serverIssuerName.Equals(dnsName, StringComparison.OrdinalIgnoreCase) ||
                    serverIssuerName.Equals(simpleName, StringComparison.OrdinalIgnoreCase))
                {
                    return true;
                }
            }

            CustomLog.WriteToConsole("Issuer {0} couldn't be verified.", serverIssuerName);
            CustomLog.WriteToConsole("Expected Issuers:");
            foreach(X509Certificate2 cert in rootCaCerts)
            {
                CustomLog.WriteToConsole("- {0}", cert.GetNameInfo(X509NameType.SimpleName, true));
            }

            return false;
        }

        private static bool VerifySubject(X509Certificate2 callerCert, string subjectName)
        {
            string dnsName = callerCert.GetNameInfo(X509NameType.DnsName, false);

            if (dnsName.Equals(subjectName, StringComparison.OrdinalIgnoreCase))
            {
                return true;
            }

            CustomLog.WriteToConsole("SubjectName {0} couldn't be verified. Expected:{1}", dnsName, subjectName);
            return false;
        }

        private static bool LogIfCertValidationFailed(bool isValidCert, X509Certificate2 cert)
        {
            if (!isValidCert)
            {
                string message = string.Format("Cert validation failed. Thumbprint:{0}, Subject:{1}, Issuer:{2}",
                    cert.Thumbprint, cert.Subject, cert.Issuer);

                CustomLog.WriteToConsole(message);
            }

            CustomLog.WriteToConsole("AME cert CA validation succeeded");
            return isValidCert;
        }

        public static void IntializeAmeCertificateValidator(AmeCertValidationSettings validationSettings, IHttpUtility httpUtility)
        {
            allowedSubjectNames = validationSettings.AmeAllowedSubjectNames;
            skipAmeCertValidation = validationSettings.SkipAmeCertValidation;
            ameHttpUtility = httpUtility;
        }

        public static bool ValidateAmeRootCA(HttpRequestMessage request, X509Certificate2 certificate, X509Chain certificateChain, SslPolicyErrors policy)
        {
            CustomLog.WriteToConsole("ValidateRootCA callback invoked");

            if(skipAmeCertValidation)
            {
                CustomLog.WriteToConsole("Skipping Ame Cert Validation");
                return true;
            }

            AmeCertificateRetriever certRetriever = new AmeCertificateRetriever(ameHttpUtility);

            IEnumerable<X509Certificate2> rootCaCerts = certRetriever.GetCertificates().GetAwaiter().GetResult(); ;

            if (VerifyIssuer(certificate, rootCaCerts) && VerifySubject(certificate, allowedSubjectNames))
            {
                CustomLog.WriteToConsole("AME cert CA validation succeeded");
                return true;
            }

            CustomLog.WriteToConsole("Force get ame root ca");

            // If verification fails, lets fetch certificates one more time and this time
            // force it to not use cached values. This is our attempt to make sure that our cache
            // is not stale. Cache is refreshed every hour.
            IEnumerable<X509Certificate2> rootAmeCerts = certRetriever.ForceGetCertificates().GetAwaiter().GetResult();
            bool isValid = VerifyIssuer(certificate, rootAmeCerts) && VerifySubject(certificate, allowedSubjectNames);
            return LogIfCertValidationFailed(isValid, certificate);
        }
    }
}
