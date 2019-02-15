//
//    Copyright (C) Microsoft.  All rights reserved.
//

using System;
using System.Runtime.Serialization;

namespace DNCService.API
{
    /// <summary>
    /// Class defintition for DncNetworkReservationException.
    /// </summary>
    [Serializable]
    [DataContract]
    public class DncNetworkReservationException : DncManagementException, ISerializable
    {
        /// <summary>
        /// ID of the network.
        /// </summary>
        [DataMember]
        public string NetworkID { get; private set; }

        /// <summary>
        /// Reservation ID.
        /// </summary>
        [DataMember]
        public string ReservationID { get; private set; }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="networkID">Supply the ID of the network.</param>
        /// <param name="errorInfo">Supply the error Info to be associated with th exception.</param>
        public DncNetworkReservationException(string networkID, string reservationID,  ErrorInfo errorInfo) : base(errorInfo)
        {
            this.NetworkID = networkID;
            this.ReservationID = reservationID;
        }

        /// <summary>
        /// ToString implementation.
        /// </summary>
        /// <returns>ToString representation of the DncNetworkManagementException object.</returns>
        public override string ToString()
        {
            return $"Network:{this.NetworkID} ReservationID:{this.ReservationID} Error:{base.ToString()}";
        }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="info">Supply the serialization info.</param>
        /// <param name="context">Supply the streaming context.</param>
        protected DncNetworkReservationException(SerializationInfo info, StreamingContext context)
                   : base(info, context)
        {
            this.NetworkID = info.GetString(nameof(this.NetworkID));
            this.ReservationID = info.GetString(nameof(this.ReservationID));
        }

        /// Implements the get object method used for serialization.
        /// </summary>
        /// <param name="info">Supply the serialization info.</param>
        /// <param name="context">Supply the streaming context.</param>
        public override void GetObjectData(SerializationInfo info, StreamingContext context)
        {
            if (info == null)
            {
                throw new ArgumentNullException("info");
            }

            info.AddValue(nameof(this.NetworkID), this.NetworkID);
            info.AddValue(nameof(this.ReservationID), this.ReservationID);

            // MUST call through to the base class to let it save its own state
            base.GetObjectData(info, context);
        }
    }
}
