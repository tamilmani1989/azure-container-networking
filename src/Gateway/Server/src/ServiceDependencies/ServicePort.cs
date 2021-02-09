// --------------------------------------------------------------------------------------------------------------------
// <copyright file="ServicePort.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.Gateway
{
    using System;

    public class ServicePort
    {
        /// <summary>
        /// Gets or sets the service name.
        /// </summary>
        public string ServiceName
        {
            get; set;
        }

        /// <summary>
        /// The management port.
        /// </summary>
        public ushort Port
        {
            get; set;
        }

        /// <summary>
        /// Gets or sets the scheme.
        /// </summary>
        public string Scheme
        {
            get; set;
        }

        /// <summary>
        /// Gets or sets the host name.
        /// </summary>
        public string Host
        {
            get; set;
        }

        /// <summary>
        /// Gets or sets the environment variable name for host.
        /// </summary>
        public string HostEnvironmentVariable
        {
            get; set;
        }

        /// <summary>
        /// Get host name
        /// </summary>
        /// <returns></returns>
        public string GetHostName()
        {
            string host = "127.0.0.1";

            if (!string.IsNullOrWhiteSpace(this.Host))
            {
                host = this.Host;
            }

            if (!string.IsNullOrWhiteSpace(this.HostEnvironmentVariable))
            {
                string hostFromEnvVar = Environment.GetEnvironmentVariable(this.HostEnvironmentVariable);
                if (!string.IsNullOrWhiteSpace(hostFromEnvVar))
                {
                    host = hostFromEnvVar;
                }
            }

            return host;
        }
    }
}