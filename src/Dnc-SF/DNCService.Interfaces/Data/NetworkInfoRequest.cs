using System;
using System.Collections.Generic;
using System.Runtime.Serialization;
using System.Text;

namespace DNCService.API
{
    [DataContract]
    public class NetworkInfoRequest : ICloneable
    {

        public const string AzureNet = "AzureNet";

        [DataMember]
        public string NetworkType { get; private set; }
        [DataMember]
        public string NetworkSpace { get; private set; }

        public NetworkInfoRequest(string networkType, string networkSpace)
        {
            if (string.IsNullOrWhiteSpace(networkType))
            {
                throw new ArgumentException("Cannot be null or empty", nameof(networkType));
            }

            if(string.Compare(networkType, AzureNet, StringComparison.CurrentCultureIgnoreCase) !=0)
            {
                throw new ArgumentException($"Only {AzureNet} network type is supported");
            }

            this.NetworkType = networkType;
            this.NetworkSpace = networkSpace;
        }

        public override string ToString()
        {
            return $"NetworkType:{this.NetworkType} NetworkSpace:{this.NetworkSpace}";
        }

        public object Clone()
        {
            return new NetworkInfoRequest(this.NetworkType, this.NetworkSpace);
        }
    }
}
