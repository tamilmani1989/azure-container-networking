using System.Runtime.Serialization;
using System.Text;

namespace NetworkManager.DataModel
{
    [DataContract]
    public enum ReservationState
    {
        [EnumMember]
        Process = 0,

        [EnumMember]
        Allocate = 1,

        [EnumMember]
        Release = 2,
    }
}
