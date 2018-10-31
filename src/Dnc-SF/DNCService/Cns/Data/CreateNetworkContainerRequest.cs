//-----------------------------------------------------------------------
// <copyright file="CreateNetworkContainerRequest.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// 
//-----------------------------------------------------------------------

using Newtonsoft.Json;
using Newtonsoft.Json.Linq;
using System.Security.Cryptography;
using System.Text;

namespace Cns.Data
{
    /// <summary>
    /// Defines the create network container request.
    /// </summary>
    public class CreateNetworkContainerRequest
    {
        /// <summary>
        /// Version of the network container.
        /// </summary>
        [JsonProperty("Version")]
        public string Version { get; private set; }

        /// <summary>
        /// Type of the container.
        /// </summary>
        [JsonProperty("NetworkContainerType")]
        public string NetworkContainerType { get; private set; }

        /// <summary>
        /// Id of the container.
        /// </summary>
        [JsonProperty("NetworkContainerid")]
        public string NetworkContainerID { get; private set; }

        /// <summary>
        /// Interface identifier.
        /// </summary>
        [JsonProperty("PrimaryInterfaceIdentifier")]
        public string PrimaryInterfaceIdentifier { get; private set; }

        /// <summary>
        /// Authroization token.
        /// </summary>
        [JsonProperty("AuthorizationToken")]
        public string AuthorizationToken { get; private set; }

        /// <summary>
        /// Orchestrator context.
        /// </summary>
        [JsonProperty("OrchestratorContext")]
        public JRaw OrchestratorContext { get; set; }

        /// <summary>
        /// IP configuration associated with the network container.
        /// </summary>
        [JsonProperty("IPConfiguration")]
        public IPConfiguration IPConfiguration { get; private set; }

        /// <summary>
        /// Local IP configuration associated with the network container.
        /// </summary>
        [JsonProperty("LocalIPConfiguration")]
        public IPConfiguration LocalIPConfiguration { get; private set; }

        /// <summary>
        /// Multitenancy info.
        /// </summary>
        [JsonProperty("MultiTenancyInfo")]
        public MultiTenancyInfo MultiTenancyInfo { get; private set; }

        /// <summary>
        /// List of cnet addresses.
        /// </summary>
        [JsonProperty("CnetAddressSpace")]
        public IPSubnet[] CnetAddressSpace { get; private set; }

        /// <summary>
        /// List of routes.
        /// </summary>
        [JsonProperty("Routes")]
        public Route[] Routes { get; private set; }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="version"></param>
        /// <param name="ncType"></param>
        /// <param name="ncID"></param>
        /// <param name="interfaceID"></param>
        /// <param name="accessToken"></param>
        /// <param name="ipConfiguration"></param>
        /// <param name="multiTenancyInfo"></param>
        /// <param name="cnetAddressSpace"></param>
        /// <param name="routes"></param>
        public CreateNetworkContainerRequest(
            string version,
            string ncType,
            string ncID,
            string interfaceID,
            string accessToken,
            IPConfiguration ipConfiguration,
            IPConfiguration localIPConfiguration,
            MultiTenancyInfo multiTenancyInfo,
            IPSubnet[] cnetAddressSpace,
            Route[] routes)
        {
            Version = version;
            NetworkContainerType = ncType;
            NetworkContainerID = "Swift_" + ncID;
            PrimaryInterfaceIdentifier = interfaceID;
            AuthorizationToken = accessToken;
            IPConfiguration = ipConfiguration;
            LocalIPConfiguration = localIPConfiguration;
            MultiTenancyInfo = multiTenancyInfo;
            CnetAddressSpace = cnetAddressSpace;
            Routes = routes;
        }

        /// <summary>
        /// ToString implementation
        /// </summary>
        /// <returns>String representation of the object.</returns>
        public override string ToString()
        {
            string orchestratorContextStr =
                OrchestratorContext != null ? OrchestratorContext.ToString() : "null";
            string cnetAddressSpaceStr = CnetAddressSpace != null ?
                string.Join(",", (object[])CnetAddressSpace) : "null";

            string routesStr = Routes != null ?
                string.Join(",", (object[])Routes) : "null";

            string tokenHashString = "";
            if (!string.IsNullOrWhiteSpace(AuthorizationToken))
            {
                using (var algorithm = SHA256.Create())
                {
                    byte[] tokenHash = algorithm.ComputeHash(Encoding.ASCII.GetBytes(AuthorizationToken));
                    tokenHashString = Encoding.Default.GetString(tokenHash);
                }
            }
            return $"[Version:{Version} NetworkContainerType:{NetworkContainerType} " +
                    $"NetworkContainerID:{NetworkContainerID} " +
                    $"PrimaryInterfaceIdentifier:{PrimaryInterfaceIdentifier} " +
                    $"AuthorizationToken : {tokenHashString}" +
                    $"OrchestratorContext :{orchestratorContextStr} " +
                    $"IPConfiguration:{IPConfiguration} LocalIPConfiguration:{LocalIPConfiguration} " +
                    $"MultiTenancyInfo:{MultiTenancyInfo} " +
                    $"CnetAddressSpace:{cnetAddressSpaceStr} Routes:{routesStr}]";
        }
    }
}