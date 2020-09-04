// -----------------------------------------------------------------------------
// <copyright file="TestHttpUtility.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP.Unittests
{
    using System;
    using System.Collections.Generic;
    using System.Net.Http;
    using System.Security.Cryptography.X509Certificates;
    using System.Text;
    using System.Threading.Tasks;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Azure.ContainerNetworking.DncRP;

#pragma warning disable CS1998 // Async method lacks 'await' operators and will run synchronously
    internal class TestHttpUtility : IHttpUtility
    {
        private List<X509Certificate2> callerSpecifiedCerts;

        public TestHttpUtility()
        {
            this.callerSpecifiedCerts = new List<X509Certificate2>();
        }

        public async Task<T> GetJsonAsync<T>(HttpCompletionOption option, string uri)
        {
            if (typeof(T) == typeof(ArmCertMetadata))
            {
                object metadata = await GetArmCertMetadata(uri);
                return (T)metadata;
            }

            throw new NotSupportedException($"Type {typeof(T).Name} is not supported");
        }

        /// <summary>
        /// Send the specified Http request and deserialize the JSON response
        /// into user specified type (T)
        /// </summary>
        /// <typeparam name="T"></typeparam>
        /// <param name="requestMessage"></param>
        /// <returns></returns>
        public Task<T> SendRequestAndGetJsonAsync<T>(HttpRequestMessage requestMessage)
        {
            return default(Task<T>);
        }

        public async Task<TResponse> SendRequestAndGetJsonAsync<TRequest, TResponse>(
            HttpMethod httpMethod,
            Uri requestUri,
            TRequest requestBody,
            string accessToken) where TResponse : class where TRequest : class
        {
            if (typeof(TRequest) == typeof(ServiceAssociationLink))
            {
                object serviceAssociationLink = requestBody as ServiceAssociationLink;
                return (TResponse)serviceAssociationLink;
            }

            if (typeof(TRequest) == typeof(ResourceOwnershipRequest))
            {
                object serviceResourceOwnership = new ResourceOwnershipResponse();
                return (TResponse)serviceResourceOwnership;
            }

            throw new NotSupportedException($"Type {typeof(TRequest).Name} is not supported");
	}

        private async Task<ArmCertMetadata> GetArmCertMetadata(string uri)
        {
            string certSubName = this.GetCertSubNameForEnvironment(uri);
            X509Certificate2 cert1 = Certificates.GenerateSelfSigned($"{certSubName}.mgmt.azure.com", $"*.{certSubName}.mgmt.azure.com");
            X509Certificate2 cert2 = Certificates.GenerateSelfSigned($"{certSubName}1.mgmt.azure.com", $"*.{certSubName}1.mgmt.azure.com");

            string base64Cert1 = Convert.ToBase64String(cert1.RawData, Base64FormattingOptions.InsertLineBreaks);
            string base64Cert2 = Convert.ToBase64String(cert2.RawData, Base64FormattingOptions.InsertLineBreaks);


            ArmCertMetadata armCertMetadata = new ArmCertMetadata();
            List<ClientCertificate> clientCertificates = new List<ClientCertificate>();

            clientCertificates.Add(new ClientCertificate
            {
                certificate = base64Cert1,
                notBefore = DateTime.UtcNow.AddDays(-1),
                notAfter = DateTime.UtcNow.AddDays(1),
                thumbprint = cert1.Thumbprint
            });

            clientCertificates.Add(new ClientCertificate
            {
                certificate = base64Cert2,
                notBefore = DateTime.UtcNow.AddDays(-1),
                notAfter = DateTime.UtcNow.AddDays(1),
                thumbprint = cert2.Thumbprint
            });

            foreach (X509Certificate2 c in this.callerSpecifiedCerts)
            {
                clientCertificates.Add(new ClientCertificate
                {
                    certificate = Convert.ToBase64String(c.RawData, Base64FormattingOptions.InsertLineBreaks),
                    notBefore = c.NotBefore,
                    notAfter = c.NotAfter,
                    thumbprint = c.Thumbprint
                });
            }

            armCertMetadata.clientCertificates = clientCertificates.ToArray();

            return armCertMetadata;
        }

        /// <summary>
        /// Set user specified certs in cache. These will be returned
        /// along with generated certs.
        /// </summary>
        /// <param name="certificates"></param>
        public void SetCertificates(IEnumerable<X509Certificate2> certificates)
        {
            this.callerSpecifiedCerts.Clear();
            this.callerSpecifiedCerts.AddRange(certificates);
        }

        private string GetCertSubNameForEnvironment(string uri)
        {
            switch (uri)
            {
                case "https://management.azure.com:24582/metadata/authentication?api-version=2015-01-01": return "azurecloud";
                case "https://api-dogfood-admin.resources.windows-int.net/metadata/authentication?api-version=2015-01-01": return "dogfood";
                case "https://management.chinacloudapi.cn:24582/metadata/authentication?api-version=2015-01-01": return "azurechinacloud";
                case "https://management.usgovcloudapi.net:24582/metadata/authentication?api-version=2015-01-01": return "azureusgovernment";
                case "https://management.microsoftazure.de:24582/metadata/authentication?api-version=2015-01-01": return "azuregermancloud";
                default:
                    throw new InvalidAzureEnvironmentException($"Invalid uri: {uri}");
            }
        }
    }
}
