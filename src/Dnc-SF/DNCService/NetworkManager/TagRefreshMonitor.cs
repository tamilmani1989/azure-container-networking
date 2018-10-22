using DNCService;
using System;
using System.Threading;
using System.Threading.Tasks;

namespace NetworkManager
{
    // TODO: Add comments
    class TagRefreshMonitor
    {
        private CancellationToken cancellationToken;

        private TagsCacheManager tagsCacheManager;

        private UInt32 defaultRefreshIntervalMs;

        public TagRefreshMonitor(
            CancellationToken cancellationToken,
            TagsCacheManager tagsCacheManager)
        {
            this.cancellationToken = cancellationToken;
            this.tagsCacheManager = tagsCacheManager;
            defaultRefreshIntervalMs = 1;
        }

        public async void StartAsync()
        {
            ServiceEventSource.Current.Message("Started tag refresh monitor");
            while (true)
            {
                cancellationToken.ThrowIfCancellationRequested();
                await tagsCacheManager.RefreshTags();
                await Task.Delay(TimeSpan.FromSeconds(60), cancellationToken);
            }
        }
    }
}
