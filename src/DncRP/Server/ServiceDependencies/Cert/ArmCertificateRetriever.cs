// -----------------------------------------------------------------------------
// <copyright file="CertificateRetriever.cs" company="Microsoft">
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
    using Microsoft.Extensions.Configuration;
    using Microsoft.Extensions.Logging;

    /// <summary>
    /// TODO: Add an API in RP to trigger cert retrieval for any livesite scenario
    /// TODO: Add API to fetch certs in use and last retrieval time
    /// TODO: Add logging to this class
    /// </summary>
    public class ArmCertificateRetriever : ICertificateRetriever
    {
        private const int defaultRefreshTimeoutInSeconds = 3600;

        private IConfiguration configuration;
        private IImdsRetriever imdsRetriever;
        private IHttpUtility httpUtility;

        private readonly TimeSpan timeout;
        private readonly List<X509Certificate2> certificateList;
        private readonly HttpClient httpClient;
        private readonly SemaphoreSlim semaphoreSlim;
        private DateTime lastCertUpdateTime;
        private string azureEnvironment;
        private ILogger<ArmCertificateRetriever> logger;

        /// <summary>
        /// Create an instance of CertificateRetriever
        /// </summary>
        /// <param name="configuration">Configuration from appsettings.json</param>
        /// <param name="azureEnvName">Optional argument. If not specified,
        /// environment will be learned by calling to IMDS endpoint in host node.
        /// Passing in environment has values in two scenarios.
        /// - For testing in devbox, environment can be passed in since there is no IMDS
        /// - For any other scenario where RP doesn't run in container and is hosted on baremetal
        /// </param>
        public ArmCertificateRetriever(
            IConfiguration configuration,
            IImdsRetriever imdsRetriever,
            IHttpUtility httpUtility,
            ILogger<ArmCertificateRetriever> logger,
            string azureEnvName = null,
            int refreshCertPeriodInSeconds = 3600)
        {
            this.configuration = configuration;
            this.imdsRetriever = imdsRetriever;
            this.httpUtility = httpUtility;
            this.logger = logger;

            if (!string.IsNullOrWhiteSpace(azureEnvName))
            {
                this.azureEnvironment = azureEnvName.ToLower();
            }

            if (refreshCertPeriodInSeconds <= 0)
            {
                refreshCertPeriodInSeconds = defaultRefreshTimeoutInSeconds;
            }

            this.timeout = TimeSpan.FromSeconds(refreshCertPeriodInSeconds);
            this.httpClient = new HttpClient();
            this.lastCertUpdateTime = DateTime.MinValue;
            this.certificateList = new List<X509Certificate2>();
            this.semaphoreSlim = new SemaphoreSlim(1, 1);
        }

        /// <summary>
        /// Retrieve certificate metadata from ARM endpoint
        /// </summary>
        /// <returns></returns>
        public async Task<IEnumerable<X509Certificate2>> GetCertificates()
        {
            return await this.GetCertificatesInternal(false);
        }

        /// <summary>
        /// Get certificates and fetch them directly from ARM metadata endpoint.
        /// Do not use cached values
        /// </summary>
        /// <returns></returns>
        public async Task<IEnumerable<X509Certificate2>> ForceGetCertificates()
        {
            return await this.GetCertificatesInternal(true);
        }

        private async Task<IEnumerable<X509Certificate2>> GetCertificatesInternal(bool doNotUseCache)
        {
            if (this.ShouldFetchCertificates(doNotUseCache))
            {
                await semaphoreSlim.WaitAsync();

                try
                {
                    if (string.IsNullOrWhiteSpace(this.azureEnvironment))
                    {
                        InstanceResponse response = await this.imdsRetriever.GetInstanceMetadata(
                            this.httpClient, configuration["Imds:InstanceEndpoint"]);
                        this.azureEnvironment = response.compute.azEnvironment.ToLower();
                    }

                    if (this.ShouldFetchCertificates(doNotUseCache))
                    {
                        // Retrieve certificates and cache in certificate list
                        string endpoint = this.GetCertMetadataEndpoint();
                        ArmCertMetadata metadata = await this.httpUtility.GetJsonAsync<ArmCertMetadata>(
                            this.httpClient,
                            HttpCompletionOption.ResponseHeadersRead, endpoint);

                        List<X509Certificate2> certs = new List<X509Certificate2>();
                        foreach (ClientCertificate c in metadata.clientCertificates)
                        {
                            var clientCert = new X509Certificate2(Convert.FromBase64String(c.certificate));
                            certs.Add(clientCert);
                        }

                        this.certificateList.Clear();
                        this.certificateList.AddRange(certs);

                        // update last retrieval time
                        this.lastCertUpdateTime = DateTime.UtcNow;
                    }
                }
                finally
                {
                    semaphoreSlim.Release();
                }
            }

            return this.certificateList;
        }

        /// <summary>
        /// These environments were gotten from below azure powershell cli. When we 
        /// get instance metadata via IMDS, same environment values are returned
        /// 
        /// PS > Get-AzEnvironment
        /// Name Resource Manager Url                  ActiveDirectory Authority
        /// ----              --------------------                  -------------------------
        /// AzureChinaCloud   https://management.chinacloudapi.cn/  https://login.chinacloudapi.cn/
        /// AzureCloud        https://management.azure.com/         https://login.microsoftonline.com/
        /// AzureGermanCloud  https://management.microsoftazure.de/ https://login.microsoftonline.de/
        /// AzureUSGovernment https://management.usgovcloudapi.net/ https://login.microsoftonline.us/
        /// PS >
        /// </summary>
        /// <returns></returns>
        private string GetCertMetadataEndpoint()
        {
            switch (this.azureEnvironment)
            {
                case "azurechinacloud": return this.configuration["CertAuthEndPoints:Mooncake"];
                case "azurecloud": return this.configuration["CertAuthEndPoints:Public"];
                case "azuregermancloud": return this.configuration["CertAuthEndPoints:BlackForest"];
                case "azureusgovernment": return this.configuration["CertAuthEndPoints:Fairfax"];
                case "dogfood": return this.configuration["CertAuthEndPoints:DogFood"];
                default:
                    throw new InvalidAzureEnvironmentException($"Invalid environment {this.azureEnvironment}");
            }
        }

        /// <summary>
        /// If cached certificates are older than 60 mins, then fetch it from ARM management URL.
        /// https://armwiki.azurewebsites.net/authorization/AuthenticateBetweenARMandRP.html
        /// </summary>
        /// <returns></returns>
        private bool ShouldFetchCertificates(bool doNotUseCache)
        {
            if (doNotUseCache)
            {
                return true;
            }

            return ((DateTime.UtcNow - lastCertUpdateTime) > timeout);
        }
    }
}
