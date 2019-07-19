using Cns.Data;
using DNCService;
using NetworkManager.DataModel;
using NMAgent.Data;
using System;
using System.Collections.Generic;
using System.Security.Cryptography;
using System.Text;

namespace NetworkManager
{
    // TODO: Add comments
    public class NetworkContainerInfo : ICloneable
    {
        public string NetworkID { private set; get; }

        public string NetworkContainerID { private set; get; }

        public string NodeID { private set; get; }

        public string NodeInterfaceID { private set; get; }

        public string AccessToken { private set; get; }

        public List<string> IPV4Addresses { get; set; }

        public string SubnetName { get; set; }

        public ulong Version { get; set; }

        public ulong CnsVersion { get; private set; }

        public List<string> Policies { get; set; }

        public int VlanID { get; set; }

        public string OrchestratorContext { get; set; }

        public string ContainerType { get; private set; }

        public string Subnet { get; private set; }

        public string GatewayIP { get; private set; }

        public List<string> DnsServers { get; private set; }

        public List<string> CnetAddressSpace { get; private set; }

        public NetworkContainerState NetworkContainerState { get; private set; }

        public Status Status { get; private set; }

        public string LocalIP { get; private set; }

        public List<Tag> Tags { get; private set; }

        public UInt64 VnetVersion { get; private set; }

        /// <summary>
        /// Cnet address list
        /// </summary>
        public List<Cns.Data.IPSubnet> CnsCnetAddresses { get; private set; }

        public NetworkContainerInfo(
            string networkID,
            string nodeID,
            string ncID,
            string nodeInterfaceID,
            string accessToken,
            List<string> ipv4Addresses,
            string subnetName,
            string ipSubnet,
            List<string> dnsServers,
            string gatewayIP,
            ulong version,
            List<string> policies,
            int vlanID,
            ulong cnsVersion,
            string containerType,
            string orchestratorContext,
            List<string> cnetAddressSpace,
            NetworkContainerState networkContainerState,
            Status status,
            string localIP,
            ICollection<Tag> tagSet,
            UInt64 vnetVersion)
        {
            this.NetworkID = networkID;
            this.NodeID = nodeID;
            this.NetworkContainerID = ncID;
            this.NodeInterfaceID = nodeInterfaceID;
            this.AccessToken = accessToken;
            this.IPV4Addresses = ipv4Addresses;
            this.SubnetName = subnetName;
            this.Subnet = ipSubnet;
            this.GatewayIP = gatewayIP;
            this.DnsServers = dnsServers;
            this.Version = version;
            this.Policies = policies ?? new List<string>();
            this.VlanID = vlanID;
            this.CnsVersion = cnsVersion;
            this.ContainerType = containerType;
            this.OrchestratorContext = orchestratorContext;
            this.CnetAddressSpace = cnetAddressSpace;
            this.NetworkContainerState = networkContainerState;
            this.Status = status;
            this.LocalIP = localIP;
            this.Tags = new List<Tag>(tagSet);
            this.VnetVersion = vnetVersion;
            CnsCnetAddresses = new List<Cns.Data.IPSubnet>();
        }

        public NetworkContainerInfo(NetworkContainerInfo ncConfig)
        {
            this.NetworkID = ncConfig.NetworkID;
            this.NodeID = ncConfig.NodeID;
            this.NetworkContainerID = ncConfig.NetworkContainerID;
            this.NodeInterfaceID = ncConfig.NodeInterfaceID;
            this.AccessToken = ncConfig.AccessToken;
            this.IPV4Addresses = ncConfig.IPV4Addresses;
            this.SubnetName = ncConfig.SubnetName;
            this.Subnet = ncConfig.Subnet;
            this.GatewayIP = ncConfig.GatewayIP;
            this.DnsServers = ncConfig.DnsServers;
            this.Version = ncConfig.Version;
            this.Policies = ncConfig.Policies ?? new List<string>();
            this.VlanID = ncConfig.VlanID;
            this.CnsVersion = ncConfig.CnsVersion;
            this.ContainerType = ncConfig.ContainerType;
            this.OrchestratorContext = ncConfig.OrchestratorContext;
            this.CnetAddressSpace = ncConfig.CnetAddressSpace;
            this.NetworkContainerState = ncConfig.NetworkContainerState;
            this.Status = ncConfig.Status;
            this.LocalIP = ncConfig.LocalIP;
            this.Tags = new List<Tag>(Tags);
            this.VnetVersion = ncConfig.VnetVersion;
            CnsCnetAddresses = new List<Cns.Data.IPSubnet>(ncConfig.CnsCnetAddresses);
        }

        public CreateOrUpdateNetworkContainerRequest GetCreateNMAgentNetworkContainerRequest()
        {
            return new CreateOrUpdateNetworkContainerRequest(
                this.NetworkID,
                this.IPV4Addresses.ToArray(),
                this.SubnetName,
                this.Version.ToString(),
                this.Policies.ToArray(),
                this.VlanID);
        }

        public void SetCnsCnetAddresses(
            List<Cns.Data.IPSubnet> cnsCnetAddresses)
        {
            CnsCnetAddresses.Clear();
            CnsCnetAddresses.AddRange(cnsCnetAddresses);
        }

        public void UpdateCnsVersion()
        {
            ++CnsVersion;
        }

        public CreateNetworkContainerRequest GetCreateCnsNetworkContainerRequest()
        {
            IPSubnet ipSubnet =
                NetworkManager.IPSubnet.Parse(this.Subnet);

            Cns.Data.IPSubnet ncIPSubnet = new Cns.Data.IPSubnet(
                this.IPV4Addresses[0].Split(NetworkManager.IPSubnet.PrefixDelimiter)[0],
                ipSubnet.Prefix);

            IPConfiguration ipConfiguration = new IPConfiguration(
                ncIPSubnet,
                this.DnsServers,
                this.GatewayIP);
            Cns.Data.IPSubnet ncLocalIPSubnet = new Cns.Data.IPSubnet(
                this.LocalIP.Split(IPSubnet.PrefixDelimiter)[0],
                LocalIPSubnetConstants.LocalIPSubnetPrefix);

            List<string> localIPSubnetDns = new List<string>();
            IPConfiguration localIPConfiguration = new IPConfiguration(
                ncLocalIPSubnet,
                localIPSubnetDns,
                LocalIPSubnetConstants.LocalIPGateway);
            MultiTenancyInfo multiTenancyInfo = new MultiTenancyInfo(
                Cns.Data.Constants.Vlan,
                this.VlanID);

            return new CreateNetworkContainerRequest(
                this.CnsVersion.ToString(),
                this.ContainerType,
                this.NetworkContainerID,
                this.NodeInterfaceID,
                "",
                ipConfiguration,
                localIPConfiguration,
                multiTenancyInfo,
                CnsCnetAddresses.ToArray(),
                null);
        }

        public override string ToString()
        {
            string tokenHashString;
            using (var algorithm = SHA256.Create())
            {
                byte[] tokenHash = algorithm.ComputeHash(Encoding.ASCII.GetBytes(this.AccessToken));
                tokenHashString = System.BitConverter.ToString(tokenHash).Replace("-", "").ToLower();
            }
            return $"[NetworkID:{this.NetworkID}, NetworkID:{this.NodeID} " +
                $"NetworkContainerID:{this.NetworkContainerID} , NodeInterfaceID:{this.NodeInterfaceID}" +
                $"AccessToken:{tokenHashString}, IPv4Addresses:{string.Join(",", this.IPV4Addresses)} " +
                $"SubnetName:{this.SubnetName} Version:{this.Version} Policies:" +
                $"{string.Join(",", this.Policies)}, VlanID:{this.VlanID} " +
                $"CnsVersion:{this.CnsVersion} ContainerType:{this.ContainerType}" +
                $"CnetAddressSpace:{this.CnetAddressSpace} " +
                $"NetworkContainerState:{NetworkContainerState} Status:{this.Status} LocalIP:{this.LocalIP} " +
                $"Tags:{Tags} VnetVersion:{VnetVersion}]";
        }

        public static string GetNetworkContainerKey(
            string networkID,
            string networkContainerID)
        {
            return string.Format(DataModel.Constants.NCKey, networkID, networkContainerID);
        }

        public object Clone()
        {
            return new NetworkContainerInfo(this);
        }
    }
}                    
