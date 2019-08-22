//--------------------------------------------------------------------------
// <copyright file="DncConfiguration.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
// Implements the DNC service.
// </summary>
//--------------------------------------------------------------------------

using Newtonsoft.Json;
using System.IO;

namespace DNCService
{
    /// <summary>
    /// Defines the host agent settings.
    /// </summary>
    public class HostAgentSettings
    {
        /// <summary>
        /// IP to use to connect to the host.
        /// </summary>
        public string WireServerIP { get; set; }

        /// <summary>
        /// URL strings.
        /// </summary>
        public class HostAgentConnectionStrings
        {
            /// <summary>
            /// Join network URL.
            /// </summary>
            public string JoinNetworkURLFmt { get; set; }

            /// <summary>
            /// Get network URL.
            /// </summary>
            public string GetNetworkConfigurationURLFmt { get; set; }

            /// <summary>
            /// Put network container URL.
            /// </summary>
            public string PutNetworkContainerURLFmt { get; set; }

            /// <summary>
            /// Delete network container URL.
            /// </summary>
            public string DeleteNetworkContainerURLFmt { get; set; }

            /// <summary>
            /// Get tag definitions URL.
            /// </summary>
            public string GetSystemTagDefinitionURLFmt { get; set; }
        }

        /// <summary>
        /// Host agent connection strings.
        /// </summary>
        public HostAgentConnectionStrings ConnectionStrings { get; set; }
    }

    /// <summary>
    /// Defines the CNS settings.
    /// </summary>
    public class CNSSettings
    {
        /// <summary>
        /// Defines the cns connection strings.
        /// </summary>
        public class CNSConnectionStrings
        {
            /// <summary>
            /// Create NC URL.
            /// </summary>
            public string CreateOrUpdateNetworkContainerURLFmt { get; set; }

            /// <summary>
            /// Delete NC URL.
            /// </summary>
            public string DeleteNetworkContainerURLFmt { get; set; }

            /// <summary>
            /// SetOrchestratorType URL.
            /// </summary>
            public string SetOrchestratorTypeURLFmt { get; set; }

            /// <summary>
            /// Get NC status URL.
            /// </summary>
            public string GetNetworkcontainerstatusURLFmt { get; set; }
        }

        /// <summary>
        /// CNS connection strings.
        /// </summary>
        public CNSConnectionStrings ConnectionStrings { get; set; }
    }

    /// <summary>
    /// Defines Vlan settings.
    /// </summary>
    public class VlanSettings
    {
        /// <summary>
        /// Min Vlan ID.
        /// </summary>
        public int Min { get; set; }

        /// <summary>
        /// Max Vlan ID.
        /// </summary>
        public int Max { get; set; }
    }

    /// <summary>
    /// Defines DNC specific settings.
    /// </summary>
    public class DNCSettings
    {
        /// <summary>
        /// Backoff base interval on failures.
        /// </summary>
        public uint RequestRetryBaseInterval { get; set; }

        /// <summary>
        /// Network refresh interval.
        /// </summary>
        public uint NetworkRefreshInterval { get; set; }

        /// <summary>
        /// Tag refresh interval.
        /// </summary>
        public uint TagRefreshInterval { get; set; }

        /// <summary>
        /// Network container refresh interval.
        /// </summary>
        public uint NetworkContainerRefreshInterval { get; set; }

        /// <summary>
        /// Concurrent requests to process
        /// </summary>
        public uint ConcurrentRequestsToProcess { get; set; }
    }

    /// <summary>
    /// Defines DataStore specific settings.
    /// </summary>
    public class DataStoreSettings
    {
        /// <summary>
        /// Defines AzureStorage specific settings.
        /// </summary>
        public class AzStorageInfo
        {
            /// <summary>
            /// Azure storage account name
            /// </summary>
            public string AccountName { get; set; }

            /// <summary>
            /// Azure storage account key
            /// </summary>
            public string Key { get; set; }

            /// <summary>
            /// Flag to enable shared subnet delegation feature
            /// </summary>
            public bool EnableSharedSubnetDelegation { get; set; }
        }

        /// <summary>
        /// Azure storage settings.
        /// </summary>
        public AzStorageInfo AzureStorageInfo { get; set; }
    }

    /// <summary>
    /// Defines the Dnc configuration.
    /// </summary>
    public class DncConfiguration
    {
        /// <summary>
        /// DataStore settings.
        /// </summary>
        public DataStoreSettings DataStoreSettings { get; set; }

        /// <summary>
        /// Host agent settings.
        /// </summary>
        public HostAgentSettings HostAgentSettings { get; set; }

        /// <summary>
        /// Cns settings.
        /// </summary>
        public CNSSettings CNSSettings { get; set; }

        /// <summary>
        /// Vlan settings.
        /// </summary>
        public VlanSettings VlanSettings { get; set; }

        /// <summary>
        /// Dnc settings.
        /// </summary>
        public DNCSettings DNCSettings { get; set; }

        /// <summary>
        /// Simulator mode for CNS.
        /// </summary>
        public bool CnsSimulatorMode { get; set; }

        /// <summary>
        /// Simulator mode for Host agent.
        /// </summary>
        public bool NMAgentSimulatorMode { get; set; }

        /// <summary>
        /// Helper to deserialize DNC configuration from a file.
        /// </summary>
        /// <param name="fileName">Supply the file name with the json configuration.</param>
        /// <returns>Dnc configuration.</returns>
        public static DncConfiguration Deserialize(string fileName)
        {
            using (StreamReader file = File.OpenText(fileName))
            {
                JsonSerializer serializer = new JsonSerializer();
                return (DncConfiguration)serializer.Deserialize(file, typeof(DncConfiguration));
            }
        }
    }
}