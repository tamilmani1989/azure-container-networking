//--------------------------------------------------------------------------
// <copyright file="NetworkContainerRequestV1.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
// Definition for NetworkContainerRequestV1 class.
// Uses an existing reservation.
// </summary>
//--------------------------------------------------------------------------

using System.Runtime.Serialization;

namespace DNCService.API
{
    /// <summary>
    /// Defintition for the network container request.
    /// </summary>
    [DataContract]
    public class NetworkContainerRequestV1 : NetworkContainerRequest
    {
        /// <summary>
        /// ID of the reservation to be used with the container.
        /// </summary>
        [DataMember]
        public string ReservationID { get; private set; }

        /// <summary>
        /// Default Constructor.
        /// </summary>
        public NetworkContainerRequestV1() { }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="reservationID">Supply the ID of the reservation.</param>
        /// <param name="associationInfo">Supply the association info.</param>
        public NetworkContainerRequestV1(
            string networkProfileID,
            string reservationID,
            AssociationInfo associationInfo) : base(associationInfo, networkProfileID)
        {
            this.ReservationID = reservationID;
        }

        /// <summary>
        /// Copy constructor.
        /// </summary>
        /// <param name="ncRequest">Supply the request object to be copied.</param>
        public NetworkContainerRequestV1(NetworkContainerRequestV1 ncRequest) : base(ncRequest.AssociationInfo, ncRequest.NetworkProfileID)
        {
            this.ReservationID = ncRequest.ReservationID;
        }

        /// <summary>
        /// ToString implementation.
        /// </summary>
        /// <returns>String representation of the NetworkContainerRequestV1 object.</returns>
        public override string ToString()
        {
            return $"[{base.ToString()} ReservationID:{this.ReservationID}]";
        }

        /// <summary>
        /// Clone implementation.
        /// </summary>
        /// <returns>Cloned network request object.</returns>
        public override object Clone()
        {
            return new NetworkContainerRequestV1(this);
        }

    }
}
