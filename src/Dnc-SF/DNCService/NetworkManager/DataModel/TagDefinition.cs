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
            this.Name = name;
            this.Type = type;
            this.CreatedDateTime = DateTime.UtcNow;
            this.LastUpdatedDateTime = DateTime.UtcNow;
            this.Values = new List<string>();
        }

        public TagDefinition(TagDefinition tagDefinition)
        {
            this.Name = tagDefinition.Name;
            this.Type = tagDefinition.Type;
            this.CreatedDateTime = tagDefinition.CreatedDateTime;
            this.LastUpdatedDateTime = tagDefinition.LastUpdatedDateTime;
            this.Values = new List<string>(tagDefinition.Values);
        }

        public void UpdateTagDefinition(UInt64 version, ICollection<string> values)
        {
            this.Values.Clear();
            this.Values.AddRange(values);
            this.LastUpdatedDateTime = DateTime.UtcNow;
        }

        public override string ToString()
        {
            return $"Name:{this.Name} Type:{this.Type} Version:{this.Version} " +
                $"Values:{string.Join(",", this.Values)} CreatedDateTime:{this.CreatedDateTime} " +
                $"LastUpdatedDateTime:{this.LastUpdatedDateTime}";
        }

        public static string GetTagDefinitionKey(string name, string type)
        {
            return string.Format(Constants.TagDefinitionKey, name, type);
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
            return GetTagDefinitionKey(this.Name, this.Type).GetHashCode();
        }

        public object Clone()
        {
            return new TagDefinition(this);
        }
    }
}
