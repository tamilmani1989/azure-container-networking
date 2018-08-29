using Newtonsoft.Json;
using System;
using System.Collections.Generic;
using System.Text;

namespace NMAgent.Data
{
    public class Tag
    {
        [JsonProperty("name")]
        public string Name { get; set; }

        [JsonProperty("type")]
        public string Type { get; set; }

        public Tag(string name, string type)
        {
            this.Name = name;
            this.Type = type;
        }

        public override string ToString()
        {
            return $"Name:{this.Name} Type:{this.Type}";
        }
    }
}
