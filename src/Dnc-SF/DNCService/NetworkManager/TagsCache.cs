using DNCService;
using System;
using NetworkManager.DataModel;
using NMAgent.Data;
using System.Collections.Concurrent;
using System.Collections.Generic;
using System.Linq;

namespace NetworkManager
{
    //TODO: Add comments
    static class TagsCache
    {
        private static volatile ConcurrentDictionary<string, TagDefinition> tagsCache =
            new ConcurrentDictionary<string, TagDefinition>();

        public static void UpdateCache(
            string key,
            TagDefinition tagDefinition)
        {
            tagsCache[key] = tagDefinition;
        }

        public static ConcurrentDictionary<string, TagDefinition> GetCache()
        {
            return tagsCache;
        }

        public static TagDefinition GetTagDefinition(
            string tagKey)
        {
            if (tagsCache.ContainsKey(tagKey))
            {
                return tagsCache[tagKey];
            }

            return null;
        }

        public static void UpdateTagDefinition(
            string tagKey,
            string[] values)
        {
            if (tagsCache.ContainsKey(tagKey))
            {
                tagsCache[tagKey].UpdateTagDefinition(values);
            }
        }

        public static void CheckAndAddTagsCache(
            VirtualNetworkResponse vnetResponse)
        {
            foreach (Subnet subnet in vnetResponse.Subnets)
            {
                foreach (Tag tag in subnet.Tags)
                {
                    RegisterTagForProcessing(tag);
                }
            }
        }

        private static void RegisterTagForProcessing(
            Tag tag)
        {
            string tagKey = Tag.GetTagKey(tag);
            if (!tagsCache.ContainsKey(tagKey))
            {
                tagsCache[tagKey] = new TagDefinition(tag.Name, tag.Type);
                ServiceEventSource.Current.Message($"Added new tag definition in tagCache. " +
                    $"Tag key:{tagKey} Tag definition: {tagsCache[tagKey].ToString()}");
            }
        }

        public static TagDefinition RegisterAndGetTagDefinition(
            Tag tag)
        {
            RegisterTagForProcessing(tag);
            return tagsCache[Tag.GetTagKey(tag)];
        }

        public static bool AreTagsUpdated(
            IReadOnlyDictionary<string, UInt64> latestTagInfo,
            IReadOnlyDictionary<string, UInt64> currentTagInfo)
        {
            if ((!latestTagInfo.Any() && !currentTagInfo.Any()) ||
                ((latestTagInfo.Count == currentTagInfo.Count) &&
                !latestTagInfo.Except(currentTagInfo).Any()))
            {
                return false;
            }

            ServiceEventSource.Current.Message($"TagsCache: TagsUpdated true");
            return true;
        }
    }
}
