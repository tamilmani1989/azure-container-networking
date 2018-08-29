using Newtonsoft.Json;
using System;
using System.Collections.Generic;
using System.Text;

namespace NMAgent.Data
{
    public class Subnet
    {
        [JsonProperty("name")]
        public string SubnetName { get; set; }

        [JsonProperty("addressPrefix")]
        public string AddressPrefix { get; set; }

        [JsonProperty("tags")]
        public Tag[] Tags { get; set; }

        [JsonProperty("defaultGateway")]
        public string DefaultGateway { get; set; }

        public Subnet(string subnetName, string addressPrefix, Tag[] tags, string defaultGateway)
        {
            this.SubnetName = subnetName;
            this.AddressPrefix = addressPrefix;
            this.Tags = tags;
            this.DefaultGateway = defaultGateway;
        }

        public override string ToString()
        {
            string tags = this.Tags != null ? string.Join(",", (object[])this.Tags) : null;
            return $"SubnetName:{this.SubnetName} :{this.AddressPrefix} Tags:{tags} DefaultGateway:{this.DefaultGateway}";
        }
    }
}
