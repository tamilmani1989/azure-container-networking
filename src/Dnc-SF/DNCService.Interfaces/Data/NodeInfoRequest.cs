using System;
using System.Collections.Generic;
using System.Runtime.Serialization;
using System.Text;

namespace DNCService.API
{
    [DataContract]
    public class NodeInfoRequest : ICloneable
    {
        [DataMember]
        public string[] IPAddresses { get; private set; }

        [DataMember]
        public string LocationType { get; private set; }

        [DataMember]
        public string OrchestratorType { get; private set; }

        public NodeInfoRequest(string[] ipAddresses, string locationType, string orchestratorType)
        {
            if (ipAddresses == null)
            {
                throw new ArgumentNullException(nameof(ipAddresses));
            }

            if (ipAddresses.Length == 0 )
            {
                throw new ArgumentException("Cannot be empty list", nameof(ipAddresses));
            }

            if (string.IsNullOrWhiteSpace(locationType))
            {
                throw new System.ArgumentException("message", nameof(locationType));
            }

            if (string.IsNullOrWhiteSpace(orchestratorType))
            {
                throw new System.ArgumentException("message", nameof(orchestratorType));
            }

            this.IPAddresses = ipAddresses;
            this.LocationType = locationType;
            this.OrchestratorType = orchestratorType;
        }

        public object Clone()
        {
            return new NodeInfoRequest(
                this.IPAddresses, 
                this.LocationType, 
                this.OrchestratorType);
        }
    }

}
