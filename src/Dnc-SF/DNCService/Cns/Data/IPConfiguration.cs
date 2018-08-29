//-----------------------------------------------------------------------
// <copyright file="IPConfiguration.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// 
//-----------------------------------------------------------------------

using Newtonsoft.Json;
using System.Collections.Generic;

namespace Cns.Data
{
    /// <summary>
    /// Defines the IPConfiguration class.
    /// </summary>
    public class IPConfiguration
    {
        /// <summary>
        /// IPSubnet.
        /// </summary>
        [JsonProperty("IPSubnet")]
        public IPSubnet IPSubnet { get; private set; }

        /// <summary>
        /// List of Dns servers.
        /// </summary>
        [JsonProperty("DNSServers")]
        public string[] DNSServers { get; private set; }

        /// <summary>
        /// Gateway IP associated with the IP configuration.
        /// </summary>
        [JsonProperty("GatewayIPAddress")]
        public string GatewayIPAddress { get; private set; }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="ipSubnet"></param>
        /// <param name="dnsServers"></param>
        /// <param name="gatewayIPAddress"></param>
        public IPConfiguration(IPSubnet ipSubnet, List<string> dnsServers, string gatewayIPAddress)
        {
            this.IPSubnet = ipSubnet;
            this.DNSServers = dnsServers.ToArray();
            this.GatewayIPAddress = gatewayIPAddress;
        }

        /// <summary>
        /// ToString implementation
        /// </summary>
        /// <returns>String representation of the object.</returns>
        public override string ToString()
        {
            return $"IPSubnet:{this.IPSubnet}, DnsServers:{string.Join(",", this.DNSServers)}, " +
                $"GatewayIPAddress:{this.GatewayIPAddress}";
        }
    }
}
