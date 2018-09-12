using System;
using System.Collections.Generic;
using System.Runtime.Serialization;
using System.Text;

namespace NetworkManager.DataModel
{
    [DataContract]    
    public enum NetworkContainerState
    {
        [EnumMember]
        Unused = 1,

        [EnumMember]
        Prepare = 1,

        [EnumMember]
        Reserve = 2,

        [EnumMember]
        Associate = 3,

        [EnumMember]
        Publish = 4,

        [EnumMember]
        Program = 5,

        [EnumMember]
        UnPublish = 6,

        [EnumMember]
        Revert = 7,

        [EnumMember]
        DisAssociate = 8,

        [EnumMember]
        UnReserve = 9,

        [EnumMember]
        Purge = 10,

        [EnumMember]
        RefreshAssociation = 11,

        [EnumMember]
        AllocateLocalIP = 12,
    }
}
