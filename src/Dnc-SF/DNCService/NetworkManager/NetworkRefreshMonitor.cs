using System;
using System.Collections.Generic;
using System.Threading;
using System.Threading.Tasks;
using DNCService;
using Microsoft.ServiceFabric.Data;
using Microsoft.ServiceFabric.Data.Collections;
using NetworkManager.DataModel;
using NMAgent;
using NMAgent.Data;


namespace NetworkManager
{
    class NetworkRefreshMonitor
    {
        private IReliableStateManager reliableStateManager;

        private CancellationToken cancellationToken;

        private NetworkRepositoryManager networkRepositoryManager;

        private HostAgentClient hostAgentClient;

        private TagsCacheManager tagsCacheManager;

        private uint refreshIntervalMs;

        private const uint defaultRefreshIntervalMs = 60000;

        public NetworkRefreshMonitor(
            IReliableStateManager reliableStateManager,
            HostAgentClient hostAgentClient,
            NetworkRepositoryManager networkRepositoryManager,
            TagsCacheManager tagsCacheManager,
            DncConfiguration dncConfiguration,
            CancellationToken cancellationToken)
        {
            this.reliableStateManager = reliableStateManager;
            this.cancellationToken = cancellationToken;
            this.hostAgentClient = hostAgentClient;
            this.networkRepositoryManager = networkRepositoryManager;
            this.tagsCacheManager = tagsCacheManager;
            refreshIntervalMs = (dncConfiguration.DNCSettings.NetworkRefreshInterval > 0) ?
                dncConfiguration.DNCSettings.NetworkRefreshInterval : defaultRefreshIntervalMs;
        }

        public async void StartAsync()
        {
            ServiceEventSource.Current.Message("Started refresh monitor");
                while (true)
                {
                    cancellationToken.ThrowIfCancellationRequested();
                    ICollection<string> networkKeys =
                            await networkRepositoryManager.GetAllNetworkKeys(Status.Completed);
                    foreach (string networkKey in networkKeys)
                    {
                        string networkID = NetworkConfiguration.GetNetworkIDFromKey(networkKey);
                        ServiceEventSource.Current.Message($"Refreshing network:{networkID}");
                        await RefreshNetwork(networkID);
                        cancellationToken.ThrowIfCancellationRequested();
                    }
                    await Task.Delay(TimeSpan.FromMilliseconds(refreshIntervalMs), cancellationToken);
                }
        }

        private async Task RefreshNetwork(string networkID)
        {
            try
            {
                VirtualNetworkResponse virtualNetworkResponse =
                    await this.hostAgentClient.JoinAndGetNetwork(networkID);
                ServiceEventSource.Current.Message($"Refresh:Retrieved network config:{virtualNetworkResponse}");

                using (var tx = RepositoryHelper.GetTransaction())
                {
                    string networkKey = NetworkConfiguration.GetNetworkKey(networkID);
                    NetworkConfiguration networkConfig = await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        networkRepositoryManager.NetworkConfigReliableCollection,
                        networkKey,
                        LockMode.Update);
                    if (networkConfig == null)
                    {
                        return;
                    }

                    if (UInt64.Parse(virtualNetworkResponse.VnetVersion) == networkConfig.VnetVersion)
                    {
                        return;
                    }

                    if ((networkConfig.State != NetworkState.Join) ||
                        (networkConfig.Status != Status.Completed))
                    {
                        ServiceEventSource.Current.Message($"Skipping refreshing of network " +
                            $"{networkID} as it is in state: {networkConfig.State}");
                        return;
                    }

                    await networkRepositoryManager.UpdateNetworkConfigurationUntransacted(
                        tx,
                        networkConfig,
                        virtualNetworkResponse);
                    await RepositoryHelper.CommitTransactionAsync(tx);
                }

                tagsCacheManager.CheckAndAddTagsCache(virtualNetworkResponse);
            }
            catch(Exception exception)
            {
                ServiceEventSource.Current.Message($"Failed to refresh network.Exception:{exception}");
            }
        }
    }
}
