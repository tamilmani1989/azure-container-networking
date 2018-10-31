using DNCService;
using System;
using NetworkManager.DataModel;
using NMAgent.Data;
using System.Collections.Concurrent;
using System.Collections.Generic;
using System.Linq;
using NMAgent;
using System.Threading.Tasks;

namespace NetworkManager
{
    public class TagsCacheManager
    {
        private HostAgentClient hostAgentClient;

        private NetworkRepositoryManager networkRepositoryManager;

        private TagsCache tagsCache;

        private bool tagsCacheRehydrated = false;

        public TagsCacheManager()
        {
            tagsCache = new TagsCache();
        }

        public void Init(
            HostAgentClient hostAgentClient,
            NetworkRepositoryManager networkRepositoryManager)
        {
            this.hostAgentClient = hostAgentClient;
            this.networkRepositoryManager = networkRepositoryManager;
        }

        private void HydrateTagsCache()
        {
            if (!tagsCacheRehydrated)
            {
                Dictionary<string, TagDefinition> repoTagsDefinitionMap =
                    networkRepositoryManager.GetAllTagDefinitions().Result;

                foreach (KeyValuePair<string, TagDefinition> repoTagDefinition in repoTagsDefinitionMap)
                {
                    tagsCache.UpdateCache(repoTagDefinition.Key, repoTagDefinition.Value);
                }

                tagsCacheRehydrated = true;
            }
        }

        public async Task RefreshTags()
        {
            HydrateTagsCache();

            Dictionary<string, TagResponse> tagsToUpdate = new Dictionary<string, TagResponse>();
            foreach (KeyValuePair<string, TagDefinition> repoTagDefinition in tagsCache.GetCache())
            {
                string tagName = repoTagDefinition.Value.Name;
                string tagType = repoTagDefinition.Value.Type;
                List<string> tagValues = repoTagDefinition.Value.Values;
                TagResponse tagResponse;
                try
                {
                    tagResponse = await hostAgentClient.GetSystemTagDefinition(tagName);
                }
                catch (Exception exception)
                {
                    string errorMsg = (exception.InnerException == null) ?
                        exception.Message : exception.InnerException.Message;
                    ServiceEventSource.Current.Message(
                        $"GetSystemTagDefinition failed due to {errorMsg}");
                    continue;
                }

                if (tagResponse != null)
                {
                    if (AreTagsUpdated(tagResponse.Values, tagValues))
                    {
                        ServiceEventSource.Current.Message($"RefreshTags: Found updated tag {tagName} " +
                            $"with response {tagResponse}");

                        tagsToUpdate.Add(
                            TagDefinition.GetTagDefinitionKey(tagType, tagName),
                            tagResponse);
                    }
                }
            }

            using (var tx = RepositoryHelper.GetTransaction())
            {
                foreach (KeyValuePair<string, TagResponse> tagKvp in tagsToUpdate)
                {
                    string tagKey = tagKvp.Key;
                    TagResponse tagResponse = tagKvp.Value;

                    // Update the tag cache with the resolved tag values
                    tagsCache.UpdateTagDefinition(tagKey, tagResponse.Values);
                    TagDefinition updatedTagDefinition = tagsCache.GetTagDefinition(tagKey);

                    // Sync repository with tags cache
                    await RepositoryHelper.CreateOrUpdateReliableDictionary(
                        tx,
                        networkRepositoryManager.TagDefinitionReliableCollection,
                        tagKey,
                        updatedTagDefinition);
                    ServiceEventSource.Current.Message($"Persist tag definition: " +
                        $"{updatedTagDefinition.ToString()}");
                }

                await RepositoryHelper.CommitTransactionAsync(tx);
            }
        }

        public void CheckAndAddTagsCache(
            VirtualNetworkResponse vnetResponse)
        {
            foreach (Subnet subnet in vnetResponse.Subnets)
            {
                foreach (Tag tag in (subnet.Tags ?? Enumerable.Empty<Tag>()))
                {
                    tagsCache.RegisterTagForProcessing(tag);
                }
            }
        }

        public TagDefinition RegisterAndGetTagDefinition(
            Tag tag)
        {
            string tagKey = Tag.GetTagKey(tag);
            TagDefinition tagDefinition = tagsCache.GetTagDefinition(tagKey);
            if ((tagDefinition != null) && (tagDefinition.Version > 0))
            {
                return tagDefinition;
            }

            tagsCache.RegisterTagForProcessing(tag);

            RefreshTags().Wait();

            return tagsCache.GetTagDefinition(tagKey);
        }

        public static bool AreTagsUpdated(
            IReadOnlyCollection<string> latestTagInfo,
            IReadOnlyCollection<string> currentTagInfo)
        {
            if (((latestTagInfo.Count == 0) && (currentTagInfo.Count == 0)) ||
                ((latestTagInfo.Count == currentTagInfo.Count) &&
                !latestTagInfo.Except(currentTagInfo).Any()))
            {
                return false;
            }

            ServiceEventSource.Current.Message($"TagRefresh: TagsUpdated true");
            return true;
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

        private class TagsCache
        {
            private ConcurrentDictionary<string, TagDefinition> tagsCache;

            public TagsCache()
            {
                tagsCache = new ConcurrentDictionary<string, TagDefinition>();
            }

            public void UpdateCache(
                string key,
                TagDefinition tagDefinition)
            {
                tagsCache[key] = tagDefinition;
            }

            public ConcurrentDictionary<string, TagDefinition> GetCache()
            {
                return tagsCache;
            }

            public TagDefinition GetTagDefinition(
                string tagKey)
            {
                if (tagsCache.ContainsKey(tagKey))
                {
                    return tagsCache[tagKey];
                }

                return null;
            }

            public void UpdateTagDefinition(
                string tagKey,
                string[] values)
            {
                if (tagsCache.ContainsKey(tagKey))
                {
                    tagsCache[tagKey].UpdateTagDefinition(values);
                }
            }

            public void RegisterTagForProcessing(
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
        }
    }

}
