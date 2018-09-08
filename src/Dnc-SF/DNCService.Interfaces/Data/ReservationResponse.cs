//--------------------------------------------------------------------------
// <copyright file="ReservationResponse.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
// Definititon for IP ReservationResponse.
// </summary>
//--------------------------------------------------------------------------

using System.Runtime.Serialization;

namespace DNCService.API
{
    /// <summary>
    /// Calss that defines the reservation response.
    /// </summary>
    [DataContract]
    public class ReservationResponse
    {
        /// <summary>
        /// ID of the network that is associated with the reservation.
        /// </summary>
        [DataMember]
        public string NetworkID { get; private set; }

        /// <summary>
        /// ID of the reservation.
        /// </summary>
        [DataMember]
        public string ReservationID { get; private set; }

        /// <summary>
        /// Request associated with the reservation.
        /// </summary>
        [DataMember]
        public IPAllocationRequest AllocationRequest { get; private set; }

        /// <summary>
        /// The allocated subnet for the request.
        /// </summary>
        [DataMember]
        public string AllocatedIPSubnet { get; private set; }

        /// <summary>
        /// Current status of the request.
        /// </summary>
        [DataMember]
        public RequestStatus RequestStatus { get; private set; }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="networkID">Supply the ID of the network.</param>
        /// <param name="reservationID">Supply the id of the reservation.</param>
        /// <param name="allocationRequest">Supply the allocation request.</param>
        /// <param name="allocatedIPSubnet">Supply the allocated IPSubnet.</param>
        /// <param name="requestStatus">Supply the status of the reservation.</param>
        public ReservationResponse(
            string networkID, 
            string reservationID, 
            IPAllocationRequest allocationRequest, 
            string allocatedIPSubnet, RequestStatus requestStatus)
        {
            this.NetworkID = networkID;
            this.ReservationID = reservationID;
            this.AllocationRequest = allocationRequest;
            this.AllocatedIPSubnet = allocatedIPSubnet;
            this.RequestStatus = requestStatus;
        }

        /// <summary>
        /// ToString implementation.
        /// </summary>
        /// <returns>String representation of the ReservationRequest object.</returns>
        public override string ToString()
        {
            return $"NetworkID:{this.NetworkID} ReservationID:{this.ReservationID} AllocationRequest:{this.AllocationRequest} " +
                $"AllocatedIPSubnet:{this.AllocatedIPSubnet} RequestStatus:{this.RequestStatus}";
        }
    }
}
