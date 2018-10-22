using DNCService;
using NetworkManager.DataModel;
using NMAgent;
using NMAgent.Data;
using System;
using System.Collections.Generic;
using System.Linq;
using System.Threading;
using System.Threading.Tasks;

namespace NetworkManager
{
    // TODO: Add comments
    class TagRefreshMonitor
    {
        private CancellationToken cancellationToken;

        private NetworkRepositoryManager networkRepositoryManager;

        private HostAgentClient hostAgentClient;

        private bool tagsCacheRehydrated = false;

        public TagRefreshMonitor(
            CancellationToken cancellationToken,
            HostAgentClient hostAgentClient,
            NetworkRepositoryManager networkRepositoryManager)
        {
            this.cancellationToken = cancellationToken;
            this.hostAgentClient = hostAgentClient;
            this.networkRepositoryManager = networkRepositoryManager;
        }

        public async void StartAsync()
        {
            ServiceEventSource.Current.Message("Started tag refresh monitor");
            while (true)
            {
                cancellationToken.ThrowIfCancellationRequested();
                await RefreshTags();
                await Task.Delay(TimeSpan.FromSeconds(20), cancellationToken);//ashvind - temp for testing
            }
        }

        private async Task HydrateTagsCache()
        {
            if (!tagsCacheRehydrated) {
                Dictionary<string, TagDefinition> repoTagsDefinitionMap =
                    await networkRepositoryManager.GetAllTagDefinitions();

                foreach (KeyValuePair<string, TagDefinition> repoTagDefinition in repoTagsDefinitionMap)
                {
                    TagsCache.UpdateCache(repoTagDefinition.Key, repoTagDefinition.Value);
                }

                tagsCacheRehydrated = true;
            }
        }

        private async Task RefreshTags()
        {
            await HydrateTagsCache();

            Dictionary<string, TagResponse> tagsToUpdate = new Dictionary<string, TagResponse>();

            foreach (KeyValuePair<string, TagDefinition> repoTagDefinition in TagsCache.GetCache())
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
                    TagsCache.UpdateTagDefinition(tagKey, tagResponse.Values);
                    TagDefinition updatedTagDefinition = TagsCache.GetTagDefinition(tagKey);

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

        private static bool AreTagsUpdated(
            IReadOnlyCollection<string> latestTagInfo,
            IReadOnlyCollection<string> currentTagInfo)
        {
            if (((latestTagInfo.Count == 0) && (currentTagInfo.Count == 0)) ||
                ((latestTagInfo.Count == currentTagInfo.Count) &&
                !latestTagInfo.Except(currentTagInfo).Any()))
            {
                return false;
            }

            ServiceEventSource.Current.Message($"TagRefreshMonitor: TagsUpdated true");
            return true;
        }

    }
}
