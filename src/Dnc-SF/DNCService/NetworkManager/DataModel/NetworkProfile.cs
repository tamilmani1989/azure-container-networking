using DNCService.API;
using System;
using System.Collections.Generic;
using System.Collections.Immutable;
using System.Linq;
using System.Runtime.Serialization;
using System.Text;

namespace NetworkManager.DataModel
{
    [DataContract]
    public class NetworkProfile : ICloneable
    {
        [DataMember]
        public string NetworkProfileID { get; set; }

        public override string ToString()
        {
            return $"networkProfileID:{this.NetworkProfileID}";
        }

        public NetworkProfile(string networkProfileID)
        {
            this.NetworkProfileID = networkProfileID;
        }

        public NetworkProfile(NetworkProfile networkProfile)
        {
            this.NetworkProfileID = networkProfile.NetworkProfileID;
        }

        public object Clone()
        {
            return new NetworkProfile(this);
        }
    }
}
