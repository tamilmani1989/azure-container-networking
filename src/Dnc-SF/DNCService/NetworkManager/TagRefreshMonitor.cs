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

        private uint refreshIntervalMs;

        private const uint defaultRefreshIntervalMs = 60000;

        public TagRefreshMonitor(
            TagsCacheManager tagsCacheManager,
            DncConfiguration dncConfiguration,
            CancellationToken cancellationToken)
        {
            this.cancellationToken = cancellationToken;
            this.tagsCacheManager = tagsCacheManager;
            refreshIntervalMs = (dncConfiguration.DNCSettings.TagRefreshInterval > 0) ?
                dncConfiguration.DNCSettings.TagRefreshInterval : defaultRefreshIntervalMs;
        }

        public async void StartAsync()
        {
            ServiceEventSource.Current.Message("Started tag refresh monitor");
            while (true)
            {
                cancellationToken.ThrowIfCancellationRequested();
                await tagsCacheManager.RefreshTags();
                await Task.Delay(TimeSpan.FromMilliseconds(refreshIntervalMs), cancellationToken);
            }
        }
    }
}
