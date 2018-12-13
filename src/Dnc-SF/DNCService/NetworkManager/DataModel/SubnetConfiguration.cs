using System.Collections.Generic;
using System.Runtime.Serialization;
using System.Collections.Immutable;
using System;
using NMAgent.Data;
using DNCService;

namespace NetworkManager.DataModel
{
    [DataContract]
    public class SubnetConfiguration : ICloneable
    {
        [DataMember]
        public string NetworkID { get; private set; }

        [DataMember]
        private Dictionary<string, string> reservations;

        [DataMember]
        public string SubnetName { get; private set; }

        [DataMember]
        public string IPSubnet { get; private set; }

        [DataMember]
        public bool IsDelegatedSubnet { get; private set; }

        [DataMember]
        public string Gateway { get; private set; }

        [DataMember]
        public string AccessToken { get; set; }

        [DataMember]
        public bool IsJoined { get; set; }

        [DataMember]
        private HashSet<Tag> tags;

        [DataMember]
        public string APIVersion { get; private set; }

        public SubnetConfiguration(
            string networkID,
            string subnetName,
            string ipSubnet,
            bool isDelegatedSubnet,
            bool isJoined,
            string gateway,
            string accessToken,
            ICollection<Tag> tagSet,
            IReadOnlyDictionary<string, string> reservations)
        {
            this.NetworkID = networkID;
            this.SubnetName = subnetName;
            this.IPSubnet = ipSubnet;
            this.IsDelegatedSubnet = isDelegatedSubnet;
            this.Gateway = gateway;
            this.AccessToken = accessToken;
            this.IsJoined = isJoined;
            this.tags = new HashSet<Tag>();
            this.reservations = new Dictionary<string, string>(reservations);
            if (tagSet != null)
            {
                this.tags.UnionWith(tagSet);
            }
        }

        public SubnetConfiguration(
            SubnetConfiguration subnetConfiguration)
        {
            this.NetworkID = subnetConfiguration.NetworkID;
            this.SubnetName = subnetConfiguration.SubnetName;
            this.IPSubnet = subnetConfiguration.IPSubnet;
            this.IsDelegatedSubnet = subnetConfiguration.IsDelegatedSubnet;
            this.Gateway = subnetConfiguration.Gateway;
            this.AccessToken = subnetConfiguration.AccessToken;
            this.tags = new HashSet<Tag>(subnetConfiguration.tags);
            this.reservations = new Dictionary<string, string>(subnetConfiguration.reservations);
            this.IsJoined = subnetConfiguration.IsJoined;
        }

        public IReadOnlyCollection<Tag> Tags 
        {
            get
            {
                return tags.ToImmutableHashSet<Tag>();
            }            
        }

        public IReadOnlyDictionary<string, string> Reservations
        {
            get
            {
                return this.reservations;
            }
        }
            
        public void SetReservations(IReadOnlyDictionary<string, string> reservations)
        {
            this.reservations.Clear();

            foreach(KeyValuePair<string, string> reservationsKvp in reservations)
            {
                this.reservations[reservationsKvp.Key] = reservationsKvp.Value;
            }
        }

        public void AddReservation(string reservationID, string ipAddress)
        {
            this.reservations[reservationID] = ipAddress;
        }

        public void RemoveReservation(string reservationID)
        {
            this.reservations.Remove(reservationID);
        }

        public static string GetSubnetConfigurationKey(string networkID, string subnetID)
        {
            return string.Format(Constants.SubnetConfigurationKey, networkID, subnetID);
        }
        
        public override bool Equals(object obj)
        {
            if (obj == null || GetType() != obj.GetType())
                return false;
            SubnetConfiguration subnetConfiguration = obj as SubnetConfiguration;

            return this.NetworkID == subnetConfiguration.NetworkID && this.SubnetName == subnetConfiguration.SubnetName;
        }

        public override int GetHashCode()
        {
            return GetSubnetConfigurationKey(this.NetworkID, this.SubnetName).GetHashCode();
        }

        public object Clone()
        {
            return new SubnetConfiguration(this);
        }
    }
}
