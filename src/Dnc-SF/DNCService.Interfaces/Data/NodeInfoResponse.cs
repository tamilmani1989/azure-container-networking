using System;
using System.Collections.Generic;
using System.Text;
using System.Linq;
using System.Runtime.Serialization;

namespace DNCService.API
{
    [DataContract]
    public class NodeInfoResponse
    {
        [DataMember]
        public NodeInfoRequest NodeInfo { get; private set; }

        [DataMember]
        public NetworkContainerResponse[] NetworkContainers { get; private set; }

        [DataMember]
        public ReservationResponse[] ConstrainedReservations { get; private set; }

        public NodeInfoResponse() { }
        public NodeInfoResponse(NodeInfoRequest nodeInfo, NetworkContainerResponse[] networkContainerResponses, ReservationResponse[] reservationResponses)
        {
            this.NodeInfo = nodeInfo;
            this.NetworkContainers = networkContainerResponses;
            this.ConstrainedReservations = reservationResponses;
        }

        public override string ToString()
        {
            string networkContainers = this.NetworkContainers != null ? string.Join(",", (object[])this.NetworkContainers) : "null";
            string constrainedReservations = this.ConstrainedReservations != null ? string.Join(",", (object[])this.ConstrainedReservations) : "null";
            return $"NodeInfo:{this.NodeInfo} NetworkContainers:{networkContainers} ConstrainedReservations:{constrainedReservations}";
        }

    }
}
