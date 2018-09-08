using System;
using System.Collections.Generic;
using System.Threading;
using System.Threading.Tasks;
using Cns;
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

        private CnsClient cnsClient;

        public NetworkRefreshMonitor(
            IReliableStateManager reliableStateManager,
            CancellationToken cancellationToken,
            HostAgentClient hostAgentClient,
            CnsClient cnsClient,
            NetworkRepositoryManager networkRepsoitoryManager)
        {
            this.reliableStateManager = reliableStateManager;
            this.cancellationToken = cancellationToken;
            this.hostAgentClient = hostAgentClient;
            this.cnsClient = cnsClient;
            this.networkRepositoryManager = networkRepsoitoryManager;
        }

        public async void StartAsync()
        {
            ServiceEventSource.Current.Message("Started refresh monitor");
                while (true)
                {
                    cancellationToken.ThrowIfCancellationRequested();
                    ICollection<string> networkIDs = 
                            await networkRepositoryManager.GetAllNetworkKeys(Status.Completed);
                    foreach (string networkID in networkIDs)
                    {
                        ServiceEventSource.Current.Message($"Refreshing network:{networkID}");
                        await RefreshNetwork(networkID);
                        cancellationToken.ThrowIfCancellationRequested();
                    }
                    await Task.Delay(TimeSpan.FromSeconds(60), cancellationToken);
                }
        }

        private async Task RefreshNetwork(string networkID)
        {
            IReliableDictionary<string, NetworkConfiguration> networkConfigReliableCollection =
                RepositoryHelper.GetReliableDictionaryAsync<string, NetworkConfiguration>(
                Constants.NetworkConfigurationTableName).Result;
            IReliableDictionary<string, SubnetConfiguration> subnetConfigReliableCollection =
                RepositoryHelper.GetReliableDictionaryAsync<string, SubnetConfiguration>(
                Constants.SubnetConfigurationTableName).Result;
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
                        networkConfigReliableCollection,
                        networkKey,
                        LockMode.Update);

                    if (networkConfig == null)
                    {
                        return;
                    }

                    if (UInt64.Parse(virtualNetworkResponse.VnetVersion) == networkConfig.Version)
                    {
                        return;
                    }
                    await networkRepositoryManager.UpdateNetworkConfigurationUntransacted(
                        tx,
                        networkConfig,
                        virtualNetworkResponse);

                    await RepositoryHelper.CommitTransactionAsync(tx);
                }
            }
            catch(Exception exception)
            {
                ServiceEventSource.Current.Message($"Failed to refresh network.Exception:{exception}");
            }
        }
    }
}
