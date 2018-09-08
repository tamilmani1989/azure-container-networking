//--------------------------------------------------------------------------
// <copyright file="NodeResponse.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
// Definititon for IP NodeResponse class.
// </summary>
//--------------------------------------------------------------------------

using System.Runtime.Serialization;

namespace DNCService.API
{
    /// <summary>
    /// Class definition for node response.
    /// </summary>
    [DataContract]
    public class NodeResponse
    {
        /// <summary>
        /// Information related to the node.
        /// </summary>
        [DataMember]
        public NodeRequest NodeInfo { get; private set; }

        /// <summary>
        /// List fo network containers information associated with the node.
        /// </summary>
        [DataMember]
        public NetworkContainerResponse[] NetworkContainers { get; private set; }

        /// <summary>
        /// List of constrained reservations.
        /// </summary>
        [DataMember]
        public ReservationResponse[] ConstrainedReservations { get; private set; }

        /// <summary>
        /// Default constructor.
        /// </summary>
        public NodeResponse() { }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="nodeInfo">Supply the node info.</param>
        /// <param name="networkContainerResponses">Suppply the NC's info deplyed on the node.</param>
        /// <param name="reservationResponses">Supply the reservations constrained to the node.</param>
        public NodeResponse(NodeRequest nodeInfo, NetworkContainerResponse[] networkContainerResponses, ReservationResponse[] reservationResponses)
        {
            this.NodeInfo = nodeInfo;
            this.NetworkContainers = networkContainerResponses;
            this.ConstrainedReservations = reservationResponses;
        }

        /// <summary>
        /// ToString implementation.
        /// </summary>
        /// <returns>String representation of the NodeInfoResponse object.</returns>
        public override string ToString()
        {
            string networkContainers = this.NetworkContainers != null ? string.Join(",", (object[])this.NetworkContainers) : "null";
            string constrainedReservations = this.ConstrainedReservations != null ? string.Join(",", (object[])this.ConstrainedReservations) : "null";
            return $"NodeInfo:{this.NodeInfo} NetworkContainers:{networkContainers} ConstrainedReservations:{constrainedReservations}";
        }

    }
}
