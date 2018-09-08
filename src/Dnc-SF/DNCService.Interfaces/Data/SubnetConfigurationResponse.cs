using System;
using System.Collections.Generic;
using System.Runtime.Serialization;
using System.Text;

namespace DNCService.API
{
    [DataContract]
    public class SubnetConfigurationResponse
    {
        [DataMember]
        public string NetworkID { get; private set; }
        [DataMember]
        public string SubnetName { get; private set; }
        [DataMember]
        public string IPSubnet { get; private set; }
        [DataMember]
        public string GatewayIP { get; private set; }
        [DataMember]
        public  string[] DNSServers { get; private set; }
        [DataMember]
        public string AccessToken { get; private set; }
        [DataMember]
        public IPAllocationData[] IPAllocationsData { get; private set; }


        public SubnetConfigurationResponse(string networkID, string subnetName, string ipSubnet, string gatewayIP, string[] dnsServers, string accessToken, IPAllocationData[] allocationDatas)
        {
            this.NetworkID = networkID;
            this.SubnetName = subnetName;
            this.IPSubnet = ipSubnet;
            this.GatewayIP = gatewayIP;
            this.DNSServers = dnsServers;
            this.AccessToken = accessToken;
            this.IPAllocationsData = allocationDatas;
        }

        public override string ToString()
        {
            string allocationDatas = this.IPAllocationsData != null ? string.Join(",", (object[])this.IPAllocationsData) : "null";
            return $"NetworkID:{this.NetworkID} SubnetName:{this.SubnetName} IPSubnet:{this.IPSubnet} " +
                $"GatewayIP:{this.GatewayIP} DNSServers:{this.DNSServers} AccessToken:{this.AccessToken} IPAllocationsData:{allocationDatas}";
        }
    }
}
