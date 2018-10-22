using NetworkManager.DataModel;
using Newtonsoft.Json;
using System.Runtime.Serialization;

namespace NMAgent.Data
{
    [DataContract]
    public class Tag
    {
        [DataMember]
        [JsonProperty("name")]
        public string Name { get; set; }

        [DataMember]
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

        public static string GetTagKey(Tag tag)
        {
            return string.Format(Constants.TagDefinitionKey, tag.Type, tag.Name);
        }
    }
}
