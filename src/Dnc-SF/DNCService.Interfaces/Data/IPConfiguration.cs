//--------------------------------------------------------------------------
// <copyright file="IPConfiguration.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
// Definititon for IP configuration.
// </summary>
//--------------------------------------------------------------------------
using System.Runtime.Serialization;

namespace DNCService.API
{
    /// <summary>
    /// Class for IPConfiguration.
    /// </summary>
    [DataContract]
    public class IPConfiguration
    {
        /// <summary>
        /// Id of the network.
        /// </summary>
        [DataMember]
        public string NetworkID { get; private set; }

        /// <summary>
        /// Subnet associated with the network.
        /// </summary>
        [DataMember]
        public string SubnetName { get; private set; }

        /// <summary>
        /// IPAddress space associated with the subnet.
        /// </summary>
        [DataMember]
        public string IPSubnet { get; private set; }

        /// <summary>
        /// Allcoated IP.
        /// </summary>
        [DataMember]
        public string IPAddress { get; private set; }

        /// <summary>
        /// Gateway IP.
        /// </summary>
        [DataMember]
        public string GatewayIP { get; private set; }

        /// <summary>
        /// Dns servers associated with the IP configuration.
        /// </summary>
        [DataMember]
        public string[] DNSServers { get; private set; }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="networkID">Supply the ID of the network.</param>
        /// <param name="subnetName">Supply the name of the subnet.</param>
        /// <param name="ipSubnet">Supply the ip address space.</param>
        /// <param name="ipAddress">Supply the allocated IP address.</param>
        /// <param name="gatewayIP">Supply the gateway IP.</param>
        /// <param name="dnsServers">Supply the dns serves list</param>
        public IPConfiguration(string networkID, string subnetName, string ipSubnet, string ipAddress, string gatewayIP, string[] dnsServers)
        {
            this.NetworkID = networkID;
            this.SubnetName = subnetName;
            this.IPSubnet = ipSubnet;
            this.IPAddress = ipAddress;
            this.GatewayIP = GatewayIP;
            this.DNSServers = dnsServers;
        }

        /// <summary>
        /// ToString implementation.
        /// </summary>
        /// <returns>String representation of the IPConfiguraton object.</returns>
        public override string ToString()
        {
            return $"[NetworkID:{this.NetworkID} SubnetName:{this.SubnetName} IPSubnet:{this.IPSubnet} IPAddress:{this.IPAddress} " +
                $"GatewayIP:{this.GatewayIP} DNSServers:{this.DNSServers}]";
        }
    }
}
