using DNCService;
using System;
using System.Collections.Generic;
using System.Runtime.Serialization;

namespace NetworkManager.DataModel
{
    [DataContract]
    public class TagDefinition : ICloneable
    {
        [DataMember]
        public string Name { get; private set; }

        [DataMember]
        public string Type { get; private set; }

        [DataMember]
        public List<string> Values { get; private set; }

        [DataMember]
        public DateTime CreatedDateTime { get; private set; }

        [DataMember]
        public DateTime LastUpdatedDateTime { get; set; }

        [DataMember]
        public UInt64 Version { get; set; }

        public TagDefinition(string name, string type)
        {
            Name = name;
            Type = type;
            Version = 0;
            CreatedDateTime = DateTime.UtcNow;
            LastUpdatedDateTime = DateTime.UtcNow;
            Values = new List<string>();
        }

        public TagDefinition(TagDefinition tagDefinition)
        {
            Name = tagDefinition.Name;
            Type = tagDefinition.Type;
            CreatedDateTime = tagDefinition.CreatedDateTime;
            LastUpdatedDateTime = tagDefinition.LastUpdatedDateTime;
            Values = new List<string>(tagDefinition.Values);
        }

        public void UpdateTagDefinition(ICollection<string> values)
        {
            Values.Clear();
            Values.AddRange(values);
            LastUpdatedDateTime = DateTime.UtcNow;
            ++Version;
            ServiceEventSource.Current.Message($"Updated tagCache with tag definition: {ToString()}");
        }

        public override string ToString()
        {
            return $"Name:{this.Name} Type:{this.Type} Version:{this.Version} " +
                $"Values:{string.Join(",", this.Values)} CreatedDateTime:{this.CreatedDateTime} " +
                $"LastUpdatedDateTime:{this.LastUpdatedDateTime}";
        }

        public static string GetTagDefinitionKey(string type, string name)
        {
            return string.Format(Constants.TagDefinitionKey, type, name);
        }


        public override bool Equals(object obj)
        {
            if (obj == null || GetType() != obj.GetType())
                return false;
            TagDefinition tagDefinition = obj as TagDefinition;

            return this.Name == tagDefinition.Name &&  this.Type == tagDefinition.Type;
        }

        public override int GetHashCode()
        {
            return GetTagDefinitionKey(this.Type, this.Name).GetHashCode();
        }

        public object Clone()
        {
            return new TagDefinition(this);
        }
    }
}
