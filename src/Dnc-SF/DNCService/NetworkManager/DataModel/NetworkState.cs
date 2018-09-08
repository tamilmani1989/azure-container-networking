using System;
using System.Collections.Generic;
using System.Runtime.Serialization;
using System.Text;

namespace NetworkManager.DataModel
{
    [DataContract]
    public enum NetworkState
    {
        [EnumMember]
        Unknown = 0,

        [EnumMember]
        Join = 1,

        [EnumMember]
        UnJoin = 2,
    }
}
