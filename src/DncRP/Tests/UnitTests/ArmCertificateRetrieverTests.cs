// -----------------------------------------------------------------------------
// <copyright file="ArmCertificateRetrieverTests.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP.Unittests
{
    using System;
    using System.Collections.Generic;
    using System.IO;
    using System.Linq;
    using System.Security.Cryptography.X509Certificates;
    using System.Threading.Tasks;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Configuration;
    using Xunit;

    public class CertRetrieverTestDependencies : IDisposable
    {
        public CertRetrieverTestDependencies()
        {
            this.Configuration = new ConfigurationBuilder()
               .SetBasePath(Directory.GetCurrentDirectory())
               .AddJsonFile("appsettings.json", optional: false)
               .Build();
        }

        public void Dispose()
        {
            // No clean up for now
        }

        public IConfiguration Configuration
        {
            get; private set;
        }
    }

    public class ArmCertificateRetrieverTests : IClassFixture<CertRetrieverTestDependencies>
    {
        private CertRetrieverTestDependencies dependencies;

        public ArmCertificateRetrieverTests(CertRetrieverTestDependencies testDependencies)
        {
            this.dependencies = testDependencies;
        }

        [Fact]
        public async void Test_Certificate_Retrieval_Is_Success()
        {
            string azEnvName = "dogfood";
            string expectedSubNamePrefix = $"CN={azEnvName}";

            IImdsRetriever imdsRetriever = new TestImdsRetriever(azEnvName);
            IHttpUtility httpUtility = new TestHttpUtility();

            ArmCertificateRetriever retriever = new ArmCertificateRetriever(this.dependencies.Configuration, imdsRetriever, httpUtility);
            IEnumerable<X509Certificate2> certs = await retriever.GetCertificates();

            Assert.True(certs.Count() != 0);
            foreach (X509Certificate2 c in certs)
            {
                Assert.StartsWith(expectedSubNamePrefix, c.Subject);
            }
        }

        [Fact]
        public async void Test_AzureEnvironment_Not_Fetched_If_Specified()
        {
            string clientSpecifiedEnv = "dogfood";
            string imdsFetchedEnv = "azurecloud";
            string expectedSubNamePrefix = $"CN={clientSpecifiedEnv}";

            IImdsRetriever imdsRetriever = new TestImdsRetriever(imdsFetchedEnv);
            IHttpUtility httpUtility = new TestHttpUtility();

            ArmCertificateRetriever retriever = new ArmCertificateRetriever(this.dependencies.Configuration, imdsRetriever, httpUtility, clientSpecifiedEnv);
            IEnumerable<X509Certificate2> certs = await retriever.GetCertificates();

            Assert.True(certs.Count() != 0);
            foreach (X509Certificate2 c in certs)
            {
                Assert.StartsWith(expectedSubNamePrefix, c.Subject);
            }
        }

        [Fact]
        public async void Test_AzureEnvironment_Is_Fetched_If_Not_Specified()
        {
            string imdsFetchedEnv = "azurecloud";
            string expectedSubNamePrefix = $"CN={imdsFetchedEnv}";

            IImdsRetriever imdsRetriever = new TestImdsRetriever(imdsFetchedEnv);
            IHttpUtility httpUtility = new TestHttpUtility();

            ArmCertificateRetriever retriever = new ArmCertificateRetriever(this.dependencies.Configuration, imdsRetriever, httpUtility);
            IEnumerable<X509Certificate2> certs = await retriever.GetCertificates();

            Assert.True(certs.Count() != 0);
            foreach (X509Certificate2 c in certs)
            {
                Assert.StartsWith(expectedSubNamePrefix, c.Subject);
            }
        }

        [Fact]
        public async void Test_Certificates_Are_Cached()
        {
            string imdsFetchedEnv = "azurecloud";

            IImdsRetriever imdsRetriever = new TestImdsRetriever(imdsFetchedEnv);
            IHttpUtility httpUtility = new TestHttpUtility();

            ArmCertificateRetriever retriever = new ArmCertificateRetriever(this.dependencies.Configuration, imdsRetriever, httpUtility);

            // Fetch certs and Validate that certs got fetched
            IEnumerable<X509Certificate2> certs = await retriever.GetCertificates();
            Assert.True(certs.Count() != 0);

            // Fetch certs again over and over and validate that thumbprint doesn't change.
            // We have setup cert fetching in testhttp utility such that cert is regenerated
            // on each call leading to change in thumbprint
            HashSet<string> thumbprints = certs.Select(c => c.Thumbprint).ToHashSet();
            for (int i = 0; i < 10; ++i)
            {
                certs = await retriever.GetCertificates();
                Assert.True(certs.Count() != 0);

                foreach (X509Certificate2 c in certs)
                {
                    Assert.Contains(c.Thumbprint, thumbprints);
                }
            }
        }

        [Fact]
        public async void Test_Certificate_Cache_Is_Refreshed_Periodically()
        {
            string imdsFetchedEnv = "azurecloud";

            IImdsRetriever imdsRetriever = new TestImdsRetriever(imdsFetchedEnv);
            IHttpUtility httpUtility = new TestHttpUtility();

            ArmCertificateRetriever retriever = new ArmCertificateRetriever(this.dependencies.Configuration, imdsRetriever, httpUtility, null, 30);
            IEnumerable<X509Certificate2> certs = await retriever.GetCertificates();
            Assert.True(certs.Count() != 0);

            // Fetch certs again over and over and validate that thumbprint doesn't change.
            // We have setup cert fetching in testhttp utility such that cert is regenerated
            // on each call leading to change in thumbprint
            HashSet<string> thumbprints = certs.Select(c => c.Thumbprint).ToHashSet();
            for (int i = 0; i < 3; ++i)
            {
                certs = await retriever.GetCertificates();
                Assert.True(certs.Count() != 0);

                foreach (X509Certificate2 c in certs)
                {
                    Assert.Contains(c.Thumbprint, thumbprints);
                }
            }

            // Wait for cached data to expire. 30 seconds is the expire interval we set in the constructor above.
            // Lets wait a bit more than that for cache interval to expire
            await Task.Delay(TimeSpan.FromSeconds(40));

            // Fetch certs and Validate that certs got fetched. Validate that cert thumbprint changed
            certs = await retriever.GetCertificates();
            Assert.True(certs.Count() != 0);
            foreach (X509Certificate2 c in certs)
            {
                Assert.DoesNotContain(c.Thumbprint, thumbprints);
            }
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
        [Fact]
        public async void Test_All_Azure_Environments_Work()
        {
            string[] azEnvs = new string[] { "AzureChinaCloud", "AzureCloud", "AzureGermanCloud", "AzureUSGovernment" };

            foreach (string e in azEnvs)
            {
                string expectedSubNamePrefix = $"CN={e.ToLower()}";

                IImdsRetriever imdsRetriever = new TestImdsRetriever(e);
                IHttpUtility httpUtility = new TestHttpUtility();

                ArmCertificateRetriever retriever = new ArmCertificateRetriever(this.dependencies.Configuration, imdsRetriever, httpUtility);
                IEnumerable<X509Certificate2> certs = await retriever.GetCertificates();

                Assert.True(certs.Count() != 0);
                foreach (X509Certificate2 c in certs)
                {
                    Assert.StartsWith(expectedSubNamePrefix, c.Subject);
                }
            }
        }

        [Fact]
        public async void Test_Invalid_Environments_Throws()
        {
            string invalidEnv = "AzureInvalid";

            IImdsRetriever imdsRetriever = new TestImdsRetriever(invalidEnv);
            IHttpUtility httpUtility = new TestHttpUtility();
            ArmCertificateRetriever retriever = new ArmCertificateRetriever(this.dependencies.Configuration, imdsRetriever, httpUtility);
            await Assert.ThrowsAsync<InvalidAzureEnvironmentException>(async () => await retriever.GetCertificates());
        }

        [Fact]
        public async void Test_Multiple_Request_For_Retrieving_Certs()
        {
            string invalidEnv = "AzureCloud";

            IImdsRetriever imdsRetriever = new TestImdsRetriever(invalidEnv);
            IHttpUtility httpUtility = new TestHttpUtility();

            // repeat the whole test multiple times
            for (int itr = 0; itr < 100; ++itr)
            {
                // create 100 tasks for each iteration that will fetch certs
                ArmCertificateRetriever retriever = new ArmCertificateRetriever(this.dependencies.Configuration, imdsRetriever, httpUtility);
                List<Task<IEnumerable<X509Certificate2>>> taskList = new List<Task<IEnumerable<X509Certificate2>>>();
                for (int i = 0; i < 100; ++i)
                {
                    Task<IEnumerable<X509Certificate2>> t = retriever.GetCertificates();
                    taskList.Add(t);
                }

                await Task.WhenAll(taskList.ToArray());

                HashSet<int> certcounts = new HashSet<int>();
                HashSet<string> thumbprints = new HashSet<string>();
                foreach (Task<IEnumerable<X509Certificate2>> t in taskList)
                {
                    Assert.True(t.Result != null);
                    int count = t.Result.Count();
                    Assert.True(count != 0);

                    certcounts.Add(count);

                    foreach (X509Certificate2 c in t.Result)
                    {
                        thumbprints.Add(c.Thumbprint);
                    }
                }

                // All thread should have returned same number of certs
                Assert.True(certcounts.Count() == 1);
                Assert.True(thumbprints.Count() == certcounts.ElementAt(0));
            }
        }

        [Fact]
        public async void Test_GetCertificatesNoCache_API()
        {
            string imdsFetchedEnv = "azurecloud";

            IImdsRetriever imdsRetriever = new TestImdsRetriever(imdsFetchedEnv);
            IHttpUtility httpUtility = new TestHttpUtility();

            ArmCertificateRetriever retriever = new ArmCertificateRetriever(this.dependencies.Configuration, imdsRetriever, httpUtility);
            IEnumerable<X509Certificate2> certs = await retriever.GetCertificates();
            Assert.True(certs.Count() != 0);

            // Fetch certs again over and over and validate that thumbprint doesn't change.
            // We have setup cert fetching in testhttp utility such that cert is regenerated
            // on each call leading to change in thumbprint. Loop is being added here to test
            // this condition multiple times just to be sure
            HashSet<string> thumbprints = certs.Select(c => c.Thumbprint).ToHashSet();
            for (int i = 0; i < 3; ++i)
            {
                certs = await retriever.GetCertificates();
                Assert.True(certs.Count() != 0);

                foreach (X509Certificate2 c in certs)
                {
                    Assert.Contains(c.Thumbprint, thumbprints);
                }
            }

            // Fetch certs with NO cache API and validate that these certs are not present
            // in the previously hashset above.
            IEnumerable<X509Certificate2>  noCacheCerts = await retriever.ForceGetCertificates();
            Assert.True(certs.Count() != 0);
            foreach (X509Certificate2 c in noCacheCerts)
            {
                Assert.DoesNotContain(c.Thumbprint, thumbprints);
            }
        }
    }
}
