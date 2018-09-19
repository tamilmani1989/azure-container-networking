using System;
using System.Collections.Generic;
using System.Runtime.Serialization;
using DNCService.API;

namespace NetworkManager.DataModel
{
    [DataContract]
    public class NetworkConfiguration : ICloneable
    {
        [DataMember]
        public string NetworkID { get; private set; }

        [DataMember]
        public string SubscriptionId { get; set; }

        [DataMember]
        public UInt64 Version { get; set; }

        [DataMember]
        public List<string> CnetSpace { get; private set; }

        [DataMember]
        public List<string> VnetSpace { get; private set; }

        [DataMember]
        private Dictionary<string, string> subnets;

        [DataMember]
        private HashSet<string> dnsServers;

        [DataMember]
        public NetworkState State { get; set; }

        [DataMember]
        public Status Status { get; set; }

        [DataMember]
        public string ApiVersion { get; private set; }

        [DataMember]
        public NetworkRequest NetworkInfoRequest { get; private set; }

        [DataMember]
        public ErrorInfo FailureDetail{ get; set; }

        public IReadOnlyDictionary<string, string> Subnets {
            get
            {
                return subnets;
            }
        }

        public IReadOnlyCollection<string> DnsServers
        {
            get
            {
                return this.dnsServers;
            }
        }

        public void AddOrUpdateSubnet(string name, string value)
        {
            if (string.IsNullOrWhiteSpace(name))
            {
                throw new ArgumentException("Cannot be null or empty", nameof(name));
            }

            if (string.IsNullOrWhiteSpace(value))
            {
                throw new ArgumentException("Cannot be null or empty", nameof(value));
            }

            subnets[name] = value;
        }

        public void RemoveSubnet(string name)
        {
            if (string.IsNullOrWhiteSpace(name))
            {
                throw new ArgumentException("Cannot be null or empty", nameof(name));
            }

            subnets.Remove(name);
        }

        public void SetSubnets(IReadOnlyDictionary<string, string> subnetKvps)
        {
            if (subnetKvps == null)
            {
                throw new ArgumentNullException(nameof(subnetKvps));
            }

            subnets.Clear();
            foreach(KeyValuePair<string, string> subnetKvp in subnetKvps)
            {
                subnets[subnetKvp.Key] = subnetKvp.Value;
            }
        }


        public void AddDnsServer(string dnsServer)
        {
            if (string.IsNullOrEmpty(dnsServer))
            {
                throw new ArgumentException("Cannot be null or empty", nameof(dnsServer));
            }

            dnsServers.Add(dnsServer);
        }

        public void RemoveDnsServer(string dnsServer)
        {
            if (dnsServer == null) if (string.IsNullOrEmpty(dnsServer))
                {
                throw new ArgumentException("Cannot be null or empty", nameof(dnsServer));
            }

            dnsServers.Remove(dnsServer);
        }

        public void SetDnsServers(IReadOnlyCollection<string> dnsServers)
        {
            if (dnsServers == null)
            {
                throw new ArgumentNullException(nameof(dnsServers));
            }

            this.dnsServers.Clear();
            foreach(string dnsServer in dnsServers)
            {
                this.dnsServers.Add(dnsServer);
            }
        }

        public NetworkConfiguration(
            string subscriptionId,
            string networkId,
            NetworkRequest networkInfoRequest,
            string apiVersion)
        {
            this.NetworkID = networkId;
            this.SubscriptionId = subscriptionId;
            this.NetworkInfoRequest = networkInfoRequest;
            this.subnets = new Dictionary<string, string>();
            this.dnsServers = new HashSet<string>();
            this.CnetSpace = new List<string>();
            this.VnetSpace = new List<string>();
            this.State = NetworkState.Unknown;
            this.Status = Status.Pending;
            this.ApiVersion = apiVersion;
            this.FailureDetail = null;
        }

        public NetworkConfiguration(NetworkConfiguration networkConfiguration)
        {
            this.NetworkID = networkConfiguration.NetworkID;
            this.SubscriptionId = networkConfiguration.SubscriptionId;
            this.NetworkInfoRequest = (NetworkRequest)networkConfiguration.NetworkInfoRequest.Clone();
            this.subnets = new Dictionary<string, string>(networkConfiguration.subnets);
            this.dnsServers = new HashSet<string>(networkConfiguration.dnsServers);
            this.CnetSpace = new List<string>(networkConfiguration.CnetSpace);
            this.VnetSpace = new List<string>(networkConfiguration.VnetSpace);
            this.State = networkConfiguration.State;
            this.Status = networkConfiguration.Status;
            this.ApiVersion = networkConfiguration.ApiVersion;
            this.FailureDetail = networkConfiguration.FailureDetail;
        }

        public void SetCNetSpace(string cnetSpace)
        {
            if (string.IsNullOrEmpty(cnetSpace))
            {
                throw new Exception($"CNetspace cannot be null or empty.");
            }
            this.CnetSpace.Clear();
            this.CnetSpace.AddRange(cnetSpace.Split(","));
        }

        public void SetVNetSpace(string vnetSpace)
        {
            if (string.IsNullOrEmpty(vnetSpace))
            {
                throw new Exception($"VNetspace cannot be null or empty.");
            }
            this.VnetSpace.Clear();
            this.VnetSpace.AddRange(vnetSpace.Split(","));
        }

        public override string ToString()
        {
            return $"[NetworkId:{this.NetworkID} SubscriptionId:{this.SubscriptionId} " +
                $"CNetSpace:{string.Join(",", this.CnetSpace)} VnetSpace:{string.Join(",", this.VnetSpace)} " +
                $"Subnets:{string.Join(",", this.subnets)} " +
                $"DnsServers:{string.Join(",", this.dnsServers)}, State:{this.State}," +
                $"Status:{this.Status}, NetworkInfoRequest:{this.NetworkInfoRequest} " +
                $"ApiVersion:{this.ApiVersion} FailureDetail:{this.FailureDetail}";
        }

        public static string GetNetworkKey(string networkID)
        {
            return string.Format(Constants.NetworkKey, networkID);
        }

        public static string GetNetworkIDFromKey(string networkKey)
        {
            return networkKey.Split('_')[1];
        }

        public override bool Equals(object obj)
        {
            if (obj == null || GetType() != obj.GetType())
                return false;
            NetworkConfiguration networkConfiguration = obj as NetworkConfiguration;

            return this.NetworkID == networkConfiguration.NetworkID;
        }

        public override int GetHashCode()
        {
            return this.NetworkID.GetHashCode();
        }

        public object Clone()
        {
            return new NetworkConfiguration(this);
        }
    }
}
