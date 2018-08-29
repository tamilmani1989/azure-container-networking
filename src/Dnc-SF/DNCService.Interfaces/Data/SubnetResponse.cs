//--------------------------------------------------------------------------
// <copyright file="SubnetResponse.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
// Class that defines a SubnetResponse 
// </summary>
//--------------------------------------------------------------------------

using System.Runtime.Serialization;

namespace DNCService.API
{
    /// <summary>
    /// Class definitition for subnet response.
    /// </summary>
    [DataContract]
    public class SubnetResponse
    {
        /// <summary>
        /// Id of the network to which the subnet is part of.
        /// </summary>
        [DataMember]
        public string NetworkID { get; private set; }

        /// <summary>
        /// Name of the subnet.
        /// </summary>
        [DataMember]
        public string SubnetName { get; private set; }

        /// <summary>
        /// IPAddress space associated with the subnet.
        /// </summary>
        [DataMember]
        public string IPSubnet { get; private set; }

        /// <summary>
        /// Default gateway IP.
        /// </summary>
        [DataMember]
        public string GatewayIP { get; private set; }

        /// <summary>
        /// Reserved DNS servers in the subnet.
        /// </summary>
        [DataMember]
        public  string[] DNSServers { get; private set; }

        /// <summary>
        /// Access token for delegation.
        /// </summary>
        [DataMember]
        public string AccessToken { get; private set; }

        /// <summary>
        /// Allocations in the subnet.
        /// </summary>
        [DataMember]
        public IPAllocationResponse[] IPAllocationsData { get; private set; }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="networkID">Supply the network ID.</param>
        /// <param name="subnetName">Supply the name of the subnet.</param>
        /// <param name="ipSubnet">Supply the IPsubnet space.</param>
        /// <param name="gatewayIP">Supply the gateway IP.</param>
        /// <param name="dnsServers">Supply the Dns servers list.</param>
        /// <param name="accessToken">Supply the access token.</param>
        /// <param name="allocationDatas">Supply the allocations.</param>
        public SubnetResponse(
            string networkID, 
            string subnetName, 
            string ipSubnet, 
            string gatewayIP, 
            string[] dnsServers, 
            string accessToken,
            IPAllocationResponse[] allocationDatas)
        {
            this.NetworkID = networkID;
            this.SubnetName = subnetName;
            this.IPSubnet = ipSubnet;
            this.GatewayIP = gatewayIP;
            this.DNSServers = dnsServers;
            this.AccessToken = accessToken;
            this.IPAllocationsData = allocationDatas;
        }

        /// <summary>
        /// ToString implementation.
        /// </summary>
        /// <returns>String representation of the SubnetResponse object.</returns>
        public override string ToString()
        {
            string allocationDatas = this.IPAllocationsData != null ? string.Join(",", (object[])this.IPAllocationsData) : "null";
            return $"NetworkID:{this.NetworkID} SubnetName:{this.SubnetName} IPSubnet:{this.IPSubnet} " +
                $"GatewayIP:{this.GatewayIP} DNSServers:{this.DNSServers} AccessToken:{this.AccessToken} IPAllocationsData:{allocationDatas}";
        }
    }
}
