//--------------------------------------------------------------------------
// <copyright file="DNCService.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
// Implements the DNC service.
// </summary>
//--------------------------------------------------------------------------

using System;
using System.Collections.Generic;
using System.Fabric;
using System.Linq;
using System.Threading;
using System.Threading.Tasks;
using Microsoft.ServiceFabric.Data.Collections;
using Microsoft.ServiceFabric.Services.Communication.Runtime;
using Microsoft.ServiceFabric.Services.Runtime;
using Microsoft.AspNetCore.Hosting;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.ServiceFabric.Data;
using DNCService.Interfaces;
using DNCService.API;
using System.IO;
using Microsoft.ServiceFabric.Data.Notifications;
using Microsoft.ServiceFabric.Services.Communication.AspNetCore;
using Microsoft.ServiceFabric.Services.Remoting.Runtime;
using NetworkManager;
using NMAgent;
using Cns;
using NetworkManager.DataModel;
using DNCService.Rest;

namespace DNCService
{
    /// <summary>
    /// An instance of this class is created for each service replica by the Service Fabric runtime.
    /// </summary>
    internal sealed class DNCService : StatefulService, IDNCService
    {
        /// <summary>
        /// Nam eof the application.
        /// </summary>
        private string applicationName;

        /// <summary>
        /// Replica instance Id.
        /// </summary>
        private long replicaInstanceid;

        /// <summary>
        /// Dnc configuration.
        /// </summary>
        public static DncConfiguration DncConfigurationInstance;

        /// <summary>
        /// Handle to the host agent client.
        /// </summary>
        private HostAgentClient hostAgentClient;

        /// <summary>
        /// Handle to the cns client.
        /// </summary>
        private CnsClient cnsClient;

        /// <summary>
        /// Handle to the request watcher.
        /// </summary>
        private RequestWatcher requestWatcher;

        /// <summary>
        /// Handle to the repository network manager.
        /// </summary>
        private NetworkRepositoryManager networkRepositoryManager;

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="context">Supply the stateful service context.</param>
        public DNCService(StatefulServiceContext context)
            : base(context)
        {
            ServiceEventSource.Current.Message("Start Init");
            string codePath = context.CodePackageActivationContext.GetConfigurationPackageObject("config").Path;
            ServiceEventSource.Current.Message($"dncSettingsFile:{codePath}");
            DncConfigurationInstance = DncConfiguration.Deserialize(Path.Combine(codePath, "settings.json"));
            this.applicationName = context.CodePackageActivationContext.ApplicationName;
            this.replicaInstanceid = context.ReplicaOrInstanceId;
            this.StateManager.StateManagerChanged += this.OnStateManagerChangedHandler;
            
            hostAgentClient = new HostAgentClient(DncConfigurationInstance);
            cnsClient = new CnsClient(DncConfigurationInstance);
            requestWatcher = new RequestWatcher(hostAgentClient, cnsClient);
            ServiceEventSource.Current.Message("Completed Init");
        }

        /// <summary>
        /// Optional override to create listeners (like tcp, http) for this service instance.
        /// </summary>
        /// <returns>The collection of listeners.</returns>
        protected override IEnumerable<ServiceReplicaListener> CreateServiceReplicaListeners()
        {
            List<ServiceReplicaListener> serviceReplicaListeners = new List<ServiceReplicaListener> (
                this.CreateServiceRemotingReplicaListeners());
            serviceReplicaListeners.Add(new ServiceReplicaListener(
                    serviceContext =>
                        new KestrelCommunicationListener(
                            serviceContext, "ServiceEndpoint",
                            (url, listener) =>
                            {
                                ServiceEventSource.Current.ServiceMessage(serviceContext, $"Starting Kestrel on {url}");

                                return new WebHostBuilder()
                                    .UseKestrel()
                                    .ConfigureServices(
                                        services => services
                                            .AddSingleton<StatefulServiceContext>(serviceContext)
                                            .AddSingleton<IReliableStateManager>(this.StateManager))
                                    .UseContentRoot(Directory.GetCurrentDirectory())
                                    .UseStartup<Startup>()
                                    .UseApplicationInsights()
                                    .UseServiceFabricIntegration(listener, ServiceFabricIntegrationOptions.UseUniqueServiceUrl)
                                    .UseUrls(url)
                                    .Build();
                            })));
            return serviceReplicaListeners.ToArray();
        }

        /// <summary>
        /// State rebuild handler.
        /// </summary>
        /// <param name="sender">Supply the sender.</param>
        /// <param name="e">Supply the event change aruments.</param>
        public void OnStateManagerChangedHandler(object sender, NotifyStateManagerChangedEventArgs e)
        {
            if (e.Action == NotifyStateManagerChangedAction.Rebuild)
            {
                return;
            }

            this.ProcessStateManagerSingleEntityNotification(e);
        }

        /// <summary>
        /// State change notification handler.
        /// </summary>
        /// <param name="e">Supply the event chnage arguments.</param>
        private void ProcessStateManagerSingleEntityNotification(NotifyStateManagerChangedEventArgs e)
        {
            var operation = e as NotifyStateManagerSingleEntityChangedEventArgs;

            if (operation.Action == NotifyStateManagerChangedAction.Add)
            {
                if (operation.ReliableState is IReliableDictionary<string, RequestContext>)
                {
                    var dictionary = (IReliableDictionary<string, RequestContext>)operation.ReliableState;
                    if (dictionary.Name.AbsoluteUri.Contains(Constants.PendingRequestsTableName))
                    {
                        dictionary.DictionaryChanged += requestWatcher.OnDictionaryChangedHandler;
                    }
                }
            }
        }

        /// <summary>
        /// This is the main entry point for your service replica.
        /// This method executes when this replica of your service becomes primary and has write status.
        /// </summary>
        /// <param name="cancellationToken">Canceled when Service Fabric needs to shut down this service replica.</param>
        protected override async Task RunAsync(CancellationToken cancellationToken)
        {
            ServiceEventSource.Current.Message("Run async");
            try
            {
                RepositoryHelper.Init(this.StateManager, cancellationToken, new TimeSpan(0, 0, 30));
                networkRepositoryManager = new NetworkRepositoryManager(DncConfigurationInstance);
                requestWatcher.Init(networkRepositoryManager);
                requestWatcher.Start();

                RequestResetMonitor requestResetMonitor = new RequestResetMonitor(
                    networkRepositoryManager,
                    cancellationToken);
                requestResetMonitor.StartAsync();
                NetworkRefreshMonitor networkRefreshMonitor = new NetworkRefreshMonitor(
                    this.StateManager,
                    cancellationToken,
                    hostAgentClient,
                    cnsClient,
                    networkRepositoryManager);
                networkRefreshMonitor.StartAsync();
                ServiceEventSource.Current.Message("Completed initializing monitors");
                while (true)
                {
                    cancellationToken.ThrowIfCancellationRequested();
                    await Task.Delay(TimeSpan.FromSeconds(1), cancellationToken);
                }

            }
            catch (Exception exception)
            {

                ServiceEventSource.Current.Message("RunAsync failed with exception:{0}", exception);
                throw;
            }
        }

        /// <summary>
        /// Join subnet.
        /// </summary>
        /// <param name="networkID">Supply the ID of the network.</param>
        /// <param name="subnetName">Supply the name of the subnet.</param>
        /// <param name="subnetRequest">Supply the subnet request.</param>
        /// <param name="apiVersion">Supply the API version.</param>
        /// <returns></returns>
        async Task IDNCService.JoinSubnet(
            string networkID, 
            string subnetName,
            SubnetRequest subnetRequest, 
            string apiVersion)
        {
            try
            { 
            IReliableDictionary<string, SubnetConfiguration> subnetConfigReliableCollection =
                RepositoryHelper.GetReliableDictionaryAsync<string, SubnetConfiguration>(
                Constants.SubnetConfigurationTableName).Result;
            SubnetConfiguration subnetConfiguration;
                using (ITransaction tx = RepositoryHelper.GetTransaction())
                {
                    string subnetKey = SubnetConfiguration.GetSubnetConfigurationKey(networkID,
                        subnetName);
                    subnetConfiguration = await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        subnetConfigReliableCollection,
                        subnetKey);

                    if (subnetConfiguration == null)
                    {
                        throw new DncNetworkManagementException(
                            networkID,
                            new ErrorInfo(
                                ErrorCode.SubnetNotFound,
                                $"Subnet {subnetRequest.SubnetName} not found"));
                    }

                    subnetConfiguration.AccessToken = subnetRequest.AuthenticationToken;
                    subnetConfiguration.Joined = true;
                    await RepositoryHelper.CreateOrUpdateReliableDictionary(
                                               tx,
                                               subnetConfigReliableCollection,
                                               subnetKey,
                                               subnetConfiguration);

                    await RepositoryHelper.CommitTransactionAsync(tx);

                }
            }
            catch (Exception exception)
            {
                ServiceEventSource.Current.Message(
                    $"Failed to join subnet {subnetName} in network {networkID} with exception:{exception}");
                throw;
            }
        }

        /// <summary>
        /// Get subnet configuration.
        /// </summary>
        /// <param name="networkID">Supply the ID of the network.</param>
        /// <param name="subnetName">Supply the subnet name.</param>
        /// <returns>Subnet configuration.</returns>
        async Task<SubnetResponse> IDNCService.GetSubnetConfigurationResponseAsync(string networkID, string subnetName)
        {
            IReliableDictionary<string, NetworkConfiguration> networkConfigReliableCollection =
                RepositoryHelper.GetReliableDictionaryAsync<string, NetworkConfiguration>(
                Constants.NetworkConfigurationTableName).Result;
            IReliableDictionary<string, SubnetConfiguration> subnetConfigReliableCollection =
                RepositoryHelper.GetReliableDictionaryAsync<string, SubnetConfiguration>(
                Constants.SubnetConfigurationTableName).Result;
            SubnetConfiguration subnetConfiguration;
            NetworkConfiguration networkConfig;
            try
            {
                List<ReservationConfiguration> reservationConfigurations = new List<ReservationConfiguration>();
                using (ITransaction tx = RepositoryHelper.GetTransaction())
                {
                    string networkKey = NetworkConfiguration.GetNetworkKey(networkID);

                    networkConfig = await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        networkConfigReliableCollection,
                        networkKey);
                    ServiceEventSource.Current.Message($"Retrieved network config.{networkConfig}");
                    string subnetKey = SubnetConfiguration.GetSubnetConfigurationKey(networkID, subnetName);
                    subnetConfiguration = await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        subnetConfigReliableCollection,
                        subnetKey);
                    foreach(string reservationID in subnetConfiguration.Reservations.Keys)
                    {
                        string reservationKey = ReservationConfiguration.GetReservationKey(
                            networkID,
                            reservationID);
                        ReservationConfiguration reservationConfiguration = await
                            RepositoryHelper.GetFromReliableDictionary(
                                tx,
                                networkRepositoryManager.ReservationReliableCollection,
                                reservationKey);
                        if(reservationConfiguration != null)
                        {
                            reservationConfigurations.Add(reservationConfiguration);
                        }
                    }
                    ServiceEventSource.Current.Message($"Retrieved network config.{subnetConfiguration}");
                }

                return RepoToApiContractTranslators.GetSubnetConfigurationResponse(
                    networkConfig,
                    subnetConfiguration,
                    reservationConfigurations);
            }
            catch(Exception exception)
            {
                ServiceEventSource.Current.Message(
                    $"Failed to get subnet {subnetName} in network {networkID} with exception:{exception}");
                throw;
            }
        }

        /// <summary>
        /// Create or update network.
        /// </summary>
        /// <param name="networkID">Supply the network ID.</param>
        /// <param name="networkInfoRequest">Supply the network request.</param>
        /// <param name="apiVersion">Supply the API version.</param>
        /// <returns></returns>
        async Task IDNCService.CreateOrUpdateNetworkAsync(
            string networkID,
            NetworkRequest networkInfoRequest,
            string apiVersion)
        {
            ServiceEventSource.Current.Message("CreateOrUpdateNetwork:{0}, Request:{1}",
                networkID,
                networkInfoRequest);
            try
            {
                IReliableDictionary<string, NetworkConfiguration> networkConfigReliableCollection =
                    RepositoryHelper.GetReliableDictionaryAsync<string, NetworkConfiguration>(
                    Constants.NetworkConfigurationTableName).Result;
                IReliableDictionary<string, RequestContext> pendingWorkItemsReliableCollection =
                    RepositoryHelper.GetReliableDictionaryAsync<string, RequestContext>(
                    Constants.PendingRequestsTableName).Result;

                NetworkConfiguration networkConfig;
                using (ITransaction tx = RepositoryHelper.GetTransaction())
                {
                    string networkKey = NetworkConfiguration.GetNetworkKey(networkID);

                    networkConfig = await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        networkConfigReliableCollection,
                        networkKey);

                    if (networkConfig == null)
                    {

                        networkConfig = new NetworkConfiguration(
                            "",
                            networkID,
                            networkInfoRequest,
                            apiVersion);
                    }

                    RequestContext requestContext = new RequestContext(
                        networkKey,
                        networkID,
                        "",
                        RequestType.VirtualNetworkRequest);

                    await RepositoryHelper.CreateOrUpdateReliableDictionary(tx, pendingWorkItemsReliableCollection, networkKey, requestContext);
                    await RepositoryHelper.CreateOrUpdateReliableDictionary(tx, networkConfigReliableCollection, networkKey, networkConfig);
                    
                    await RepositoryHelper.CommitTransactionAsync(tx);
                    ServiceEventSource.Current.Message("Completed persisting request for network {0}", networkID);
                }
            }
            catch(Exception exception)
            {
                ServiceEventSource.Current.Message("Faield to process network {0} request {1} due to excpetion {2}",
                    networkID,
                    networkInfoRequest,
                    exception);
                throw;
            }
        }

        /// <summary>
        /// Create or update node.
        /// </summary>
        /// <param name="nodeID">Supply the ID of the node.</param>
        /// <param name="nodeInfoRequest">Supply the node request.</param>
        /// <param name="apiVersion">Supply the API version.</param>
        /// <returns></returns>
        async Task IDNCService.CreateOrUpdateNodeAsync(
            string nodeID,
            NodeRequest nodeInfoRequest,
            string apiVersion)
        {
            ServiceEventSource.Current.Message("CreateOrUpdateNode ID:{0} Request:{1}", nodeID, nodeInfoRequest);

            try
            {
                IReliableDictionary<string, NetworkManager.DataModel.NodeConfiguration> nodeConfigReliableCollection =
                    RepositoryHelper.GetReliableDictionaryAsync<string, NetworkManager.DataModel.NodeConfiguration>(
                    Constants.NodeTableName).Result;

                NetworkManager.DataModel.NodeConfiguration nodeConfig;
                using (ITransaction tx = RepositoryHelper.GetTransaction())
                {
                    string nodeKey = NetworkManager.DataModel.NodeConfiguration.GetNodeKey(nodeID);

                    nodeConfig = await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        nodeConfigReliableCollection,
                        nodeKey);

                    if (nodeConfig != null)
                    {
                        return;
                    }

                    await cnsClient.SetOrchestratorType(nodeInfoRequest);
                    nodeConfig = new NetworkManager.DataModel.NodeConfiguration(
                        nodeID,
                        apiVersion,
                        nodeInfoRequest);
                    await RepositoryHelper.CreateOrUpdateReliableDictionary(
                        tx,
                        nodeConfigReliableCollection,
                        NetworkManager.DataModel.NodeConfiguration.GetNodeKey(nodeID),
                        nodeConfig);
                    await RepositoryHelper.CommitTransactionAsync(tx);
                    ServiceEventSource.Current.Message("Updated node ID:{0} successfully", nodeID);
                }

                return;
            }
            catch (Exception exception)
            {
                ServiceEventSource.Current.Message("Failed to process node {0} request {1} due to excpetion {2}",
                    nodeID,
                    nodeInfoRequest,
                    exception);
                throw;
            }
        }

        /// <summary>
        /// Create or update network container.
        /// </summary>
        /// <param name="networkID">Supply the ID of the network.</param>
        /// <param name="nodeID">Supply the ID of the network container.</param>
        /// <param name="ncRequest">Supply the network container request.</param>
        /// <param name="apiVersion">Supply the API version.</param>
        /// <returns></returns>
        async Task IDNCService.CreateOrUpdateNetworkContainerAsync(
            string networkID,
            string networkContainerID,
            NetworkContainerRequest ncRequest,
            string apiVersion)
        {
            try
            {
                if (ncRequest is NetworkContainerRequestV1)
                {
                    throw new DncNetworkContainerException(
                        networkID,
                        networkContainerID,
                        new ErrorInfo(ErrorCode.RequestNotsupported,
                        "NetworkcontainerRequestV1 currently not supported"));
                }

                NetworkContainerRequestV2 ncRequestV2 = (NetworkContainerRequestV2)ncRequest;

                NCConfiguration ncConfig;
                using (ITransaction tx = RepositoryHelper.GetTransaction())
                {
                    string ncKey = NCConfiguration.GetNCKey(networkID, networkContainerID);
                    RequestContext requestContext = new RequestContext(
                        ncKey,
                        networkID,
                        ncRequest.AssociationInfo.NodeID,
                        RequestType.NetworkContainerRequest);

                    ncConfig = await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        networkRepositoryManager.NCConfigReliableCollection,
                        ncKey);


                    if (ncConfig != null)
                    {
                        goto Cleanup;
                    }

                    string nodeKey = NetworkManager.DataModel.NodeConfiguration.GetNodeKey(ncRequest.AssociationInfo.NodeID);
                    List<NetworkContainerResponse> networkContainerResponses = new List<NetworkContainerResponse>();

                    NetworkManager.DataModel.NodeConfiguration nodeConfiguration =
                        await RepositoryHelper.GetFromReliableDictionary(
                            tx,
                            networkRepositoryManager.NodeConfigReliableCollection,
                            nodeKey);
                    if (nodeConfiguration == null)
                    {
                        throw new DncNetworkContainerException(
                            networkID,
                            ncRequest.AssociationInfo.NodeID,
                            new ErrorInfo(ErrorCode.NodeNotFound, "Node not found"));
                    }

                    string networkKey = NetworkConfiguration.GetNetworkKey(networkID);
                    NetworkConfiguration networkConfiguration =
                        await RepositoryHelper.GetFromReliableDictionary(
                            tx,
                            networkRepositoryManager.NetworkConfigReliableCollection,
                            networkKey);
                    if (networkConfiguration == null)
                    {
                        throw new DncNetworkContainerException(
                            networkID,
                            networkContainerID,
                            new ErrorInfo(ErrorCode.NetworkNotFound, "Network associated with container not found"));
                    }

                    string subnetKey = SubnetConfiguration.GetSubnetConfigurationKey(networkID,
                        ncRequestV2.AllocationRequest.SubnetName);
                    SubnetConfiguration subnetConfiguration =
                        await RepositoryHelper.GetFromReliableDictionary(
                            tx,
                            networkRepositoryManager.SubnetConfigReliableCollection,
                            subnetKey);

                    if (subnetConfiguration == null)
                    {
                        throw new DncNetworkContainerException(
                            networkID,
                            networkContainerID,
                            new ErrorInfo(ErrorCode.SubnetNotFound,
                            $"Subnet {ncRequestV2.AllocationRequest.SubnetName} associated with container request not found"));
                    }

                    if (!subnetConfiguration.Joined)
                    {
                        throw new DncNetworkContainerException(
                            networkID,
                            networkContainerID,
                            new ErrorInfo(ErrorCode.SubnetIsNotJoined,
                            $"Subnet {ncRequestV2.AllocationRequest.SubnetName} associated with container is not joined"));
                    }

                    List<IPSubnet> allocatedSubnets = new List<IPSubnet>();
                    foreach(KeyValuePair<string, string> reservationKvp in subnetConfiguration.Reservations)
                    {
                        allocatedSubnets.Add(IPSubnet.Parse(reservationKvp.Value));
                        ServiceEventSource.Current.Message($"Reservation Id:{reservationKvp.Key} " +
                            $"Value:{reservationKvp.Value}");
                    }

                    ReservationConfiguration reservationConfiguration = new ReservationConfiguration(
                        networkID,
                        Guid.NewGuid().ToString(),
                        ncRequestV2.AllocationRequest);

                    reservationConfiguration.AllocatedSubnet = 
                        AllocationHelper.AllocateSubnetFromBaseSubnet(
                            IPSubnet.Parse(subnetConfiguration.IPSubnet),
                            allocatedSubnets,
                            32).ToString();
                    reservationConfiguration.AssociatedNCID = networkContainerID;
                    ncConfig = new NCConfiguration(
                        networkID,
                        networkContainerID,
                        ncRequest,
                        apiVersion);
                    ncConfig.ReservationID = reservationConfiguration.ReservationID;
                    ncConfig.ImplicitReservation = true;
                    ncConfig.State = NetworkContainerState.Publish;
                    ncConfig.Status = Status.Pending;
                    
                    await RepositoryHelper.CreateOrUpdateReliableDictionary(
                        tx,
                        networkRepositoryManager.NCConfigReliableCollection,
                        ncKey,
                        ncConfig);
                    ncConfig.CNSVersion = 1;

                    ServiceEventSource.Current.Message(
                        $"NConfig:{ncKey} Config:{ncConfig}");

                    string reservationKey = ReservationConfiguration.GetReservationKey(
                            networkID,
                            reservationConfiguration.ReservationID);
                    await RepositoryHelper.CreateOrUpdateReliableDictionary(
                        tx,
                        networkRepositoryManager.ReservationReliableCollection,
                        reservationKey,
                        reservationConfiguration);
                    subnetConfiguration.AddReservation(
                        reservationConfiguration.ReservationID, 
                        reservationConfiguration.AllocatedSubnet);
                    ServiceEventSource.Current.Message(
                        $"Reservation:{reservationKey} Config:{reservationConfiguration}");

                    await RepositoryHelper.CreateOrUpdateReliableDictionary(
                        tx,
                        networkRepositoryManager.SubnetConfigReliableCollection,
                        subnetKey,
                        subnetConfiguration);
                    ServiceEventSource.Current.Message(
                        $"Subnet:{subnetKey} Config:{subnetConfiguration}");

                    nodeConfiguration.AddNetworkContainer(networkID, networkContainerID);

                    await RepositoryHelper.CreateOrUpdateReliableDictionary(
                        tx,
                        networkRepositoryManager.NodeConfigReliableCollection,
                        nodeKey,
                        nodeConfiguration);
                    ServiceEventSource.Current.Message(
                        $"Node:{nodeKey} Config:{nodeConfiguration}");

                    Cleanup:
                    await RepositoryHelper.RemoveFromReliableDictionary(
                                            tx,
                                            networkRepositoryManager.FailedRequestsReliableCollection,
                                            ncKey);

                    await RepositoryHelper.CreateOrUpdateReliableDictionary(
                                            tx,
                                            networkRepositoryManager.PendingRequestsReliableCollection,
                                            ncKey,
                                            requestContext);

                    ServiceEventSource.Current.Message(
                        $"Request:{ncKey} Config:{requestContext}");

                    await RepositoryHelper.CommitTransactionAsync(tx);
                }

                return;
            }
            catch (Exception exception)
            {
                ServiceEventSource.Current.Message($"CreateOrUpdateNetworkContainerAsync failed with exception:{exception}");
                throw;
            }

        }

        /// <summary>
        /// Delete network.
        /// </summary>
        /// <param name="networkID">Supply the ID of the network.</param>
        /// <returns></returns>
        async Task IDNCService.DeleteNetworkAsync(string networkID)
        {
            try
            {
                using (ITransaction tx = RepositoryHelper.GetTransaction())
                {
                    ServiceEventSource.Current.Message($"Deleting network:{networkID}");
                    string networkKey = NetworkManager.DataModel.NetworkConfiguration.GetNetworkKey(networkID);
                    NetworkConfiguration networkConfiguration =
                        await RepositoryHelper.GetFromReliableDictionary(
                            tx,
                            networkRepositoryManager.NetworkConfigReliableCollection,
                            networkKey);

                    if (networkConfiguration != null)
                    {
                        ServiceEventSource.Current.Message($"Retrieved config:{networkConfiguration}");
                        foreach (string subnet in networkConfiguration.Subnets.Keys)
                        {
                            string subnetKey = SubnetConfiguration.GetSubnetConfigurationKey(networkID, subnet);
                            ServiceEventSource.Current.Message($"Removing subnet:{subnet}");
                            SubnetConfiguration subnetConfiguration =
                                                await RepositoryHelper.GetFromReliableDictionary(
                                                    tx,
                                                    networkRepositoryManager.SubnetConfigReliableCollection,
                                                    subnetKey);

                            if (subnetConfiguration != null && subnetConfiguration.Reservations.Count > 0)
                            {
                                throw new DncNetworkManagementException(
                                    networkID,
                                    new ErrorInfo(ErrorCode.SubnetInUse,
                                    $"Subnet {subnetConfiguration.SubnetName} has reservations in it"));
                            }
                            ServiceEventSource.Current.Message($"Removing subnet configuration:{subnetConfiguration}");
                            await RepositoryHelper.RemoveFromReliableDictionary(
                                tx,
                                networkRepositoryManager.SubnetConfigReliableCollection,
                                subnetKey);
                        }

                        await RepositoryHelper.RemoveFromReliableDictionary(
                            tx,
                            networkRepositoryManager.NetworkConfigReliableCollection,
                            networkKey);
                        await RepositoryHelper.CommitTransactionAsync(tx);
                    }
                }
            }
            catch(Exception exception)
            {
                ServiceEventSource.Current.Message($"Delete network:{networkID} failed with exception {exception}");
                throw;
            }
        }

        /// <summary>
        /// Delete node.
        /// </summary>
        /// <param name="nodeID">Supply the ID of the node.</param>
        /// <returns></returns>
        async Task IDNCService.DeleteNodeAsync(string nodeID)
        {
            try
            {
                using (ITransaction tx = RepositoryHelper.GetTransaction())
                {
                    string nodeKey = NetworkManager.DataModel.NodeConfiguration.GetNodeKey(nodeID);
                    NetworkManager.DataModel.NodeConfiguration nodeConfiguration =
                        await RepositoryHelper.GetFromReliableDictionary(
                            tx,
                            networkRepositoryManager.NodeConfigReliableCollection,
                            nodeKey);

                    if (nodeConfiguration != null)
                    {
                        if (nodeConfiguration.DeployedNCMap.Count != 0)
                        {
                            throw new DncNodeManagementException(
                                nodeID,
                                new ErrorInfo(ErrorCode.NodeInUse, "Node has NC deployed in it"));
                        }

                        await RepositoryHelper.RemoveFromReliableDictionary(
                            tx,
                            networkRepositoryManager.NodeConfigReliableCollection,
                            nodeKey);
                        await RepositoryHelper.CommitTransactionAsync(tx);
                    }
                }
            }
            catch (Exception exception)
            {
                ServiceEventSource.Current.Message("Failed to delete node {0} due to excpetion {1}",
                    nodeID,
                    exception);
                throw;
            }
        }

        async Task IDNCService.DeleteNetworkContainerAsync(string networkID, string networkContainerID)
        {
            try
            { 
                using (ITransaction tx = RepositoryHelper.GetTransaction())
                {
                    string ncKey = NCConfiguration.GetNCKey(networkID, networkContainerID);
                    NetworkManager.DataModel.NCConfiguration ncConfiguration =
                        await RepositoryHelper.GetFromReliableDictionary(
                            tx,
                            networkRepositoryManager.NCConfigReliableCollection,
                            ncKey);

                    if (ncConfiguration != null)
                    {
                        RequestContext requestContext = new RequestContext(
                            ncKey,
                            networkID,
                            ncConfiguration.Request.AssociationInfo.NodeID,
                            RequestType.NetworkContainerRequest);

                        await RepositoryHelper.RemoveFromReliableDictionary(
                                                tx,
                                                networkRepositoryManager.FailedRequestsReliableCollection,
                                                ncKey);

                        await RepositoryHelper.CreateOrUpdateReliableDictionary(
                                                tx,
                                                networkRepositoryManager.PendingRequestsReliableCollection,
                                                ncKey,
                                                requestContext);
                        ncConfiguration.MarkedForDelete = true;
                        ncConfiguration.State = NetworkContainerState.DisAssociate;
                        await RepositoryHelper.CreateOrUpdateReliableDictionary(
                                                tx,
                                                networkRepositoryManager.NCConfigReliableCollection,
                                                ncKey,
                                                ncConfiguration);
                        await RepositoryHelper.CommitTransactionAsync(tx);

                    }
                }
            }
            catch (Exception exception)
            {
                ServiceEventSource.Current.Message("Failed to delete network container {0} in network {1} due to excpetion {2}",
                    networkContainerID,
                    networkID,
                    exception);
                throw;
            }
        }

        /// <summary>
        /// Get network container.
        /// </summary>
        /// <param name="networkID">Supply the network ID.</param>
        /// <param name="networkContainerID">Supply the network container ID.</param>
        /// <returns>Network container response.</returns>
        async Task<NetworkContainerResponse> IDNCService.GetNetworkContainerResponseAsync(
            string networkID,
            string networkContainerID)
        {
            try
            {
                using (ITransaction tx = RepositoryHelper.GetTransaction())
                {
                    string ncKey = NCConfiguration.GetNCKey(networkID, networkContainerID);
                    NCConfiguration ncConfiguration =
                        await RepositoryHelper.GetFromReliableDictionary(
                            tx,
                            networkRepositoryManager.NCConfigReliableCollection,
                            ncKey);

                    if (ncConfiguration == null)
                    {
                        throw new DncNetworkContainerException(
                            networkID,
                            networkContainerID,
                            new ErrorInfo(ErrorCode.NetworkContainerNotFound, "Container not found"));
                    }

                    string reservationKey = ReservationConfiguration.GetReservationKey(
                        networkID,
                        ncConfiguration.ReservationID);
                    ReservationConfiguration reservationConfiguration = await RepositoryHelper.GetFromReliableDictionary(
                            tx,
                            networkRepositoryManager.ReservationReliableCollection,
                            reservationKey);
                    if (reservationConfiguration == null)
                    {
                        throw new DncNetworkContainerException(
                            networkID,
                            networkContainerID,
                            new ErrorInfo(
                                ErrorCode.ReservationNotFound,
                                $"Reservation {ncConfiguration.ReservationID} for Container not found"));
                    }

                    string networkKey = NetworkConfiguration.GetNetworkKey(networkID);
                    NetworkConfiguration networkConfiguration =
                        await RepositoryHelper.GetFromReliableDictionary(
                            tx,
                            networkRepositoryManager.NetworkConfigReliableCollection,
                            networkKey);

                    if (networkConfiguration == null)
                    {
                        throw new DncNetworkContainerException(
                            networkID,
                            networkContainerID,
                            new ErrorInfo(ErrorCode.NetworkNotFound, "Network associated with container not found"));
                    }

                    string subnetKey = SubnetConfiguration.GetSubnetConfigurationKey(networkID,
                        reservationConfiguration.AllocationRequest.SubnetName);
                    SubnetConfiguration subnetConfiguration =
                        await RepositoryHelper.GetFromReliableDictionary(
                            tx,
                            networkRepositoryManager.SubnetConfigReliableCollection,
                            subnetKey);

                    if (subnetConfiguration == null)
                    {
                        throw new DncNetworkContainerException(
                            networkID,
                            networkContainerID,
                            new ErrorInfo(ErrorCode.SubnetNotFound,
                            $"Subnet {reservationConfiguration.AllocationRequest.SubnetName} associated with container not found"));
                    }

                    string nodeKey = NetworkManager.DataModel.NodeConfiguration.GetNodeKey(
                        ncConfiguration.Request.AssociationInfo.NodeID);
                    NetworkManager.DataModel.NodeConfiguration nodeConfiguration =
                        await RepositoryHelper.GetFromReliableDictionary(
                            tx,
                            networkRepositoryManager.NodeConfigReliableCollection,
                            nodeKey);
                    return RepoToApiContractTranslators.GetNetworkContainerResponse(
                        networkConfiguration,
                        ncConfiguration,
                        subnetConfiguration,
                        reservationConfiguration,
                        nodeConfiguration);
                }
            }
            catch (Exception exception)
            {
                ServiceEventSource.Current.Message("Failed to get network container {0} in network {1} due to excpetion {2}",
                    networkContainerID,
                    networkID,
                    exception);
                throw;
            }
        }

        /// <summary>
        /// Get node information.
        /// </summary>
        /// <param name="nodeID">Supply the node ID.</param>
        /// <returns>Node response.</returns>
        async Task<NodeResponse> IDNCService.GetNodeInfoResponseAsync(string nodeID)
        {
            try
            {
                using (ITransaction tx = RepositoryHelper.GetTransaction())
                {
                    string nodeKey = NetworkManager.DataModel.NodeConfiguration.GetNodeKey(nodeID);
                    List<NetworkContainerResponse> networkContainerResponses = new List<NetworkContainerResponse>();

                    NetworkManager.DataModel.NodeConfiguration nodeConfiguration =
                        await RepositoryHelper.GetFromReliableDictionary(
                            tx,
                            networkRepositoryManager.NodeConfigReliableCollection,
                            nodeKey);
                    if (nodeConfiguration == null)
                    {
                        throw new DncNodeManagementException(
                            nodeID,
                            new ErrorInfo(ErrorCode.NodeNotFound, "Node not found"));
                    }

                    foreach (KeyValuePair<string, IReadOnlyDictionary<string, string>> networkNCKvp in nodeConfiguration.DeployedNCMap)
                    {
                        string networkID = networkNCKvp.Key;
                        foreach (string networkContainerID in networkNCKvp.Value.Keys)
                        {
                            string ncKey = NCConfiguration.GetNCKey(networkID, networkContainerID);

                            NCConfiguration ncConfiguration =
                                await RepositoryHelper.GetFromReliableDictionary(
                                    tx,
                                    networkRepositoryManager.NCConfigReliableCollection,
                                    ncKey);

                            if (ncConfiguration == null)
                            {
                                continue;
                            }

                            string reservationKey = ReservationConfiguration.GetReservationKey(
                                networkID,
                                ncConfiguration.ReservationID);
                            ReservationConfiguration reservationConfiguration = await RepositoryHelper.GetFromReliableDictionary(
                                    tx,
                                    networkRepositoryManager.ReservationReliableCollection,
                                    reservationKey);
                            if (reservationConfiguration == null)
                            {
                                continue;
                            }

                            string networkKey = NetworkConfiguration.GetNetworkKey(networkID);
                            NetworkConfiguration networkConfiguration =
                                await RepositoryHelper.GetFromReliableDictionary(
                                    tx,
                                    networkRepositoryManager.NetworkConfigReliableCollection,
                                    networkKey);

                            if (networkConfiguration == null)
                            {
                                continue;
                            }

                            string subnetKey = SubnetConfiguration.GetSubnetConfigurationKey(networkID,
                                reservationConfiguration.AllocationRequest.SubnetName);
                            SubnetConfiguration subnetConfiguration =
                                await RepositoryHelper.GetFromReliableDictionary(
                                    tx,
                                    networkRepositoryManager.SubnetConfigReliableCollection,
                                    subnetKey);

                            if (subnetConfiguration == null)
                            {
                                continue;
                            }

                            
                            NetworkContainerResponse ncResponse =
                                RepoToApiContractTranslators.GetNetworkContainerResponse(
                                networkConfiguration,
                                ncConfiguration,
                                subnetConfiguration,
                                reservationConfiguration,
                                nodeConfiguration);

                            networkContainerResponses.Add(ncResponse);
                        }
                    }

                    return new NodeResponse(
                        nodeConfiguration.NodeInfo,
                        networkContainerResponses.ToArray(),
                        new List<ReservationResponse>().ToArray());
                }
            }
            catch (Exception exception)
            {
                ServiceEventSource.Current.Message($"GetNodeInfoResponseAsync for node:{nodeID} failed with exception:{exception}");
                throw;
            }
        }
    }
}
