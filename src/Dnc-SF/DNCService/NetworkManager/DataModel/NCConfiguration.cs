using DNCService.API;
using System;
using System.Collections.Generic;
using System.Linq;
using System.Runtime.Serialization;
using System.Text;
using System.Text.RegularExpressions;

namespace NetworkManager.DataModel
{
    [DataContract]
    public class NCConfiguration :ICloneable
    {
        [DataMember]
        public string NetworkContainerID { get; private set; }

        [DataMember]
        public string NetworkID { get; private set; }

        [DataMember]
        public string ReservationID { get; set; }

        [DataMember]
        public bool ImplicitReservation { get; set; }

        [DataMember]
        public NetworkContainerRequest Request { get; private set; }
        
        [DataMember]
        public NetworkContainerState State { get; set; }

        [DataMember]
        public Status Status { get; set; }

        [DataMember]
        public UInt64 CNSVersion { get; set; }

        [DataMember]
        public UInt64 AgentVersion { get; set; }

        [DataMember]
        public UInt64 ProgrammedVnetVersion { get; set; }

        [DataMember]
        private Dictionary<string, UInt64> programmedTagsInfo;

        [DataMember]
        public ErrorInfo FailureDetail { get; set; }

        [DataMember]
        public bool MarkedForDelete { get; set; }

        [DataMember]
        public string APIVersion { get; set; }

        [DataMember]
        public string LocalIP { get; set; }

        public IReadOnlyDictionary<string, UInt64> ProgrammedTagsInfo
        {
            get
            {
                return this.programmedTagsInfo;
            }
        }

        public NCConfiguration(string networkID, string ncID, NetworkContainerRequest ncRequest, string apiVersion)
        {
            this.NetworkID = networkID;
            this.NetworkContainerID = ncID;
            this.programmedTagsInfo = new Dictionary<string, UInt64>();
            this.Request = ncRequest;
            this.APIVersion = apiVersion;
        }

        public void UpdateProgrammedTagsInfo(IReadOnlyDictionary<string, ulong> programmedTags)
        {
            programmedTagsInfo.Clear();

            foreach (KeyValuePair<string, ulong> TagsKvp in programmedTags)
            {
                programmedTagsInfo.Add(TagsKvp.Key, TagsKvp.Value);
            }
        }

        public NCConfiguration(NCConfiguration ncConfiguration)
        {
            this.NetworkID = ncConfiguration.NetworkID;
            this.NetworkContainerID = ncConfiguration.NetworkContainerID;
            this.Request = (NetworkContainerRequest)ncConfiguration.Request.Clone();
            this.ReservationID = ncConfiguration.ReservationID;
            this.ImplicitReservation = ncConfiguration.ImplicitReservation;
            this.State = ncConfiguration.State;
            this.Status = ncConfiguration.Status;
            this.CNSVersion = ncConfiguration.CNSVersion;
            this.AgentVersion = ncConfiguration.AgentVersion;
            this.ProgrammedVnetVersion = ncConfiguration.ProgrammedVnetVersion;
            this.programmedTagsInfo = new Dictionary<string, ulong>(ncConfiguration.programmedTagsInfo);
            this.FailureDetail = ncConfiguration.FailureDetail != null ? new ErrorInfo(ncConfiguration.FailureDetail) : null;
            this.MarkedForDelete = ncConfiguration.MarkedForDelete;
            this.APIVersion = ncConfiguration.APIVersion;
            this.LocalIP = ncConfiguration.LocalIP;
        }

        public override string ToString()
        {
            StringBuilder sb = new StringBuilder();
            sb.Append($"NetworkContainerID:{this.NetworkContainerID} ");
            sb.Append($" NetworkID:{this.NetworkID} ");
            sb.Append($"ReservationID:{this.ReservationID} ");
            sb.Append($"ImplicitReservation:{this.ImplicitReservation} ");
            sb.Append($"NCRequest:{this.Request} ");
            sb.Append($"State:{this.State} ");
            sb.Append($"Status:{ this.Status} ");
            sb.Append($"CNSVersion:{this.CNSVersion} ");
            sb.Append($"AgentVersion:{this.AgentVersion} ");
            sb.Append($"ProgrammedVnetVersion:{this.ProgrammedVnetVersion} ");
            sb.AppendFormat("ProgrammedTagsInfo:{0} ",
                string.Join(",",
                    this.ProgrammedTagsInfo.Select(item => $"[{item.Key}:{item.Value}]")));
            sb.Append($"FailureDetail:{this.FailureDetail}");
            sb.Append($"MarkedForDelete:{this.MarkedForDelete}");
            sb.Append($"APIVersion:{this.APIVersion}");
            sb.Append($"LocalIP:{this.LocalIP}");
            return sb.ToString();
        }

        public static string GetNCKey(string networkID, string networkContainerID)
        {
            return string.Format(Constants.NCKey, networkID, networkContainerID);
        }

        public static string GetNetworkIDFromKey(string ncKey)
        {
            string ncRegex = string.Format(Constants.NCKey, "(.*)", "(.*)");
            return Regex.Match(ncKey, ncRegex).Groups[1].Captures[0].Value;
        }

        public static string GetNetworkContainerIDFromKey(string ncKey)
        {
            string ncRegex = string.Format(Constants.NCKey, "(.*)", "(.*)");
            return Regex.Match(ncKey, ncRegex).Groups[2].Captures[0].Value;
        }

        public override bool Equals(object obj)
        {
            if (obj == null || GetType() != obj.GetType())
                return false;
            NCConfiguration ncConfiguration = obj as NCConfiguration;

            return this.NetworkID == ncConfiguration.NetworkID && this.NetworkContainerID == ncConfiguration.NetworkContainerID;
        }

        public override int GetHashCode()
        {
            return GetNCKey(this.NetworkID, this.NetworkContainerID).GetHashCode();
        }

        public object Clone()
        {
            return new NCConfiguration(this);
        }

    }
}
