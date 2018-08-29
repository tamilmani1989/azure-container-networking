using Newtonsoft.Json;
using System;
using System.Collections.Generic;
using System.Text;

namespace NMAgent.Data
{
    public class CreateOrUpdateNetworkContainerRequest
    {
        [JsonProperty("virtualNetworkId")]
        public string VirtualNetworkID { get; set; }

        [JsonProperty("ipV4Addresses")]
        public string[] IPV4Addresses { get; set; }

        [JsonProperty("subnetName")]
        public string SubnetName { get; set; }

        [JsonProperty("version")]
        public string Version { get; set; }

        [JsonProperty("policies")]
        public string[] Policies { get; set; }

        [JsonProperty("vlanId")]
        public int VlanID { get; set; }

        public CreateOrUpdateNetworkContainerRequest(
            string virtualNetworkID,
            string[] ipV4Addresses, 
            string subnetName, 
            string version, 
            string[] policies,
            string vlanID)
        {
            this.VirtualNetworkID = virtualNetworkID;
            this.IPV4Addresses = ipV4Addresses;
            this.SubnetName = subnetName;
            this.Version = version;
            this.Policies = policies;
            this.VlanID = VlanID;
        }

        public override string ToString()
        {
            string ipV4Addresses = this.IPV4Addresses != null ? string.Join(",", this.IPV4Addresses) : "null";
            string policies = this.Policies != null ? string.Join(",", this.Policies) : "null";
            return $"VirtualNetworkID:{this.VirtualNetworkID} Version:{this.Version}  " +
                $"IPV4Addresses:{ipV4Addresses} Policies:{policies} VlanID:{this.VlanID}";
        }
    }
}
