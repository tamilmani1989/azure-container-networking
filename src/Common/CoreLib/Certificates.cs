// -----------------------------------------------------------------------------
// <copyright file="Certificates.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using System;
    using System.Security.Cryptography;
    using System.Security.Cryptography.X509Certificates;
    using Microsoft.Azure.KeyVault;
    using Microsoft.Azure.KeyVault.Models;
    using Microsoft.Azure.Services.AppAuthentication;

    public class Certificates
    {
        /// <summary>
        /// Fetch SSL/TLS certificate from key vault (details of which are configured in appsettings.json)
        /// </summary>
        /// <param name="configuration">config object containing appsettings.json</param>
        /// <returns>Certificate object</returns>
        public static X509Certificate2 GetKeyVaultCertificate(string keyVaultUrl, string certificateName)
        {
            return FetchCertificateFromKV(keyVaultUrl, certificateName);
        }

        /// <summary>
        /// Generate a self signed certificate for dnc gateway
        /// </summary>
        /// <returns></returns>
        public static X509Certificate2 GenerateSelfSigned(string subjectName, string san = null)
        {
            SubjectAlternativeNameBuilder sanBuilder = new SubjectAlternativeNameBuilder();
            if (!string.IsNullOrWhiteSpace(san))
            {
                sanBuilder.AddDnsName(san);
            }

            X500DistinguishedName distinguishedName = new X500DistinguishedName($"CN={subjectName}");

            using (RSA rsa = RSA.Create(2048))
            {
                var request = new CertificateRequest(distinguishedName, rsa, HashAlgorithmName.SHA256, RSASignaturePadding.Pkcs1);

                request.CertificateExtensions.Add(
                    new X509KeyUsageExtension(
                        X509KeyUsageFlags.DataEncipherment | X509KeyUsageFlags.KeyEncipherment | X509KeyUsageFlags.DigitalSignature, false));


                request.CertificateExtensions.Add(
                   new X509EnhancedKeyUsageExtension(
                       new OidCollection { new Oid("1.3.6.1.5.5.7.3.1") }, false));

                request.CertificateExtensions.Add(sanBuilder.Build());

                var certificate = request.CreateSelfSigned(new DateTimeOffset(DateTime.UtcNow.AddDays(-1)), new DateTimeOffset(DateTime.UtcNow.AddDays(365)));

                X509Certificate2 cert = new X509Certificate2(certificate.Export(X509ContentType.Pfx));

                return cert;
            }
        }

        /// <summary>
        /// Fetch certificate from a key vault
        /// </summary>
        /// <param name="keyVaultUrl"></param>
        /// <param name="certificateName"></param>
        /// <returns></returns>
        private static X509Certificate2 FetchCertificateFromKV(string keyVaultUrl, string certificateName)
        {
            return Retry.WithRetryCount(3, TimeSpan.FromSeconds(10), () =>
            {
                // Instantiate a new KeyVaultClient object, with an access token to Key Vault
                // https://docs.microsoft.com/en-us/azure/key-vault/general/service-to-service-authentication#connection-string-support
                AzureServiceTokenProvider azureServiceTokenProvider = new AzureServiceTokenProvider();
                KeyVaultClient client = new KeyVaultClient(new KeyVaultClient.AuthenticationCallback(azureServiceTokenProvider.KeyVaultTokenCallback));

                SecretBundle secret = client.GetSecretAsync(keyVaultUrl, certificateName).GetAwaiter().GetResult();

                byte[] certBytes = Convert.FromBase64String(secret.Value);
                X509Certificate2 sslCert = new X509Certificate2(certBytes);

                return sslCert;
            },
            null);
        }
    }
}
