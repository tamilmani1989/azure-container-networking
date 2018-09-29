using System;
using System.Collections.Generic;
using System.Collections.Immutable;
using System.Linq;
using System.Runtime.Serialization;
using System.Text;
using DNCService.API;

namespace NetworkManager.DataModel
{
    [DataContract]
    public class NodeConfiguration : ICloneable
    {
        [DataMember]
        public string NodeID { get; private set; }

        [DataMember]
        public string APIVersion { get; private set; }

        [DataMember]
        public NodeRequest NodeInfo { get; set; }

        [DataMember]
        private Dictionary<string, Dictionary<string, string>> deployedNetworkContainerMap;

        [DataMember]
        public Dictionary<string, string> constrainedReservations;

        [DataMember]
        public Dictionary<string, UInt16> networkVlanIDMap;

        public IReadOnlyDictionary<string, IReadOnlyDictionary<string, string>> DeployedNCMap
        {
            get
            {
                return deployedNetworkContainerMap.ToDictionary(
                    kvp => kvp.Key,
                    kvp => (IReadOnlyDictionary<string, string>)kvp.Value);
            }
        }

        public IReadOnlyDictionary<string, string> ConstrainedReservations
        {
            get
            {
                return constrainedReservations;
            }
        }

        public IReadOnlyDictionary<string, ushort> NetworkVlanIDMap
        {
            get
            {
                return networkVlanIDMap;
            }
        }

        public NodeConfiguration(string nodeId, string apiVersion, NodeRequest nodeInfo)
        {
            this.NodeID = nodeId;
            this.APIVersion = apiVersion;
            this.NodeInfo = nodeInfo;
            this.deployedNetworkContainerMap = new Dictionary<string, Dictionary<string, string>>();
            this.networkVlanIDMap = new Dictionary<string, UInt16>();
            this.constrainedReservations = new Dictionary<string, string>();
        }

        public NodeConfiguration(NodeConfiguration nodeConfiguration)
        {
            this.NodeID = nodeConfiguration.NodeID;
            this.APIVersion = nodeConfiguration.APIVersion;
            this.NodeInfo = (NodeRequest)nodeConfiguration.NodeInfo.Clone();
            this.deployedNetworkContainerMap = new Dictionary<string, Dictionary<string, string>>(nodeConfiguration.deployedNetworkContainerMap);
            this.networkVlanIDMap = new Dictionary<string, UInt16>(nodeConfiguration.networkVlanIDMap);
            this.constrainedReservations = new Dictionary<string, string>(nodeConfiguration.constrainedReservations);
        }

        public void AddNetworkContainer(string networkID, string ncID)
        {
            Dictionary<string, string> networkMap;
            if(!this.deployedNetworkContainerMap.TryGetValue(networkID, out networkMap))
            {
                networkMap = new Dictionary<string, string>();
                this.deployedNetworkContainerMap[networkID] = networkMap;
            }
            networkMap[ncID] = NCConfiguration.GetNCKey(networkID, ncID);
            AddVlanID(networkID);
        }

        public void AddVlanID(string networkID)
        {
            if (!this.networkVlanIDMap.ContainsKey(networkID))
            {
                HashSet<ushort> allocatedVlanIDs = this.NetworkVlanIDMap.Values.ToHashSet();
                this.networkVlanIDMap[networkID] = GetNextVlanID(
                    networkID,
                    allocatedVlanIDs);
            }
        }

        private ushort GetNextVlanID(
            string networkID,
            HashSet<ushort> usedVlanIds)
        {
            IEnumerable<ushort> vlanRange = Enumerable.Range(
                DNCService.DNCService.DncConfigurationInstance.VlanSettings.Min,
                DNCService.DNCService.DncConfigurationInstance.VlanSettings.Max).Select(i => (ushort)i);
            foreach (ushort vlanID in vlanRange)
            {
                if (!usedVlanIds.Contains(vlanID))
                {
                    return vlanID;
                }
            }

            throw new DncNodeManagementException(
                NodeID,
                new ErrorInfo(ErrorCode.OutOfVlanIDs, "No free vlan ID's available on the node",
                ErrorType.Retriable));

        }
        public void RemoveNetworkContainer(string networkID, string ncID)
        {
            Dictionary<string, string> networkMap;
            if (this.deployedNetworkContainerMap.TryGetValue(networkID, out networkMap))
            {
                networkMap.Remove(ncID);
                if(networkMap.Count == 0)
                {
                    this.deployedNetworkContainerMap.Remove(networkID);
                    this.networkVlanIDMap.Remove(networkID);
                }
            }
        }

        public override string ToString()
        {
            string networkVlanMap = string.Join(
                ",",
                this.networkVlanIDMap.Select(kvp => string.Format("{0}:{1}", kvp.Key, kvp.Value)));

            string constrainedReservations = string.Join(
                ",",
                this.constrainedReservations.Select(kvp => string.Format("{0}:{1}", kvp.Key, kvp.Value)));

            string deployedNetworkContainers = string.Join(
                ",",
                this.deployedNetworkContainerMap.Select(ncKvp => 
                    string.Format("[NetworkId:{0}: NCs:{1}]", 
                        ncKvp.Key, 
                        string.Join(",", ncKvp.Value.Select(nc => 
                            string.Format("{0}:{1}",
                                nc.Key,
                                nc.Value))))));

            return $"Id:{this.NodeID} Info:{this.NodeInfo} NetworkVlanIDMap:[{string.Join(",", this.networkVlanIDMap)}] " +
                $"ConstrainedReservations:[{constrainedReservations}] DeployedNetworkContainers:[{deployedNetworkContainers}] " +
                $"NetworkVlanMap:[{networkVlanMap}]";
        }

        public static string GetNodeKey(string nodeID)
        {
            return string.Format(Constants.NodeKey, nodeID);
        }

        public override bool Equals(object obj)
        {
            if (obj == null || GetType() != obj.GetType())
                return false;
            NodeConfiguration nodeConfiguration = obj as NodeConfiguration;

            return this.NodeID == nodeConfiguration.NodeID;
        }

        public override int GetHashCode()
        {
            return this.NodeID.GetHashCode();
        }

        public object Clone()
        {
            return new NodeConfiguration(this);
        }
    }
}
