// --------------------------------------------------------------------------------------------------------------------
// <copyright file="ServicePorts.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.Gateway
{
    using System;
    using System.Collections.Generic;
    using System.IO;
    using System.Linq;
    using System.Reflection;

    using Newtonsoft.Json;

    public class ServicePorts : IServicePorts
    {
        private Dictionary<string, ServicePort> mapping;

        private object lockObject;

        private const string servicePortsFileName = "ServicePorts.json";

        public ServicePorts()
        {
            this.Services = new List<ServicePort>();
            this.lockObject = new object();
        }

        /// <summary>
        /// List of services in the map
        /// </summary>
        public List<ServicePort> Services
        {
            get; set;
        }

        public IDictionary<string, ServicePort> GetServicePortsMapping()
        {
            if (this.mapping == null)
            {
                lock (this.lockObject)
                {
                    if (this.mapping == null)
                    {
                        this.mapping = this.Services.ToDictionary(
                                           s => s.ServiceName,
                                           StringComparer.OrdinalIgnoreCase);

                        this.ValidateServicePorts();
                    }
                }
            }

            return this.mapping;
        }

        /// <summary>
        /// Load service ports. It expects ServicePorts.json to be present
        /// in the current directory
        /// </summary>
        /// <returns>List of service ports.</returns>
        public static ServicePorts Load()
        {
            string filePath = Path.Combine(GetCurrentAssemblyPath(), servicePortsFileName);
            return JsonConvert.DeserializeObject<ServicePorts>(File.ReadAllText(filePath));
        }

        /// <summary>
        /// Load service ports from specified file
        /// </summary>
        /// <param name="filePath">Path to json file which contains service ports</param>
        /// <returns>List of service ports.</returns>
        public static ServicePorts Load(string filePath)
        {
            return JsonConvert.DeserializeObject<ServicePorts>(File.ReadAllText(filePath));
        }

        /// <summary>
        /// Get path where current dll is loaded from
        /// </summary>
        /// <returns></returns>
        private static string GetCurrentAssemblyPath()
        {
            string codeBase = Assembly.GetExecutingAssembly().CodeBase;
            UriBuilder uri = new UriBuilder(codeBase);
            string path = Uri.UnescapeDataString(uri.Path);
            return Path.GetDirectoryName(path);
        }

        /// <summary>
        /// Validate Service ports file
        /// </summary>
        private void ValidateServicePorts()
        {
            foreach (ServicePort sp in this.mapping.Values)
            {
                if (string.IsNullOrWhiteSpace(sp.ServiceName))
                {
                    throw new InvalidServiceNameException("ServicePorts.json has an entry with empty service name");
                }

                if (string.IsNullOrWhiteSpace(sp.Scheme))
                {
                    throw new InvalidSchemeException($"Service '{sp.ServiceName}' has empty scheme");
                }

                if (!(sp.Port > 0))
                {
                    throw new InvalidServicePortException($"Service '{sp.ServiceName}' has empty port");
                }

                if (string.IsNullOrWhiteSpace(sp.Host) && string.IsNullOrWhiteSpace(sp.HostEnvironmentVariable))
                {
                    throw new InvalidHostException($"Service '{sp.ServiceName}' has both Host and HostEnvironmentVariable as empty. One should be specified.");
                }
            }
        }
    }
}
