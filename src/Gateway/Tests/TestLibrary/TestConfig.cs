// --------------------------------------------------------------------------------------------------------------------
// <copyright file="TestConfig.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.Gateway.TestLib
{
    using System;
    using System.Globalization;
    using System.IO;

    using Microsoft.Extensions.Configuration;

    public class TestConfig
    {
        public string Instance
        {
            get; set;
        } = "https://login.microsoftonline.com/{0}";

        public string TenantId
        {
            get; set;
        }
        public string ClientId
        {
            get; set;
        }
        public string Authority
        {
            get
            {
                return String.Format(CultureInfo.InvariantCulture,
                                     Instance, TenantId);
            }
        }
        public string ClientSecret
        {
            get; set;
        }
        public string BaseAddress
        {
            get; set;
        }
        public string ResourceID
        {
            get; set;
        }

        public int StressTestIterations
        {
            get; set;
        }

        public int StressTestThreadCount
        {
            get; set;
        }

        public bool UseClientCertificate
        {
            get; set;
        }

        public string ClientCertificatePath
        {
            get; set;
        }

        public bool DecryptToken
        {
            get; set;
        }

        public string DecryptTokenCertificatePath
        {
            get; set;
        }

        public static TestConfig ReadFromJsonFile(string path)
        {
            IConfiguration Configuration;

            var builder = new ConfigurationBuilder()
              .SetBasePath(Directory.GetCurrentDirectory())
              .AddJsonFile(path);

            Configuration = builder.Build();

            return Configuration.Get<TestConfig>();
        }
    }
}
