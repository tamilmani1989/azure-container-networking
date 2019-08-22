using System;
using System.Collections.Generic;
using System.Runtime.Serialization;
using System.Text;

namespace NetworkManager.DataModel
{
    [DataContract]
    public class RequestContext :ICloneable
    {
        [DataMember]
        public string RequestID { get; private set; }

        [DataMember]
        public string NetworkID { get; private set; }

        [DataMember]
        public string NodeID { get; private set; }

        [DataMember]
        public string SubnetName { get; private set; }

        [DataMember]
        public RequestType Type { get; private set; }

        [DataMember]
        public DateTime CreatedDateTime { get; private set; }

        [DataMember]
        public DateTime LastRetriedDateTime { get; set; }

        [DataMember]
        public TimeSpan RetryInterval { get; set; }

        [DataMember]
        public Guid TaskID  { get; private set; }

        public RequestContext(string requestID, string networkID, string nodeID, RequestType requestType)
        {
            this.RequestID = requestID;
            this.NodeID = nodeID;
            this.NetworkID = networkID;
            this.Type = requestType;
            this.CreatedDateTime = DateTime.UtcNow;
            this.TaskID = Guid.NewGuid();
        }

        public RequestContext(RequestContext requestContext)
        {
            this.RequestID = requestContext.RequestID;
            this.NodeID = requestContext.NodeID;
            this.NetworkID = requestContext.NetworkID;
            this.Type = requestContext.Type;
            this.CreatedDateTime = requestContext.CreatedDateTime;
            this.LastRetriedDateTime = requestContext.LastRetriedDateTime;
            this.RetryInterval = requestContext.RetryInterval;
            this.TaskID = requestContext.TaskID;
            this.SubnetName = requestContext.SubnetName;
        }

        public void SetSubnetName(string subnetName)
        {
            SubnetName = subnetName;
        }

        public override string ToString()
        {
            return $"RequestID:{this.RequestID} NetworkID:{this.NetworkID} NodeID:{this.NodeID} SubnetName:{this.SubnetName} " +
                $"Type:{this.Type} CreatedDateTime:{this.CreatedDateTime} LastRetriedDateTime:{this.LastRetriedDateTime} " +
                $"RetryInterval:{this.RetryInterval}, TaskID: {TaskID}";
        }

        public override bool Equals(object obj)
        {
            if (obj == null || GetType() != obj.GetType())
                return false;
            RequestContext requestContext = obj as RequestContext;

            return this.RequestID == requestContext.RequestID && this.TaskID == requestContext.TaskID;
        }

        public override int GetHashCode()
        {
            return this.RequestID.GetHashCode() ^ this.TaskID.GetHashCode();
        }

        public object Clone()
        {
            return new RequestContext(this);
        }
    }
}
