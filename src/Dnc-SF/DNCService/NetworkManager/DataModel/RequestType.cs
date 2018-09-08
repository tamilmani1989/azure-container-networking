using System;
using System.Collections.Generic;
using System.Runtime.Serialization;
using System.Text;

namespace NetworkManager.DataModel
{
    [DataContract]
    public enum RequestType
    {
        [EnumMember]
        InvalidRequest = 0,

        [EnumMember]
        ReservationRequest = 1,

        [EnumMember]
        NetworkContainerRequest = 2,

        [EnumMember]
        VirtualNetworkRequest = 3,
    }
}
