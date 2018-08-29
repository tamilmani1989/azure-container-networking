//--------------------------------------------------------------------------
// <copyright file="IPAllocationResponse.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
// Definititon for IP allocation response.
// </summary>
//--------------------------------------------------------------------------

using System.Runtime.Serialization;

namespace DNCService.API
{
    /// <summary>
    /// Defines the class for the IPAllocation response.
    /// </summary>
    [DataContract]
    public class IPAllocationResponse
    {
        /// <summary>
        /// Allocated IP Address.
        /// </summary>
        [DataMember]
        public string IPAddress { get; private set; }

        /// <summary>
        /// Optional. Associated network container ID.
        /// </summary>
        [DataMember]
        public string AssociatedNetworkContainerID { get; private set; }

        /// <summary>
        /// Associtaed reservation ID.
        /// </summary>
        [DataMember]
        public string ReservationID { get; private set; }

        /// <summary>
        /// Default Constructor.
        /// </summary>
        public IPAllocationResponse() { }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="ipAddress">Supply the IP Address.</param>
        /// <param name="ncID">Supply the associated network container ID.</param>
        /// <param name="reservationId">Supply the reservation ID.</param>
        public IPAllocationResponse(string ipAddress, string ncID, string reservationId)
        {
            this.IPAddress = ipAddress;
            this.AssociatedNetworkContainerID = ncID;
            this.ReservationID = reservationId;
        }

        /// <summary>
        /// ToString implementation.
        /// </summary>
        /// <returns>String representation of the IPAllocationResponse object.</returns>        
        public override string ToString()
        {
            return $"[IPAddress:{this.IPAddress}, AssociatedNCID:{this.AssociatedNetworkContainerID}, " +
                $"ReservationID:{this.ReservationID}]";
        }
    }
}
