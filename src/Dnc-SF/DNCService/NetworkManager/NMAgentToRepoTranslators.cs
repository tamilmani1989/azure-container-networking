
using NetworkManager.DataModel;
using NMAgent;
using NMAgent.Data;
using System;
using System.Collections.Generic;
using System.Linq;

namespace NetworkManager
{
    public static class NMAgentToRepoTranslators
    {
        public static SubnetConfiguration GetSubnetConfiguration(
            string networkID,
            Subnet nmagentContractSubnet)
        {
            List<Tag> tags = new List<Tag>();
            if(nmagentContractSubnet.Tags != null && nmagentContractSubnet.Tags.Any())
            {
                tags.AddRange(nmagentContractSubnet.Tags);
            }

            SubnetConfiguration subnetConfiguration = new SubnetConfiguration(
                networkID,
                nmagentContractSubnet.SubnetName,
                nmagentContractSubnet.AddressPrefix,
                true,
                nmagentContractSubnet.DefaultGateway,
                "",
                tags);

            return subnetConfiguration;
        }

        public static bool UpdateNetworkAndSubnetConfigurations(
            NetworkConfiguration networkConfiguration,
            Dictionary<string, SubnetConfiguration> subnetConfigurationsMap,
            VirtualNetworkResponse virtualNetworkResponse,
            out HashSet<string> removedKeys)
        {
            UInt64 receivedVnetVersion = UInt64.Parse(virtualNetworkResponse.VnetVersion);
            removedKeys = null;
            if (receivedVnetVersion == networkConfiguration.Version)
            {
                return false;
            }

            networkConfiguration.Version = UInt64.Parse(virtualNetworkResponse.VnetVersion);
            networkConfiguration.SetVNetSpace(virtualNetworkResponse.VNetSpace);
            networkConfiguration.SetCNetSpace(virtualNetworkResponse.CNetSpace);
            List<string> dnsServers = new List<string>();
            if (!string.IsNullOrWhiteSpace(virtualNetworkResponse.DNSServers))
            {
                dnsServers.AddRange(virtualNetworkResponse.DNSServers.Split(","));
            }

            networkConfiguration.SetDnsServers(dnsServers);

            Dictionary<string, NMAgent.Data.Subnet> updatedSubnetSet = 
                new Dictionary<string, NMAgent.Data.Subnet>();
            if (virtualNetworkResponse.Subnets!= null && virtualNetworkResponse.Subnets.Any())
            {
                foreach(NMAgent.Data.Subnet nmagentContractSubnet in virtualNetworkResponse.Subnets)
                {
                    updatedSubnetSet[SubnetConfiguration.GetSubnetConfigurationKey(
                        networkConfiguration.NetworkID,
                        nmagentContractSubnet.SubnetName)] = nmagentContractSubnet;
                }
            }

            IEnumerable<string> addedKeys = updatedSubnetSet.Keys.Except(subnetConfigurationsMap.Keys);
            removedKeys = new HashSet<string>(subnetConfigurationsMap.Keys.Except(updatedSubnetSet.Keys));
            IEnumerable<string> unChangedKeys = subnetConfigurationsMap.Keys.Intersect(updatedSubnetSet.Keys);

            foreach(string key in removedKeys)
            {
                SubnetConfiguration subnetConfiguration;
                if(subnetConfigurationsMap.TryGetValue(key, out subnetConfiguration))
                {
                    if(subnetConfiguration.Reservations.Count > 0)
                    {
                        throw new Exception($"Subnet {subnetConfiguration.SubnetName} cannot be removed as there are allocations in it. " +
                            $"Allocations:{string.Join(",", subnetConfiguration.Reservations)}");
                    }
                }
            }

            foreach (string key in unChangedKeys)
            {
                SubnetConfiguration subnetConfiguration = subnetConfigurationsMap[key];
                NMAgent.Data.Subnet nmAgentContractSubnet = updatedSubnetSet[key];
                IPSubnet updatedIPSubnet = IPSubnet.Parse(nmAgentContractSubnet.AddressPrefix);
                if (subnetConfiguration.Reservations.Count > 0)
                { 
                    if (!subnetConfiguration.IPSubnet.Equals(updatedIPSubnet))
                    {
                        throw new Exception(
                            $"Subnet {subnetConfiguration.SubnetName} cannot be updated as there are allocations in it. " +
                            $"Allocations:{string.Join(",", subnetConfiguration.Reservations)}. " +
                            $"Current IPSubnet:{subnetConfiguration.IPSubnet} UpdatedIPSubnet:{updatedIPSubnet}");
                    }
                }
            }

            foreach (string key in removedKeys)
            {
                subnetConfigurationsMap.Remove(key);
            }

            networkConfiguration.SetSubnets(updatedSubnetSet.ToDictionary(
                kvp => kvp.Value.SubnetName,
                kvp => kvp.Key));
            
            foreach(KeyValuePair<string, NMAgent.Data.Subnet> subnetKvp in updatedSubnetSet)
            {
                SubnetConfiguration updatedSubnetConfiguration = GetSubnetConfiguration(
                    networkConfiguration.NetworkID,
                    subnetKvp.Value);
                SubnetConfiguration curentSubnetConfiguration;
                if(subnetConfigurationsMap.TryGetValue(subnetKvp.Key, out curentSubnetConfiguration))
                {
                    updatedSubnetConfiguration.SetReservations(curentSubnetConfiguration.Reservations);
                }
                subnetConfigurationsMap[subnetKvp.Key] = updatedSubnetConfiguration;
            }

            return true;
        }

        public static TagDefinition GetTagDefinition(
            string tagName,
            string tagType,
            UInt64 version,
            NMAgent.Data.TagResponse tagResponse)
        {
            TagDefinition tagDefinition = new TagDefinition(
                tagResponse.Name,
                tagType);
            tagDefinition.UpdateTagDefinition(version, tagResponse.Values);

            return tagDefinition;
        }
    }
}
