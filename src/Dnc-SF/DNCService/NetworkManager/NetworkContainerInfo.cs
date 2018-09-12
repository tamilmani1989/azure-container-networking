using Cns.Data;
using DNCService;
using NetworkManager.DataModel;
using Newtonsoft.Json.Linq;
using NMAgent;
using NMAgent.Data;
using System.Collections.Generic;
using System.Security.Cryptography;
using System.Text;

namespace NetworkManager
{
    public class NetworkContainerInfo
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
            Status  status)
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
            ServiceEventSource.Current.Message($"DnsServers:{string.Join(",", DnsServers)}");
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

        public CreateNetworkContainerRequest GetCreateCnsNetworkContainerRequest()
        {
            IPSubnet ipSubnet =
                NetworkManager.IPSubnet.Parse(this.Subnet);

            Cns.Data.IPSubnet ncIPSubnet = new Cns.Data.IPSubnet(
                this.IPV4Addresses[0].Split(NetworkManager.IPSubnet.PrefixDelimiter)[0],
                ipSubnet.Prefix);

            ServiceEventSource.Current.Message($"CreateNetworkContainerRequest:DnsServers:{string.Join(",", DnsServers)}");
            IPConfiguration ipConfiguration = new IPConfiguration(
                ncIPSubnet,
                this.DnsServers,
                this.GatewayIP);

            MultiTenancyInfo multiTenancyInfo = new MultiTenancyInfo(
                Cns.Data.Constants.Vlan,
                this.VlanID);

            return new CreateNetworkContainerRequest(
                this.CnsVersion.ToString(),
                this.ContainerType,
                this.NetworkContainerID,
                this.NodeInterfaceID,
                this.AccessToken,
                ipConfiguration,
                multiTenancyInfo,
                GetIPSubnets(this.CnetAddressSpace),
                null);
        }

        private Cns.Data.IPSubnet[] GetIPSubnets(List<string> ipSubnetStrings)
        {
            List<Cns.Data.IPSubnet> ipSubnets = new List<Cns.Data.IPSubnet>();
            foreach(string ipSubnetString in ipSubnetStrings)
            {
                IPSubnet vnetIPSubnet = IPSubnet.Parse(ipSubnetString);

                Cns.Data.IPSubnet ipSubnet = new Cns.Data.IPSubnet(
                    vnetIPSubnet.IPAddress.ToString(),
                    vnetIPSubnet.Prefix);
                ipSubnets.Add(ipSubnet);
            }

            return ipSubnets.ToArray();
        }

        public override string ToString()
        {
            string tokenHashString;
            using (var algorithm = SHA256.Create())
            {
                byte[] tokenHash = algorithm.ComputeHash(Encoding.ASCII.GetBytes(this.AccessToken));
                tokenHashString = System.Text.Encoding.Default.GetString(tokenHash);
            }
            return $"[NetworkID:{this.NetworkID}, NetworkID:{this.NodeID} " +
                $"NetworkContainerID:{this.NetworkContainerID} , NodeInterfaceID:{this.NodeInterfaceID}" +
                $"AccessToken:{tokenHashString}, IPv4Addresses:{string.Join(",", this.IPV4Addresses)} " +
                $"SubnetName:{this.SubnetName} Version:{this.Version} Policies:" +
                $"{string.Join(",", this.Policies)}, VlanID:{this.VlanID} " +
                $"CnsVersion:{this.CnsVersion} ContainerType:{this.ContainerType}" +
                $"CnetAddressSpace:{this.CnetAddressSpace} " +
                $"NetworkContainerState:{NetworkContainerState} Status:{this.Status}]";
        }
    }
}                    
