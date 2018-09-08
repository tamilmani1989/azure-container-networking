using System;
using System.Collections.Generic;
using System.Runtime.Serialization;
using System.Text;

namespace DNCService.API
{
    [DataContract]
    public class IPAllocationData
    {
        [DataMember]
        public string IPAddress { get; private set; }
        [DataMember]
        public string AssociatedNetworkContainerID { get; private set; }
        [DataMember]
        public string ReservationID { get; private set; }

        public IPAllocationData() { }
        public IPAllocationData(string ipAddress, string ncID, string reservationId)
        {
            this.IPAddress = ipAddress;
            this.AssociatedNetworkContainerID = ncID;
            this.ReservationID = reservationId;
        }

        public override string ToString()
        {
            return $"[IPAddress:{this.IPAddress}, AssociatedNCID:{this.AssociatedNetworkContainerID}, " +
                $"ReservationID:{this.ReservationID}]";
        }
    }
}
