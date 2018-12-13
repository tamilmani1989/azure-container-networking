//-----------------------------------------------------------------------
// <copyright file="RequestResetMonitor.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
//-----------------------------------------------------------------------

using DNCService;
using Microsoft.ServiceFabric.Data.Collections;
using NetworkManager.DataModel;
using System;
using System.Collections.Generic;
using System.Threading;
using System.Threading.Tasks;

namespace NetworkManager
{
    /// <summary>
    /// Implements the network container monitor.
    /// It monitors for the changes in network containers
    /// and processes those changes.
    /// </summary>
    class NetworkContainerMonitor
    {
        /// <summary>
        /// Cached cancellation token.
        /// </summary>
        private CancellationToken cancellationToken;

        /// <summary>
        /// Handle to the network manager.
        /// </summary>
        private NetworkRepositoryManager networkRepositoryManager;

        private uint refreshIntervalMs;

        private const uint defaultRefreshIntervalMs = 60000;

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="networkRepositoryManager">Supply the cancellation token.</param>
        /// <param name="dncConfiguration">Supply the Dnc configuration.</param>
        /// <param name="cancellationToken">Supply the cancellation token.</param>
        public NetworkContainerMonitor(
            NetworkRepositoryManager networkRepositoryManager,
            DncConfiguration dncConfiguration,
            CancellationToken cancellationToken)
        {
            this.networkRepositoryManager = networkRepositoryManager;
            this.cancellationToken = cancellationToken;
            refreshIntervalMs = (dncConfiguration.DNCSettings.NetworkContainerRefreshInterval > 0) ?
                dncConfiguration.DNCSettings.NetworkContainerRefreshInterval : defaultRefreshIntervalMs;
        }

        /// <summary>
        /// Start monitor.
        /// </summary>
        public async void StartAsync()
        {
            ServiceEventSource.Current.Message("Started network container refresh monitor");
            while (true)
            {
                cancellationToken.ThrowIfCancellationRequested();
                ICollection<string> nodeKeys =
                    await networkRepositoryManager.GetAllNodeKeys();
                foreach (string nodeKey in nodeKeys)
                {
                    await RefreshNodeNetworkContainers(nodeKey);
                }

                await Task.Delay(TimeSpan.FromMilliseconds(refreshIntervalMs), cancellationToken);
            }
        }

        /// <summary>
        /// Refresh network container configurations on the node.
        /// </summary>
        private async Task RefreshNodeNetworkContainers(
            string nodeKey)
        {
            try
            {
                NodeConfiguration nodeConfig;
                using (var tx = RepositoryHelper.GetTransaction())
                {
                    nodeConfig = await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        networkRepositoryManager.NodeConfigReliableCollection,
                        nodeKey);
                    if (nodeConfig == null)
                    {
                        return;
                    }
                }

                foreach (KeyValuePair<string, IReadOnlyDictionary<string, string>> deployedNCs
                    in nodeConfig.DeployedNCMap)
                {
                    foreach (string networkContainerID in deployedNCs.Value.Keys)
                    {
                        await CreateAssociationTask(nodeConfig.NodeID, deployedNCs.Key, networkContainerID);
                    }
                }
            }
            catch (Exception exception)
            {
                ServiceEventSource.Current.Message($"RefreshNodeNetworkContainers received exception: {exception}");
            }
        }

        /// <summary>
        /// Create network container association task by
        /// changing the state to RefreshAssociation
        /// </summary>
        private async Task CreateAssociationTask(
            string nodeID,
            string networkID,
            string networkContainerID)
        {
            using (var tx = RepositoryHelper.GetTransaction())
            {
                string networkContainerKey =
                    NetworkContainerInfo.GetNetworkContainerKey(networkID, networkContainerID);
                NCConfiguration ncConfig = await RepositoryHelper.GetFromReliableDictionary(
                    tx,
                    networkRepositoryManager.NCConfigReliableCollection,
                    networkContainerKey,
                    LockMode.Update);
                if (ncConfig == null)
                {
                    ServiceEventSource.Current.Message($"Configuration for network container " +
                        $"{networkContainerID} not found.");
                    return;
                }

                if (ncConfig.MarkedForDelete)
                {
                    ServiceEventSource.Current.Message($"Network container {networkContainerID} " +
                        $"is marked for deletion. Skipping refresh association.");
                    return;
                }

                if ((ncConfig.Status == Status.Failed) ||
                    (ncConfig.Status == Status.Denied))
                {
                    return;
                }

                if ((ncConfig.Status == Status.Completed) &&
                    (ncConfig.State == NetworkContainerState.Program))
                {
                    // Change the state to RefreshAssociation
                    ncConfig.State = NetworkContainerState.RefreshAssociation;
                    await RepositoryHelper.CreateOrUpdateReliableDictionary(
                        tx,
                        networkRepositoryManager.NCConfigReliableCollection,
                        networkContainerKey,
                        ncConfig);

                    ReservationConfiguration reservationConfig = await networkRepositoryManager.GetReservationConfiguration(
                        ReservationConfiguration.GetReservationKey(networkID, ncConfig.ReservationID),
                        tx);

                    // Add new request in pending queue to process refresh association
                    RequestContext requestContext = new RequestContext(
                        networkContainerKey,
                        networkID,
                        nodeID,
                        RequestType.NetworkContainerRequest);
                    requestContext.SetSubnetName(reservationConfig.AllocationRequest.SubnetName);

                    await RepositoryHelper.CreateOrUpdateReliableDictionary(
                        tx,
                        networkRepositoryManager.PendingRequestsReliableCollection,
                        networkContainerKey,
                        requestContext);

                    await RepositoryHelper.CommitTransactionAsync(tx);
                }
            }

            return;
        }
    }
}
