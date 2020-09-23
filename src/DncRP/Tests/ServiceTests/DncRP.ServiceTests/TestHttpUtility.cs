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
    using System.Linq;
    using System.Net.Http;
    using System.Security.Cryptography.X509Certificates;
    using System.Security.Permissions;
    using System.Text;
    using System.Threading;
    using System.Threading.Tasks;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Azure.ContainerNetworking.DncRP;
    using Newtonsoft.Json;
    using NetworkModel = Microsoft.Azure.Management.Network.Models;

#pragma warning disable CS1998 // Async method lacks 'await' operators and will run synchronously
    internal class TestHttpUtility : IHttpUtility
    {
        private bool validateCreateSal { get; set; }
        private string PrimaryContextRequestId { get; set; }
        private List<X509Certificate2> callerSpecifiedCerts;
        private bool validateMetadataHeader = false;

        public TestHttpUtility(bool validateCreateSal = true)
        {
            this.callerSpecifiedCerts = new List<X509Certificate2>();
            this.validateCreateSal = validateCreateSal;
            PrimaryContextRequestId = String.Empty;
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
        public async Task<T> SendRequestAndGetJsonAsync<T>(HttpRequestMessage requestMessage)
        {
            return default(T);
        }

        /// <summary>
        /// Invoke Http request and deserialize its JSON result to caller specified object
        /// Deserialize it using the given settings
        /// </summary>
        /// <typeparam name="T"></typeparam>
        /// <param name="requestMessage"></param>
        /// <param name="deserializeSettings"></param>
        /// <returns></returns>
        public async Task<T> SendRequestAndGetJsonAsync<T>(HttpRequestMessage requestMessage, JsonSerializerSettings deserializeSettings)
        {
            if (typeof(T) == typeof(NetworkModel.VirtualNetwork))
            {
                // split RequestURI
                object virtualNetworkResponse = new NetworkModel.VirtualNetwork(
                    $"/subscriptions /testSub/resourceGroups/testRg/providers/Microsoft.Network/virtualNetworks/testVnet",
                    name: "testName",
                    location: CurrentContext.AzureRegion,
                    resourceGuid: Guid.NewGuid().ToString(),
                    provisioningState: NetworkModel.ProvisioningState.Succeeded.ToString());

                return (T)virtualNetworkResponse;
            }
            if (typeof(T) == typeof(ServiceAssociationLink))
            {
                string json = await requestMessage.Content.ReadAsStringAsync();
                object request = JsonConvert.DeserializeObject<ServiceAssociationLink>(json);
                ServiceAssociationLink sal = request as ServiceAssociationLink;
                PrimaryContextRequestId = sal.Properties.Details[0].Properties.PrimaryContextRequestId;
                return (T)request;
            }

            if (typeof(T) == typeof(ResourceOwnershipResponse))
            {
                object serviceResourceOwnership = new ResourceOwnershipResponse();
                return (T)serviceResourceOwnership;
            }

            return default(T);
        }

        /// <summary>
        /// Invoke Http request and deserialize its JSON result to caller specified object
        /// </summary>
        /// <typeparam name="T"></typeparam>
        /// <param name="requestMessage"></param>
        /// <param name="option"></param>
        /// <param name="token"></param>
        /// <returns></returns>
        public async Task<T> SendRequestAndGetJsonAsync<T>(
            HttpRequestMessage requestMessage, HttpCompletionOption option, CancellationToken token)
        {
            if (this.validateMetadataHeader)
            {
                if (!requestMessage.Headers.Contains("Metadata"))
                {
                    throw new Exception("No metadata header found in the request");
                }

                IEnumerable<string> values = requestMessage.Headers.GetValues("Metadata");
                if (!values.Contains("true", StringComparer.OrdinalIgnoreCase))
                {
                    throw new Exception("Metadata header value not set to true");
                }
            }

            if (typeof(T) == typeof(InstanceResponse))
            {
                object instance = new InstanceResponse { Compute = null };
                return (T)instance;
            }

            throw new NotSupportedException($"Type {typeof(T).Name} is not supported");
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

        public void SetValidateMetadataHeaderFlag(bool value)
        {
            this.validateMetadataHeader = value;
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

        public async Task<HttpResponseMessage> SendAsync(HttpRequestMessage message)
        {
            HttpResponseMessage httpResponseMessage = new HttpResponseMessage();
            if (message.RequestUri.ToString().Contains("ServiceAssociationLinks/ManagedDNC/Details/ManagedDNCDetail"))
            {
                // Returns the actualy SAL when Sal is created

                if (validateCreateSal)
                {
                    if (string.IsNullOrEmpty(PrimaryContextRequestId))
                    {
                        httpResponseMessage.StatusCode = System.Net.HttpStatusCode.NotFound;
                    }
                    else
                    {
                        ServiceAssociationLinkDetail salDetail = new ServiceAssociationLinkDetail();
                        salDetail.Id = Guid.NewGuid().ToString();
                        salDetail.Name = "ManagedDNCDetail";
                        salDetail.Properties = new ServiceAssociationLinkDetailProperties();
                        salDetail.Properties.PrimaryContextId = Guid.NewGuid().ToString();
                        salDetail.Properties.PrimaryContextRequestId = PrimaryContextRequestId;
                        salDetail.Properties.SecondaryContextId = Guid.NewGuid().ToString();
                        salDetail.Properties.SecondaryContextRequestId = Guid.NewGuid().ToString();

                        string json = JsonUtility.Serialize(salDetail);

                        httpResponseMessage.Content = new StringContent(json, Encoding.UTF8, "application/json");
                        httpResponseMessage.StatusCode = System.Net.HttpStatusCode.OK;
                    }
                }
                else
                {
                    // This is scneario where SAL is already created and Get is returning the token info
                    ServiceAssociationLinkDetail salDetail = new ServiceAssociationLinkDetail();
                    salDetail.Id = Guid.NewGuid().ToString();
                    salDetail.Name = "ManagedDNCDetail";
                    salDetail.Properties = new ServiceAssociationLinkDetailProperties();
                    salDetail.Properties.PrimaryContextId = Guid.NewGuid().ToString();
                    salDetail.Properties.PrimaryContextRequestId = PrimaryContextRequestId;
                    salDetail.Properties.SecondaryContextId = Guid.NewGuid().ToString();
                    salDetail.Properties.SecondaryContextRequestId = Guid.NewGuid().ToString();

                    string json = JsonUtility.Serialize(salDetail);

                    httpResponseMessage.Content = new StringContent(json, Encoding.UTF8, "application/json");
                    httpResponseMessage.StatusCode = System.Net.HttpStatusCode.OK;
                }

                return httpResponseMessage;
            }

            if (message.RequestUri.ToString().Contains("https://management.azure.com/subscriptions/"))
            {
                HttpUtility httpUtility = new HttpUtility(new HttpClient());
                return await httpUtility.SendAsync(message);
            };

            return httpResponseMessage;
        }
    }
}
