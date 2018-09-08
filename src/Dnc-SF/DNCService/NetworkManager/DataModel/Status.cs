using System.Runtime.Serialization;

namespace NetworkManager.DataModel
{
    [DataContract]
    public enum Status
    {
        [EnumMember]
        NotStarted = 0,

        [EnumMember]
        Pending = 1,

        [EnumMember]
        Failed = 2,

        [EnumMember]
        Denied = 3,

        [EnumMember]
        Completed = 4,
    }
}
