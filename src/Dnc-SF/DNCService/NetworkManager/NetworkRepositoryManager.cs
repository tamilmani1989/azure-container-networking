using DNCService;
using DNCService.API;
using Microsoft.ServiceFabric.Data;
using Microsoft.ServiceFabric.Data.Collections;
using NetworkManager.DataModel;
using NMAgent;
using NMAgent.Data;
using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;
namespace NetworkManager
{
    public class NetworkRepositoryManager
    {
        public IReliableDictionary<string, RequestContext> PendingRequestsReliableCollection { get; private set; }

        public IReliableDictionary<string, RequestContext> FailedRequestsReliableCollection { get; private set; }

        public IReliableDictionary<string, NetworkConfiguration> NetworkConfigReliableCollection { get; private set; }

        public IReliableDictionary<string, SubnetConfiguration> SubnetConfigReliableCollection { get; private set; }

        public IReliableDictionary<string, NCConfiguration> NCConfigReliableCollection { get; private set; }

        public IReliableDictionary<string, ReservationConfiguration> ReservationReliableCollection { get; private set; }

        public IReliableDictionary<string, NodeConfiguration> NodeConfigReliableCollection { get; private set; }

        public IReliableDictionary<string, NetworkProfile> NetworkProfilesReliableCollection { get; private set; }

        public IReliableDictionary<string, TagDefinition> TagDefinitionReliableCollection { get; private set; }

        private DncConfiguration dncConfiguration;

        public NetworkRepositoryManager(DncConfiguration dncConfiguration)
        {
            PendingRequestsReliableCollection =
                RepositoryHelper.GetReliableDictionaryAsync<string, RequestContext>(
                    Constants.PendingRequestsTableName).Result;
            FailedRequestsReliableCollection =
                RepositoryHelper.GetReliableDictionaryAsync<string, RequestContext>(
                    Constants.FailedRequestsTableName).Result;
            NetworkConfigReliableCollection =
                RepositoryHelper.GetReliableDictionaryAsync<string, NetworkConfiguration>(
                Constants.NetworkConfigurationTableName).Result;
            SubnetConfigReliableCollection =
                RepositoryHelper.GetReliableDictionaryAsync<string, SubnetConfiguration>(
                Constants.SubnetConfigurationTableName).Result;
            NCConfigReliableCollection =
                RepositoryHelper.GetReliableDictionaryAsync<string, NCConfiguration>(
                    Constants.NetworkContainerTableName).Result;
            ReservationReliableCollection =
                RepositoryHelper.GetReliableDictionaryAsync<string, ReservationConfiguration>(
                    Constants.IPReservationsTableName).Result;
            NodeConfigReliableCollection =
                RepositoryHelper.GetReliableDictionaryAsync<string, NodeConfiguration>(
                    Constants.NodeTableName).Result;
            NetworkProfilesReliableCollection =
                RepositoryHelper.GetReliableDictionaryAsync<string, NetworkProfile>(
                    Constants.NetworkProfilesTableName).Result;
            TagDefinitionReliableCollection =
                RepositoryHelper.GetReliableDictionaryAsync<string, TagDefinition>(
                    Constants.TagDefinitionTableName).Result;

            this.dncConfiguration = dncConfiguration;
        }

        public async Task<ICollection<string>> GetAllNetworkKeys(Status status)
        {
            List<string> networkKeys = new List<string>();

            using (var tx = RepositoryHelper.GetTransaction())
            {
                var enumerable = await NetworkConfigReliableCollection.CreateEnumerableAsync(tx);
                var asyncEnumerator = enumerable.GetAsyncEnumerator();

                while (await asyncEnumerator.MoveNextAsync(RepositoryHelper.CancellationToken))
                {
                    if (asyncEnumerator.Current.Value.Status == status)
                    {
                        networkKeys.Add(asyncEnumerator.Current.Key);
                    }
                }
            }

            return networkKeys;
        }

        public async Task<ICollection<string>> GetAllNodeKeys()
        {
            List<string> nodeKeys = new List<string>();

            using (var tx = RepositoryHelper.GetTransaction())
            {
                var enumerable = await NodeConfigReliableCollection.CreateEnumerableAsync(tx);
                var asyncEnumerator = enumerable.GetAsyncEnumerator();

                while (await asyncEnumerator.MoveNextAsync(RepositoryHelper.CancellationToken))
                {
                    nodeKeys.Add(asyncEnumerator.Current.Key);
                }
            }

            return nodeKeys;
        }

        public async Task<ICollection<string>> GetAllRequestIDs(bool pending = true)
        {
            List<string> requestIDs = new List<string>();
            using (var tx = RepositoryHelper.GetTransaction())
            {
                var enumerable = pending ?
                    await PendingRequestsReliableCollection.CreateEnumerableAsync(tx) :
                    await FailedRequestsReliableCollection.CreateEnumerableAsync(tx);
                var asyncEnumerator = enumerable.GetAsyncEnumerator();

                while (await asyncEnumerator.MoveNextAsync(RepositoryHelper.CancellationToken))
                {
                    requestIDs.Add(asyncEnumerator.Current.Key);
                }
            }

            return requestIDs;
        }

        public async Task<Dictionary<string, TagDefinition>> GetAllTagDefinitions()
        {
            Dictionary<string, TagDefinition> tagDefinitionMap = new Dictionary<string, TagDefinition>();
            using (var tx = RepositoryHelper.GetTransaction())
            {
                var enumerable = await TagDefinitionReliableCollection.CreateEnumerableAsync(tx);
                var asyncEnumerator = enumerable.GetAsyncEnumerator();

                while (await asyncEnumerator.MoveNextAsync(RepositoryHelper.CancellationToken))
                {
                    tagDefinitionMap.Add(asyncEnumerator.Current.Key, asyncEnumerator.Current.Value);
                }
            }

            return tagDefinitionMap;
        }

        public async Task<RequestContext> GetRequestContext(string requestID)
        {
            using (var tx = RepositoryHelper.GetTransaction())
            {
                RequestContext requestContext = await RepositoryHelper.GetFromReliableDictionary<string, RequestContext>(
                    tx,
                    PendingRequestsReliableCollection,
                    requestID);

                return requestContext;
            }
        }

        public async Task UpdateNetworkConfigurationUntransacted(
            ITransaction transaction,
            NetworkConfiguration networkConfig,
            VirtualNetworkResponse virtualNetworkResponse)
        {

            Dictionary<string, SubnetConfiguration> subnetConfigMap =
                new Dictionary<string, SubnetConfiguration>();
            HashSet<string> missingSubnetsMap = new HashSet<string>();
            foreach (KeyValuePair<string, string> subnetKvp in networkConfig.Subnets)
            {
                string subnetName = subnetKvp.Key;
                string subnetKey = subnetKvp.Value;

                SubnetConfiguration subnetConfiguration = await RepositoryHelper.GetFromReliableDictionary(
                    transaction,
                    SubnetConfigReliableCollection,
                    subnetKey,
                    LockMode.Update);

                if (subnetConfiguration == null)
                {
                    missingSubnetsMap.Add(subnetName);
                    continue;
                }

                subnetConfigMap[subnetKey] = subnetConfiguration;
            }

            foreach (string subnetName in missingSubnetsMap)
            {
                networkConfig.RemoveSubnet(subnetName);
            }

            HashSet<string> removedSubnets;
            NMAgentToRepoTranslators.UpdateNetworkAndSubnetConfigurations(
                networkConfig,
                subnetConfigMap,
                virtualNetworkResponse,
                out removedSubnets);

            // Mark the join operation as completed.
            networkConfig.State = NetworkState.Join;
            networkConfig.Status = Status.Completed;

            await RepositoryHelper.CreateOrUpdateReliableDictionary(
                transaction,
                NetworkConfigReliableCollection,
                NetworkConfiguration.GetNetworkKey(networkConfig.NetworkID),
                networkConfig);

            foreach (KeyValuePair<string, SubnetConfiguration> subnetKvp in subnetConfigMap)
            {
                await RepositoryHelper.CreateOrUpdateReliableDictionary(
                    transaction,
                    SubnetConfigReliableCollection,
                    subnetKvp.Key,
                    subnetKvp.Value);
            }

            foreach (string removedSubnet in removedSubnets)
            {
                await RepositoryHelper.RemoveFromReliableDictionary(
                    transaction,
                    SubnetConfigReliableCollection,
                    removedSubnet);
            }

            ServiceEventSource.Current.Message($"Updated network config:{networkConfig}");
        }

        public async Task UpdateNetworkConfiguration(
            string requestID,
            string networkID,
            VirtualNetworkResponse virtualNetworkResponse)
        {
            using (var tx = RepositoryHelper.GetTransaction())
            {
                string networkKey = NetworkConfiguration.GetNetworkKey(networkID);

                NetworkConfiguration networkConfig = await RepositoryHelper.GetFromReliableDictionary(
                    tx,
                    NetworkConfigReliableCollection,
                    networkKey,
                    LockMode.Update);

                if (networkConfig == null)
                {
                    goto Cleanup;
                }

                await UpdateNetworkConfigurationUntransacted(tx, networkConfig, virtualNetworkResponse);

                Cleanup:
                await RepositoryHelper.RemoveFromReliableDictionary(
                    tx,
                    PendingRequestsReliableCollection,
                    requestID);

                await RepositoryHelper.CommitTransactionAsync(tx);
            }
        }


        public async Task UpdateFailureDetail(
            RequestContext requestContext,
            ErrorInfo failureDetail)
        {
            using (ITransaction tx = RepositoryHelper.GetTransaction())
            {
                switch (requestContext.Type)
                {
                    case RequestType.NetworkContainerRequest:
                    {
                        NCConfiguration ncConfiguration =
                            await RepositoryHelper.GetFromReliableDictionary(
                                tx,
                                NCConfigReliableCollection,
                                requestContext.RequestID,
                                LockMode.Update);

                        if (ncConfiguration != null)
                        {
                            ncConfiguration.FailureDetail = failureDetail;
                            await RepositoryHelper.CreateOrUpdateReliableDictionary(
                                tx,
                                NCConfigReliableCollection,
                                requestContext.RequestID,
                                ncConfiguration);
                        }
                    }
                    break;

                    case RequestType.VirtualNetworkRequest:
                    {
                        NetworkConfiguration networkConfig =
                            await RepositoryHelper.GetFromReliableDictionary(
                                tx,
                                NetworkConfigReliableCollection,
                                requestContext.RequestID,
                                LockMode.Update);

                        if (networkConfig != null)
                        {
                            networkConfig.FailureDetail = failureDetail;
                            await RepositoryHelper.CreateOrUpdateReliableDictionary(
                                tx,
                                NetworkConfigReliableCollection,
                                requestContext.RequestID,
                                networkConfig);
                        }
                    }
                    break;

                    case RequestType.ReservationRequest:
                    {
                        ReservationConfiguration reservationConfiguration =
                            await RepositoryHelper.GetFromReliableDictionary(
                                tx,
                                ReservationReliableCollection,
                                requestContext.RequestID,
                                LockMode.Update);

                        if (reservationConfiguration != null)
                        {
                            reservationConfiguration.FailureDetail = failureDetail;
                            await RepositoryHelper.CreateOrUpdateReliableDictionary(
                                tx,
                                ReservationReliableCollection,
                                requestContext.RequestID,
                                reservationConfiguration);
                        }
                    }
                    break;
                }
                await RepositoryHelper.CommitTransactionAsync(tx);
            }
        }

        public async Task MoveRequestToFailedQueue(
            string requestID,
            int baseRetryinterval,
            ErrorInfo failureDetail)
        {
            using (ITransaction tx = RepositoryHelper.GetTransaction())
            {
                RequestContext requestContext = await RepositoryHelper.GetFromReliableDictionary(
                    tx,
                    PendingRequestsReliableCollection,
                    requestID,
                    LockMode.Update);

                if (requestContext == null)
                {
                    goto Cleanup;
                }

                requestContext.LastRetriedDateTime = DateTime.UtcNow;
                if (requestContext.RetryInterval == null || requestContext.RetryInterval == TimeSpan.Zero)
                {
                    requestContext.RetryInterval = TimeSpan.FromMilliseconds(baseRetryinterval);
                }
                else
                {
                    requestContext.RetryInterval = TimeSpan.FromTicks(requestContext.RetryInterval.Ticks * 2);
                    if (requestContext.RetryInterval.TotalMinutes > 5)
                    {
                        requestContext.RetryInterval = TimeSpan.FromMinutes(5);
                    }
                }

                await RepositoryHelper.CreateOrUpdateReliableDictionary(
                    tx,
                    FailedRequestsReliableCollection,
                    requestID,
                    requestContext);

                Cleanup:
                await RepositoryHelper.RemoveFromReliableDictionary(
                    tx,
                    PendingRequestsReliableCollection,
                    requestID);

                await RepositoryHelper.CommitTransactionAsync(tx);
            }
        }

        public async Task<NCConfiguration> GetNCConfigurationAsync(string ncKey)
        {
            using (var tx = RepositoryHelper.GetTransaction())
            {
                return await RepositoryHelper.GetFromReliableDictionary(
                    tx,
                    NCConfigReliableCollection,
                    ncKey);
            }
        }

        public async Task UpdateNCConfiguration(
            RequestContext requestContext,
            NCConfiguration ncConfig)
        {
            using (var tx = RepositoryHelper.GetTransaction())
            {
                await RepositoryHelper.CreateOrUpdateReliableDictionary(
                    tx,
                    NCConfigReliableCollection,
                    requestContext.RequestID,
                    ncConfig);
                await RepositoryHelper.CommitTransactionAsync(tx);
            }
        }

        public async Task<NCConfiguration> GetNCConfigurationAsync(RequestContext requestContext)
        {
            using (var tx = RepositoryHelper.GetTransaction())
            {
                return await RepositoryHelper.GetFromReliableDictionary(
                    tx,
                    NCConfigReliableCollection,
                    requestContext.RequestID);
            }
        }

        public async Task<NetworkContainerInfo> GetNetworkContainerInfo(string ncKey)
        {
            string networkID = NCConfiguration.GetNetworkIDFromKey(ncKey);
            string networkContainerID = NCConfiguration.GetNetworkContainerIDFromKey(ncKey);

            using (ITransaction tx = RepositoryHelper.GetTransaction())
            {
                NCConfiguration ncConfiguration =
                    await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        this.NCConfigReliableCollection,
                        ncKey);

                if (ncConfiguration == null)
                {
                    return null;
                }

                string reservationKey = ReservationConfiguration.GetReservationKey(
                    networkID,
                    ncConfiguration.ReservationID);
                ReservationConfiguration reservationConfiguration = await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        ReservationReliableCollection,
                        reservationKey);
                if (reservationConfiguration == null)
                {
                    throw new DncNetworkContainerException(
                        networkID,
                        networkContainerID,
                        new ErrorInfo(
                            ErrorCode.ReservationNotFound,
                            $"Reservation {ncConfiguration.ReservationID} for Container not found",
                            ErrorType.NonRetriable));
                }

                string networkKey = NetworkConfiguration.GetNetworkKey(networkID);
                NetworkConfiguration networkConfiguration =
                    await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        NetworkConfigReliableCollection,
                        networkKey);

                if (networkConfiguration == null)
                {
                    throw new DncNetworkContainerException(
                        networkID,
                        networkContainerID,
                        new ErrorInfo(ErrorCode.NetworkNotFound,
                            "Network associated with container not found",
                            ErrorType.NonRetriable));
                }

                string subnetKey = SubnetConfiguration.GetSubnetConfigurationKey(networkID,
                    reservationConfiguration.AllocationRequest.SubnetName);
                SubnetConfiguration subnetConfiguration =
                    await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        SubnetConfigReliableCollection,
                        subnetKey);

                if (subnetConfiguration == null)
                {
                    throw new DncNetworkContainerException(
                        networkID,
                        networkContainerID,
                        new ErrorInfo(ErrorCode.SubnetNotFound,
                        $"Subnet {reservationConfiguration.AllocationRequest.SubnetName} " +
                        $"associated with container not found",
                        ErrorType.NonRetriable));
                }

                List<Tag> tags = new List<Tag>();
                if ((subnetConfiguration.Tags != null) && subnetConfiguration.Tags.Any())
                {
                    tags.AddRange(subnetConfiguration.Tags);
                }

                string nodeKey = NodeConfiguration.GetNodeKey(
                    ncConfiguration.Request.AssociationInfo.NodeID);

                NodeConfiguration nodeConfiguration =
                    await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        NodeConfigReliableCollection,
                        nodeKey);

                if (nodeConfiguration == null)
                {
                    throw new DncNetworkContainerException(
                        networkID,
                        networkContainerID,
                        new ErrorInfo(ErrorCode.NodeNotFound,
                        $"Node {nodeConfiguration.NodeID} associated with container not found",
                        ErrorType.NonRetriable));
                }

                string interfaceID =
                    string.IsNullOrWhiteSpace(ncConfiguration.Request.AssociationInfo.InterfaceID) ?
                        nodeConfiguration.NodeInfo.IPAddresses[0] :
                        ncConfiguration.Request.AssociationInfo.InterfaceID;
                List<string> dnsServers;
                if (networkConfiguration.DnsServers.Count > 0)
                {
                    dnsServers = new List<string>(networkConfiguration.DnsServers);
                }
                else
                {
                    dnsServers = AllocationHelper.GetDNSServerAddressesForSubnet(IPSubnet.Parse(subnetConfiguration.IPSubnet)).Select(item => item.ToString()).ToList();
                }
                ServiceEventSource.Current.Message($"DnsServers:{string.Join(",", dnsServers)}");
                ushort vlanID = 0;
                if(nodeConfiguration.networkVlanIDMap.ContainsKey(networkID))
                {
                    vlanID = nodeConfiguration.networkVlanIDMap[networkID];
                }

                return new NetworkContainerInfo(
                    networkID,
                    nodeConfiguration.NodeID,
                    networkContainerID,
                    interfaceID,
                    subnetConfiguration.AccessToken,
                    new List<string> { reservationConfiguration.AllocatedSubnet.Split(IPSubnet.PrefixDelimiter)[0] },
                    subnetConfiguration.SubnetName,
                    subnetConfiguration.IPSubnet,
                    dnsServers,
                    subnetConfiguration.Gateway,
                    ncConfiguration.AgentVersion,
                    null,
                    vlanID,
                    ncConfiguration.CNSVersion,
                    ncConfiguration.Request.AssociationInfo.ContainerType,
                    ncConfiguration.Request.AssociationInfo.OrchestratorContext,
                    networkConfiguration.CnetSpace,
                    ncConfiguration.State,
                    ncConfiguration.Status,
                    ncConfiguration.LocalIP,
                    tags,
                    networkConfiguration.Version);
            }
        }


        public async Task UpdateNetworkContainerState(
            RequestContext requestContext,
            NetworkContainerState currentState,
            NetworkContainerState nextState,
            Status status,
            bool removeRequestContext)
        {
            using (var tx = RepositoryHelper.GetTransaction())
            {
                if (requestContext == null)
                {
                    return;
                }

                RequestContext retrievedRequestContext = await RepositoryHelper.GetFromReliableDictionary(
                    tx,
                    PendingRequestsReliableCollection,
                    requestContext.RequestID,
                    LockMode.Update);

                if (!requestContext.Equals(retrievedRequestContext))
                {
                    return;
                }

                NCConfiguration ncConfiguration = await RepositoryHelper.GetFromReliableDictionary(
                    tx,
                    NCConfigReliableCollection,
                    requestContext.RequestID,
                    LockMode.Update);
                if (ncConfiguration == null)
                {
                    return;
                }

                if (ncConfiguration.State != currentState)
                {
                    return;
                }

                ncConfiguration.State = nextState;
                ncConfiguration.Status = status;
                await RepositoryHelper.CreateOrUpdateReliableDictionary(
                    tx,
                    NCConfigReliableCollection,
                    requestContext.RequestID,
                    ncConfiguration);

                if (removeRequestContext)
                {
                    await RepositoryHelper.RemoveFromReliableDictionary(
                        tx,
                        PendingRequestsReliableCollection,
                        requestContext.RequestID);
                }
                else
                {
                    retrievedRequestContext.LastRetriedDateTime = DateTime.UtcNow;
                    retrievedRequestContext.RetryInterval = TimeSpan.FromSeconds(0);
                    await RepositoryHelper.CreateOrUpdateReliableDictionary(
                        tx,
                        PendingRequestsReliableCollection,
                        requestContext.RequestID,
                        requestContext);
                    ServiceEventSource.Current.Message($"Updated request context:{requestContext}");
                }

                ServiceEventSource.Current.Message($"Updated NC State:{ncConfiguration.State} " +
                    $"Status:{ncConfiguration.Status}");
                await RepositoryHelper.CommitTransactionAsync(tx);
            }
        }

        public async Task DeleteRequestContext(string requestID)
        {
            using (var tx = RepositoryHelper.GetTransaction())
            {
                await RepositoryHelper.RemoveFromReliableDictionary(
                tx,
                PendingRequestsReliableCollection,
                requestID);
                await RepositoryHelper.CommitTransactionAsync(tx);
            }
        }

        public async Task DeleteNetworkContainer(
            NetworkContainerInfo networkContainerInfo,
            bool deleteRequest)
        {
            using (var tx = RepositoryHelper.GetTransaction())
            {
                string ncKey = NCConfiguration.GetNCKey(
                    networkContainerInfo.NetworkID,
                    networkContainerInfo.NetworkContainerID);

                NCConfiguration ncConfiguration =
                    await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        NCConfigReliableCollection,
                        ncKey);
                ServiceEventSource.Current.Message(
                    $"Retrieved NCConfiguration {ncConfiguration} for delete");
                if (ncConfiguration == null)
                {
                    goto Cleanup;
                }

                string nodeKey = NodeConfiguration.GetNodeKey(networkContainerInfo.NodeID);

                NodeConfiguration nodeConfiguration =
                    await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        NodeConfigReliableCollection,
                        nodeKey);
                if (nodeConfiguration != null)
                {
                    ServiceEventSource.Current.Message(
                        $"Retrieved NodeConfiguration {nodeConfiguration} while deleting NC:{ncKey}");
                    nodeConfiguration.RemoveNetworkContainer(
                        networkContainerInfo.NetworkID,
                        networkContainerInfo.NetworkContainerID);
                    await RepositoryHelper.CreateOrUpdateReliableDictionary(
                        tx,
                        NodeConfigReliableCollection,
                        nodeKey,
                        nodeConfiguration);
                }

                string subnetKey = SubnetConfiguration.GetSubnetConfigurationKey(
                    networkContainerInfo.NetworkID,
                    networkContainerInfo.SubnetName);

                SubnetConfiguration subnetConfiguration =
                    await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        SubnetConfigReliableCollection,
                        subnetKey);

                if (subnetConfiguration != null)
                {
                    ServiceEventSource.Current.Message(
                        $"Retrieved SubnetConfiguration {subnetConfiguration} while deleting NC:{ncKey}");
                    if (ncConfiguration.ImplicitReservation)
                    {
                        ServiceEventSource.Current.Message(
                            $"Removing reservation from {ncConfiguration.ReservationID} from " +
                            $"{subnetConfiguration} while deleting NC:{ncKey}");
                        subnetConfiguration.RemoveReservation(ncConfiguration.ReservationID);

                        await RepositoryHelper.CreateOrUpdateReliableDictionary(
                            tx,
                            SubnetConfigReliableCollection,
                            subnetKey,
                            subnetConfiguration);
                    }
                }

                string reservationKey = ReservationConfiguration.GetReservationKey(
                    networkContainerInfo.NetworkID,
                    ncConfiguration.ReservationID);
                ReservationConfiguration reservationConfiguration =
                    await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        ReservationReliableCollection,
                        reservationKey);
                if (subnetConfiguration != null)
                {
                    if (ncConfiguration.ImplicitReservation)
                    {
                        ServiceEventSource.Current.Message(
                            $"Deleting reservation  {reservationConfiguration} from " +
                            $"while deleting NC:{ncKey}");

                        await RepositoryHelper.RemoveFromReliableDictionary(
                            tx,
                            ReservationReliableCollection,
                            reservationKey);
                    }
                    else
                    {
                        ServiceEventSource.Current.Message(
                            $"Setting NC association ID in  {reservationConfiguration} to null");
                        reservationConfiguration.AssociatedNCID = null;
                        await RepositoryHelper.CreateOrUpdateReliableDictionary(
                            tx,
                            ReservationReliableCollection,
                            reservationKey,
                            reservationConfiguration);
                    }
                }

                ServiceEventSource.Current.Message(
                    $"Deleting NCConfigurtaion  {ncConfiguration}");

                await RepositoryHelper.RemoveFromReliableDictionary(
                            tx,
                            NCConfigReliableCollection,
                            ncKey);

                Cleanup:

                if(deleteRequest)
                {
                    ServiceEventSource.Current.Message(
                        $"Deleting RequestContext with ID  {ncKey}");
                    await RepositoryHelper.RemoveFromReliableDictionary(
                        tx,
                        PendingRequestsReliableCollection,
                        ncKey);
                }
                await RepositoryHelper.CommitTransactionAsync(tx);
            }
        }

        public async Task<NodeConfiguration> GetNodeConfiguration(
            string nodeID)
        {
            using (var tx = RepositoryHelper.GetTransaction())
            {
                return await RepositoryHelper.GetFromReliableDictionary(
                    tx,
                    NodeConfigReliableCollection,
                    NodeConfiguration.GetNodeKey(nodeID));
            }
        }

        public async Task<SubnetConfiguration> GetSubnetConfiguration(
            string subnetKey)
        {
            using (var tx = RepositoryHelper.GetTransaction())
            {
                return await RepositoryHelper.GetFromReliableDictionary(
                    tx,
                    SubnetConfigReliableCollection,
                    subnetKey);
            }
        }

        public async Task UpdateSubnetConfiguration(
            string subnetKey,
            SubnetConfiguration subnetConfig)
        {
            using (var tx = RepositoryHelper.GetTransaction())
            {
                await RepositoryHelper.CreateOrUpdateReliableDictionary(
                    tx,
                    SubnetConfigReliableCollection,
                    subnetKey,
                    subnetConfig);
                await RepositoryHelper.CommitTransactionAsync(tx);
            }
        }

        public async Task<ReservationConfiguration> GetReservationConfiguration(
            string reservationKey)
        {
            using (var tx = RepositoryHelper.GetTransaction())
            {
                return await RepositoryHelper.GetFromReliableDictionary(
                    tx,
                    ReservationReliableCollection,
                    reservationKey);
            }
        }

        public async Task UpdateReservationConfiguration(
            string reservationKey,
            ReservationConfiguration reservationConfig)
        {
            using (var tx = RepositoryHelper.GetTransaction())
            {
                await RepositoryHelper.CreateOrUpdateReliableDictionary(
                    tx,
                    ReservationReliableCollection,
                    reservationKey,
                    reservationConfig);
                await RepositoryHelper.CommitTransactionAsync(tx);
            }
        }

        public async Task<NCConfiguration> GetNetworkContainerConfiguration(
            string networkID,
            string networkContainerID)
        {
            using (var tx = RepositoryHelper.GetTransaction())
            {
                return await RepositoryHelper.GetFromReliableDictionary(
                    tx,
                    NCConfigReliableCollection,
                    NCConfiguration.GetNCKey(networkID, networkContainerID));
            }
        }

        public async Task SetNetworkContainerLocalIP(
            RequestContext requestContext,
            string localIP)
        {
            using (var tx = RepositoryHelper.GetTransaction())
            {
                RequestContext retrievedRequestContext = await RepositoryHelper.GetFromReliableDictionary(
                    tx,
                    PendingRequestsReliableCollection,
                    requestContext.RequestID,
                    LockMode.Update);

                if (requestContext == null || !requestContext.Equals(retrievedRequestContext))
                {
                    return;
                }

                NCConfiguration ncConfiguration = await RepositoryHelper.GetFromReliableDictionary(
                    tx,
                    NCConfigReliableCollection,
                    requestContext.RequestID,
                    LockMode.Update);
                if (ncConfiguration == null)
                {
                    return;
                }

                ncConfiguration.LocalIP = localIP;
                await RepositoryHelper.CreateOrUpdateReliableDictionary(
                    tx,
                    NCConfigReliableCollection,
                    requestContext.RequestID,
                    ncConfiguration);

                ServiceEventSource.Current.Message($"Set NC LocalIP:{ncConfiguration.LocalIP} ");
                await RepositoryHelper.CommitTransactionAsync(tx);
            }
        }
    }
}