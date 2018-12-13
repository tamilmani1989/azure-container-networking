using DNCService;
using DNCService.API;
using Microsoft.ServiceFabric.Data;
using Microsoft.ServiceFabric.Data.Collections;
using NetworkManager.DataModel;
using NMAgent.Data;
using System;
using System.Collections.Generic;
using System.Linq;
using System.Net;
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
            ErrorInfo failureDetail,
            bool retriableError = true)
        {
            using (var tx = RepositoryHelper.GetTransaction())
            {
                switch (requestContext.Type)
                {
                    case RequestType.NetworkContainerRequest:
                    {
                        NCConfiguration ncConfiguration = await RepositoryHelper.GetFromReliableDictionary(
                            tx,
                            NCConfigReliableCollection,
                            requestContext.RequestID,
                            LockMode.Update);

                        if (ncConfiguration != null)
                        {
                            if (!retriableError)
                            {
                                ncConfiguration.Status = Status.Failed;
                            }
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
                        NetworkConfiguration networkConfig = await RepositoryHelper.GetFromReliableDictionary(
                            tx,
                            NetworkConfigReliableCollection,
                            requestContext.RequestID,
                            LockMode.Update);

                        if (networkConfig != null)
                        {
                            if (!retriableError)
                            {
                                networkConfig.Status = Status.Failed;
                            }
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
                        ReservationConfiguration reservationConfiguration = await RepositoryHelper.GetFromReliableDictionary(
                            tx,
                            ReservationReliableCollection,
                            requestContext.RequestID,
                            LockMode.Update);

                        if (reservationConfiguration != null)
                        {
                            if (!retriableError)
                            {
                                reservationConfiguration.Status = Status.Failed;
                            }
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
            uint baseRetryinterval,
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

        public async Task<NCConfiguration> GetNetworkContainerConfiguration(
            string ncKey)
        {
            using (var tx = RepositoryHelper.GetTransaction())
            {
                return await RepositoryHelper.GetFromReliableDictionary(
                    tx,
                    NCConfigReliableCollection,
                    ncKey);
            }
        }

        public async Task<NetworkConfiguration> GetNetworkConfiguration(
            string networkKey,
            ITransaction tx,
            LockMode lockMode = LockMode.Default)
        {
            return await RepositoryHelper.GetFromReliableDictionary(
                tx,
                NetworkConfigReliableCollection,
                networkKey,
                lockMode);
        }

        public async Task<NCConfiguration> GetNetworkContainerConfiguration(
            string ncKey,
            ITransaction tx,
            LockMode lockMode = LockMode.Default)
        {
            return await RepositoryHelper.GetFromReliableDictionary(
                tx,
                NCConfigReliableCollection,
                ncKey,
                lockMode);
        }

        public async Task<NetworkContainerInfo> GetNetworkContainerInfo(
            RequestContext requestContext)
        {
            string ncKey = requestContext.RequestID;
            string networkID = NCConfiguration.GetNetworkIDFromKey(ncKey);
            string networkContainerID = NCConfiguration.GetNetworkContainerIDFromKey(ncKey);

            NodeConfiguration nodeConfiguration;
            using (var tx = RepositoryHelper.GetTransaction())
            {
                string nodeKey = NodeConfiguration.GetNodeKey(requestContext.NodeID);
                nodeConfiguration = await RepositoryHelper.GetFromReliableDictionary(
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
            }

            NetworkConfiguration networkConfiguration;
            using (var tx = RepositoryHelper.GetTransaction())
            {
                string networkKey = NetworkConfiguration.GetNetworkKey(networkID);
                networkConfiguration = await RepositoryHelper.GetFromReliableDictionary(
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
            }

            SubnetConfiguration subnetConfiguration;
            using (var tx = RepositoryHelper.GetTransaction())
            {
                ServiceEventSource.Current.Message($"GetNetworkContainerInfo subnetname: {requestContext.SubnetName}");
                string subnetKey = SubnetConfiguration.GetSubnetConfigurationKey(
                    networkID,
                    requestContext.SubnetName);
                subnetConfiguration = await RepositoryHelper.GetFromReliableDictionary(
                    tx,
                    SubnetConfigReliableCollection,
                    subnetKey);
                if (subnetConfiguration == null)
                {
                    throw new DncNetworkContainerException(
                        networkID,
                        networkContainerID,
                        new ErrorInfo(ErrorCode.SubnetNotFound,
                        $"Subnet {requestContext.SubnetName} " +
                        $"associated with container not found",
                        ErrorType.NonRetriable));
                }
            }

            NCConfiguration ncConfiguration;
            using (var tx = RepositoryHelper.GetTransaction())
            {
                ncConfiguration = await RepositoryHelper.GetFromReliableDictionary(
                    tx,
                    NCConfigReliableCollection,
                    ncKey);
                if (ncConfiguration == null)
                {
                    return null;
                }
            }

            ReservationConfiguration reservationConfiguration;
            using (var tx = RepositoryHelper.GetTransaction())
            {
                string reservationKey = ReservationConfiguration.GetReservationKey(
                    networkID,
                    ncConfiguration.ReservationID);
                reservationConfiguration = await RepositoryHelper.GetFromReliableDictionary(
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
            }

            List<Tag> tags = new List<Tag>();
            if ((subnetConfiguration.Tags != null) && subnetConfiguration.Tags.Any())
            {
                tags.AddRange(subnetConfiguration.Tags);
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
                networkConfiguration.VnetVersion);
        }

        public async Task UpdateNodeConfigurationUntransacted(
            string nodeKey,
            NodeConfiguration nodeConfiguration,
            ITransaction tx)
        {
            await RepositoryHelper.CreateOrUpdateReliableDictionary(
                tx,
                NodeConfigReliableCollection,
                nodeKey,
                nodeConfiguration);
        }

        public async Task UpdateNetworkContainerConfigUntransacted(
            string ncKey,
            NCConfiguration ncConfig,
            ITransaction tx)
        {
            await RepositoryHelper.CreateOrUpdateReliableDictionary(
                tx,
                NCConfigReliableCollection,
                ncKey,
                ncConfig);
        }

        public async Task UpdateNetworkContainerStateUntransacted(
            NCConfiguration ncConfig,
            RequestContext requestContext,
            NetworkContainerState currentState,
            NetworkContainerState nextState,
            Status status,
            bool removeRequestContext,
            ITransaction tx)
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

            if ((ncConfig == null) ||
                (ncConfig.State != currentState))
            {
                return;
            }

            ncConfig.State = nextState;
            ncConfig.Status = status;
            await RepositoryHelper.CreateOrUpdateReliableDictionary(
                tx,
                NCConfigReliableCollection,
                requestContext.RequestID,
                ncConfig);

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
                    retrievedRequestContext.RequestID,
                    retrievedRequestContext);
                ServiceEventSource.Current.Message($"Updated request context:{retrievedRequestContext}");
            }

            ServiceEventSource.Current.Message($"Updated NC State: {ncConfig.State} " +
                $"Status: {ncConfig.Status} ncKey: {retrievedRequestContext.RequestID}");
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

        public async Task DeleteRequestContextUntransacted(
            string requestID,
            ITransaction tx)
        {
            await RepositoryHelper.RemoveFromReliableDictionary(
                tx,
                PendingRequestsReliableCollection,
                requestID);
        }

        public async Task DeleteReservationUntransacted(
            ReservationConfiguration reservationConfiguration,
            RequestContext requestContext,
            ITransaction tx)
        {
            string subnetKey = SubnetConfiguration.GetSubnetConfigurationKey(
                requestContext.NetworkID,
                requestContext.SubnetName);
            SubnetConfiguration subnetConfiguration = await GetSubnetConfiguration(
                subnetKey,
                tx,
                LockMode.Update);
            if (subnetConfiguration != null)
            {
                ServiceEventSource.Current.Message($"Retrieved SubnetConfiguration {subnetConfiguration} " +
                    $"while deleting reservation: {reservationConfiguration.ReservationID}");

                ServiceEventSource.Current.Message(
                    $"Deleting reservation {reservationConfiguration.ReservationID} from SubnetConfiguration {subnetConfiguration}");
                subnetConfiguration.RemoveReservation(reservationConfiguration.ReservationID);
                await UpdateSubnetConfigurationUntransacted(
                    subnetKey,
                    subnetConfiguration,
                    tx);

                ServiceEventSource.Current.Message(
                    $"Deleting reservation {reservationConfiguration}");
                await RepositoryHelper.RemoveFromReliableDictionary(
                    tx,
                    ReservationReliableCollection,
                    requestContext.RequestID);
            }
            else
            {
                ServiceEventSource.Current.Message($"No SubnetConfiguration found " +
                    $"while deleting reservation: {reservationConfiguration.ReservationID}");
                throw new DncManagementException(
                    new ErrorInfo(
                        ErrorCode.SubnetConfigurationNotFound,
                        $"No SubnetConfiguration found while deleting reservation {reservationConfiguration.ReservationID}",
                        ErrorType.NonRetriable));
            }

            if (!string.IsNullOrWhiteSpace(reservationConfiguration.AllocationRequest.NodeConstraint))
            {
                NodeConfiguration nodeConfig = await GetNodeConfiguration(
                    reservationConfiguration.AllocationRequest.NodeConstraint,
                    tx,
                    LockMode.Update);
                if (nodeConfig == null)
                {
                    throw new DncManagementException(
                        new ErrorInfo(
                            ErrorCode.NodeNotFound,
                            $"No NodeConfiguration found while deleting reservation {reservationConfiguration.ReservationID}" +
                            $"for reservation config {reservationConfiguration}",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }
                nodeConfig.RemoveConstrainedReservation(reservationConfiguration.ReservationID);
                await UpdateNodeConfigurationUntransacted(
                    NodeConfiguration.GetNodeKey(reservationConfiguration.AllocationRequest.NodeConstraint),
                    nodeConfig,
                    tx);
            }
        }

        public async Task DeleteNetworkContainerUntransacted(
            NodeConfiguration nodeConfiguration,
            SubnetConfiguration subnetConfiguration,
            ReservationConfiguration reservationConfiguration,
            NCConfiguration ncConfiguration,
            RequestContext requestContext,
            bool deleteRequest,
            ITransaction tx)
        {
            ServiceEventSource.Current.Message($"Retrieved NCConfiguration: {ncConfiguration} for delete");
            string ncKey = requestContext.RequestID;
            string networkContainerID = NCConfiguration.GetNetworkContainerIDFromKey(ncKey);
            if (nodeConfiguration != null)
            {
                ServiceEventSource.Current.Message(
                    $"Retrieved NodeConfiguration: {nodeConfiguration} while deleting NC: {ncKey}");
                nodeConfiguration.RemoveNetworkContainer(requestContext.NetworkID, networkContainerID);

                if (ncConfiguration.ImplicitReservation)
                {
                    nodeConfiguration.constrainedReservations.Remove(reservationConfiguration.ReservationID);
                }

                await UpdateNodeConfigurationUntransacted(
                    NodeConfiguration.GetNodeKey(requestContext.NodeID),
                    nodeConfiguration,
                    tx);
            }

            if (subnetConfiguration != null)
            {
                ServiceEventSource.Current.Message($"Retrieved SubnetConfiguration: {subnetConfiguration} " +
                    $"while deleting NC: {ncKey}");
                string subnetKey = SubnetConfiguration.GetSubnetConfigurationKey(
                    requestContext.NetworkID,
                    requestContext.SubnetName);
                string reservationKey = ReservationConfiguration.GetReservationKey(
                    requestContext.NetworkID,
                    ncConfiguration.ReservationID);
                if (ncConfiguration.ImplicitReservation)
                {
                    ServiceEventSource.Current.Message($"Removing reservation: {ncConfiguration.ReservationID} from " +
                        $"{subnetConfiguration} while deleting NC: {ncKey}");
                    subnetConfiguration.RemoveReservation(ncConfiguration.ReservationID);
                    await UpdateSubnetConfigurationUntransacted(
                        subnetKey,
                        subnetConfiguration,
                        tx);

                    ServiceEventSource.Current.Message($"Deleting reservation: {reservationConfiguration} " +
                        $"while deleting NC: {ncKey}");
                    await RepositoryHelper.RemoveFromReliableDictionary(
                        tx,
                        ReservationReliableCollection,
                        reservationKey);
                }
                else
                {
                    ServiceEventSource.Current.Message($"Setting associated NCID in {reservationConfiguration} to null");
                    reservationConfiguration.AssociatedNCID = null;
                    await UpdateReservationConfigurationUntransacted(
                        reservationKey,
                        reservationConfiguration,
                        tx);
                }
            }

            ServiceEventSource.Current.Message($"Deleting NCConfiguration: {ncConfiguration}");
            await RepositoryHelper.RemoveFromReliableDictionary(
                tx,
                NCConfigReliableCollection,
                ncKey);

            if (deleteRequest)
            {
                ServiceEventSource.Current.Message($"Deleting RequestContext with ID: {ncKey}");
                await RepositoryHelper.RemoveFromReliableDictionary(
                    tx,
                    PendingRequestsReliableCollection,
                    ncKey);
            }
        }

        public async Task<NodeConfiguration> GetNodeConfiguration(
            string nodeID,
            ITransaction tx,
            LockMode lockMode = LockMode.Default)
        {
            return await RepositoryHelper.GetFromReliableDictionary(
                tx,
                NodeConfigReliableCollection,
                NodeConfiguration.GetNodeKey(nodeID),
                lockMode);
        }

        public async Task<SubnetConfiguration> GetSubnetConfiguration(
            string subnetKey,
            ITransaction tx,
            LockMode lockMode = LockMode.Default)
        {
            return await RepositoryHelper.GetFromReliableDictionary(
                tx,
                SubnetConfigReliableCollection,
                subnetKey,
                lockMode);
        }

        public async Task UpdateSubnetConfigurationUntransacted(
            string subnetKey,
            SubnetConfiguration subnetConfig,
            ITransaction tx)
        {
            await RepositoryHelper.CreateOrUpdateReliableDictionary(
                tx,
                SubnetConfigReliableCollection,
                subnetKey,
                subnetConfig);
        }

        public async Task<ReservationConfiguration> GetReservationConfiguration(
            string reservationKey,
            ITransaction tx,
            LockMode lockMode = LockMode.Default)
        {
            return await RepositoryHelper.GetFromReliableDictionary(
                tx,
                ReservationReliableCollection,
                reservationKey,
                lockMode);
        }

        public async Task UpdateReservationConfigurationUntransacted(
            string reservationKey,
            ReservationConfiguration reservationConfig,
            ITransaction tx)
        {
            await RepositoryHelper.CreateOrUpdateReliableDictionary(
                tx,
                ReservationReliableCollection,
                reservationKey,
                reservationConfig);
        }

        public async Task SetNetworkContainerLocalIPUntransacted(
            NCConfiguration ncConfiguration,
            RequestContext requestContext,
            string localIP,
            ITransaction tx)
        {
            if (requestContext == null)
            {
                return;
            }

            RequestContext retrievedRequestContext = await RepositoryHelper.GetFromReliableDictionary(
                tx,
                PendingRequestsReliableCollection,
                requestContext.RequestID);
            if (!requestContext.Equals(retrievedRequestContext))
            {
                return;
            }

            ncConfiguration.LocalIP = localIP;
            await RepositoryHelper.CreateOrUpdateReliableDictionary(
                tx,
                NCConfigReliableCollection,
                requestContext.RequestID,
                ncConfiguration);

            ServiceEventSource.Current.Message($"Set LocalIP: {ncConfiguration.LocalIP} for " +
                $"NC: {requestContext.RequestID}");
        }
    }
}