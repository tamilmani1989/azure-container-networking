//-----------------------------------------------------------------------
// <copyright file="CreateNetworkContainerRequest.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// 
//-----------------------------------------------------------------------

using API;
using Newtonsoft.Json;
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
        [JsonConverter(typeof(ByteArrayConverter))]
        public byte[] OrchestratorContext { get; private set; }

        /// <summary>
        /// IP configuration associated with the network container.
        /// </summary>
        [JsonProperty("IPConfiguration")]
        public IPConfiguration IPConfiguration { get; private set; }

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
        /// <param name="orchestratorContext"></param>
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
            byte[] orchestratorContext,
            IPConfiguration ipConfiguration,
            MultiTenancyInfo multiTenancyInfo,
            IPSubnet[] cnetAddressSpace,
            Route[] routes)
        {
            this.Version = version;
            this.NetworkContainerType = ncType;
            this.NetworkContainerID = ncID;
            this.PrimaryInterfaceIdentifier = interfaceID;
            this.AuthorizationToken = accessToken;
            this.OrchestratorContext = orchestratorContext;
            this.IPConfiguration = ipConfiguration;
            this.MultiTenancyInfo = multiTenancyInfo;
            this.CnetAddressSpace = cnetAddressSpace;
            this.Routes = routes;
        }

        /// <summary>
        /// ToString implementation
        /// </summary>
        /// <returns>String representation of the object.</returns>
        public override string ToString()
        {
            string orchestratorContextStr = this.OrchestratorContext != null ? Encoding.Default.GetString(this.OrchestratorContext) : "null";
            string cnetAddressSpaceStr = this.CnetAddressSpace != null ?
                string.Join(",", (object[])this.CnetAddressSpace) : "null";

            string routesStr = this.Routes != null ?
                string.Join(",", (object[])this.Routes) : "null";

            string tokenHashString = "";
            if (!string.IsNullOrWhiteSpace(this.AuthorizationToken))
            {
                using (var algorithm = SHA256.Create())
                {
                    byte[] tokenHash = algorithm.ComputeHash(Encoding.ASCII.GetBytes(this.AuthorizationToken));
                    tokenHashString = System.Text.Encoding.Default.GetString(tokenHash);
                }
            }
            return $"[Version:{this.Version} NetworkContainerType:{this.NetworkContainerType} " +
                    $"NetworkContainerID:{this.NetworkContainerID} " +
                    $"PrimaryInterfaceIdentifier:{this.PrimaryInterfaceIdentifier} " +
                    $"AuthorizationToken : {tokenHashString}" +
                    $"OrchestratorContext :{orchestratorContextStr} " +
                    $"IPConfiguration:{this.IPConfiguration} MultiTenancyInfo:{this.MultiTenancyInfo} " +
                    $"CnetAddressSpace:{cnetAddressSpaceStr} Routes:{routesStr}]";
        }
    }
}