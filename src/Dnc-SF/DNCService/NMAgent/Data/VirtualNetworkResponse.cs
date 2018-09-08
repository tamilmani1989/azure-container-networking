using System;
using System.Collections.Generic;
using System.Text;
using Newtonsoft.Json;

namespace NMAgent.Data
{
    public class VirtualNetworkResponse
    {
        [JsonProperty("httpStatusCode")]
        public string ResponseCode { get; set; }

        [JsonProperty("vnetVersion")]
        public string VnetVersion { get; set; }

        [JsonProperty("cnetSpace")]
        public string CNetSpace { get; set; }

        [JsonProperty("vnetSpace")]
        public string VNetSpace { get; set; }

        [JsonProperty("subnets")]
        public Subnet[]  Subnets { get; set; }

        [JsonProperty("dnsServers")]
        public string DNSServers { get; set; }

        public VirtualNetworkResponse(
            string responseCode, 
            string vnetVersion, 
            string cnetSpace, 
            string vnetSpace,
            Subnet[] subnets,
            string dnsServers)
        {
            this.ResponseCode = responseCode;
            this.VnetVersion = vnetVersion;
            this.CNetSpace = cnetSpace;
            this.VNetSpace = vnetSpace;
            this.Subnets = subnets;
            this.DNSServers = dnsServers;
        }

        public override string ToString()
        {
            string subnets = this.Subnets != null ? string.Join(",", (object[])this.Subnets) : "null";
            return $"ResponseCode:{this.ResponseCode} VnetVersion:{this.VnetVersion} " +
                $"CNetSpace:{this.CNetSpace} VNetSpace:{this.VNetSpace} Subnets:{subnets}" +
                $"DNSServers:{this.DNSServers} ";
        }
    }
}
