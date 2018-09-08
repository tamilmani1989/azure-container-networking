using Newtonsoft.Json;
using System;
using System.Collections.Generic;
using System.Text;

namespace NMAgent.Data
{
    public class TagResponse
    {
        [JsonProperty("httpStatusCode")]
        public string ResponseCode { get; set; }

        [JsonProperty("tagName")]
        public string Name { get; set; }

        [JsonProperty("addresses")]
        public string[] Values { get; set; }

        public TagResponse(string responseCode, string name, string[] values)
        {
            this.ResponseCode = responseCode;
            this.Name = name;
            this.Values = values;
        }

        public override string ToString()
        {
            string values = this.Values != null ? string.Join(",", this.Values) : "null";
            return $"ResponseCode:{this.ResponseCode} Name:{this.Name} Values:{values}";
        }
    }
}
