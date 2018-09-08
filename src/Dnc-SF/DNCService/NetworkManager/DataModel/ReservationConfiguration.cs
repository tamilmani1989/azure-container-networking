using DNCService.API;
using System;
using System.Collections.Generic;
using System.Runtime.Serialization;
using System.Text;

namespace NetworkManager.DataModel
{
    [DataContract]
    public class ReservationConfiguration : ICloneable
    {
        [DataMember]
        public string ReservationID { get; private set; }

        [DataMember]
        public string NetworkID { get; private set; }

        [DataMember]
        public IPAllocationRequest AllocationRequest { get; private set; }

        [DataMember]
        public string CreatedDateTime { get; private set; }

        [DataMember]
        public ReservationState ReservationState { get; set; }

        [DataMember]
        public Status Status { get; set; }

        [DataMember]
        public string AllocatedSubnet { get; set; }

        [DataMember]
        public ErrorInfo FailureDetail { get; set; }

        [DataMember]
        public string LastUpdatedDateTime { get; set; }

        [DataMember]
        public string AssociatedNCID { get; set; }

        [DataMember]
        public bool MarkedForDelete { get; set; }

        [DataMember]
        public string APIVersion { get; set; }


        public ReservationConfiguration(string networkID, string reservationID, IPAllocationRequest allocationRequest)
        {
            this.NetworkID = networkID;
            this.ReservationID = reservationID;
            this.AllocationRequest = (IPAllocationRequest)allocationRequest.Clone();
        }

        public override string ToString()
        {
            StringBuilder sb = new StringBuilder();
            sb.Append($"NetworkID:{this.NetworkID} ");
            sb.Append($"ReservationID:{this.ReservationID} ");
            sb.Append($"AllocationRequest:{this.AllocationRequest} ");
            sb.Append($"CreatedDateTime:{this.CreatedDateTime} ");
            sb.Append($"ReservationState:{this.ReservationState} ");
            sb.Append($"Status:{this.Status} ");
            sb.Append($"AllocatedSubnet:{this.AllocatedSubnet} ");
            sb.Append($"FailureDetail:{this.FailureDetail} ");
            sb.Append($"LastUpdatedDateTime:{this.LastUpdatedDateTime} ");
            sb.Append($"AssociatedNCID:{this.AssociatedNCID} ");
            sb.Append($"MarkedForDelete:{this.MarkedForDelete} ");
            sb.Append($"APIVersion:{this.APIVersion}");

            return sb.ToString();
        }

        public static string GetReservationKey(string networkID, string reservationID)
        {
            return string.Format(Constants.IPReservationKey, networkID, reservationID);
        }

        public override bool Equals(object obj)
        {
            if (obj == null || GetType() != obj.GetType())
                return false;
            ReservationConfiguration reservationConfiguration = obj as ReservationConfiguration;

            return this.NetworkID == reservationConfiguration.NetworkID && this.ReservationID == reservationConfiguration.ReservationID;
        }

        public override int GetHashCode()
        {
            return GetReservationKey(this.NetworkID, this.ReservationID).GetHashCode();
        }

        public object Clone()
        {
            ReservationConfiguration reservationConfiguration = new ReservationConfiguration(
                this.NetworkID,
                this.ReservationID,
                this.AllocationRequest);
            reservationConfiguration.CreatedDateTime = this.CreatedDateTime;
            reservationConfiguration.LastUpdatedDateTime = this.LastUpdatedDateTime;
            reservationConfiguration.ReservationState = this.ReservationState;
            reservationConfiguration.Status = this.Status;
            reservationConfiguration.AllocatedSubnet = this.AllocatedSubnet;
            reservationConfiguration.AssociatedNCID = this.AssociatedNCID;
            reservationConfiguration.APIVersion = this.APIVersion;
            reservationConfiguration.FailureDetail = this.FailureDetail != null ? (ErrorInfo)this.FailureDetail.Clone() : null;
            reservationConfiguration.AssociatedNCID = this.AssociatedNCID;
            return reservationConfiguration;
        }
    }
}
