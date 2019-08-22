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
using System.Net;
using NodeConfiguration = NetworkManager.DataModel.NodeConfiguration;

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

        private TagsCacheManager tagsCacheManager;

        private CancellationToken cancellationToken;

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
            tagsCacheManager = new TagsCacheManager();
            cnsClient = new CnsClient(DncConfigurationInstance, tagsCacheManager);
            requestWatcher = new RequestWatcher(hostAgentClient, cnsClient, tagsCacheManager, DncConfigurationInstance);
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
                                .UseServiceFabricIntegration(listener, ServiceFabricIntegrationOptions.None)
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
                this.cancellationToken = cancellationToken;
                RepositoryHelper.Init(this.StateManager, cancellationToken, new TimeSpan(0, 0, 30));
                networkRepositoryManager = new NetworkRepositoryManager(DncConfigurationInstance);

                tagsCacheManager.Init(hostAgentClient, networkRepositoryManager);

                await requestWatcher.Init(networkRepositoryManager, cancellationToken);
                requestWatcher.Start();

                RequestResetMonitor requestResetMonitor = new RequestResetMonitor(
                    networkRepositoryManager,
                    cancellationToken);
                requestResetMonitor.StartAsync();

                NetworkRefreshMonitor networkRefreshMonitor = new NetworkRefreshMonitor(
                    StateManager,
                    hostAgentClient,
                    networkRepositoryManager,
                    tagsCacheManager,
                    DncConfigurationInstance,
                    cancellationToken);
                networkRefreshMonitor.StartAsync();

                NetworkContainerMonitor networkContainerMonitor = new NetworkContainerMonitor(
                    networkRepositoryManager,
                    DncConfigurationInstance,
                    cancellationToken);
                networkContainerMonitor.StartAsync();

                TagRefreshMonitor tagRefreshMonitor = new TagRefreshMonitor(
                    tagsCacheManager,
                    DncConfigurationInstance,
                    cancellationToken);
                tagRefreshMonitor.StartAsync();

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
                SubnetConfiguration subnetConfiguration;
                using (ITransaction tx = RepositoryHelper.GetTransaction())
                {
                    NetworkConfiguration networkConfig =
                        await RepositoryHelper.GetFromReliableDictionary(
                            tx,
                            networkRepositoryManager.NetworkConfigReliableCollection,
                            NetworkConfiguration.GetNetworkKey(networkID));
                    if ((networkConfig == null) ||
                        ((networkConfig.State != NetworkState.Join) &&
                         (networkConfig.Status != Status.Completed)))
                    {
                        throw new DncNetworkManagementException(
                            networkID,
                            new ErrorInfo(ErrorCode.NetworkNotFound,
                                "Network not found",
                                (int)HttpStatusCode.NotFound,
                                ErrorType.NonRetriable));
                    }

                    string subnetKey =
                        SubnetConfiguration.GetSubnetConfigurationKey(networkID, subnetName);
                    subnetConfiguration = await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        networkRepositoryManager.SubnetConfigReliableCollection,
                        subnetKey,
                        LockMode.Update);

                    if (subnetConfiguration == null)
                    {
                        throw new DncNetworkManagementException(
                            networkID,
                            new ErrorInfo(
                                ErrorCode.SubnetNotFound,
                                $"Subnet {subnetName} not found",
                                (int)HttpStatusCode.NotFound,
                                ErrorType.NonRetriable));
                    }

                    subnetConfiguration.AccessToken = subnetRequest.AuthenticationToken;
                    subnetConfiguration.IsJoined = true;
                    await RepositoryHelper.CreateOrUpdateReliableDictionary(
                        tx,
                        networkRepositoryManager.SubnetConfigReliableCollection,
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
        async Task<SubnetResponse> IDNCService.GetSubnetConfigurationResponseAsync(
            string networkID,
            string subnetName)
        {
            SubnetConfiguration subnetConfiguration;
            NetworkConfiguration networkConfig;
            try
            {
                List<ReservationConfiguration> reservationConfigurations = new List<ReservationConfiguration>();
                using (ITransaction tx = RepositoryHelper.GetTransaction())
                {
                    networkConfig = await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        networkRepositoryManager.NetworkConfigReliableCollection,
                        NetworkConfiguration.GetNetworkKey(networkID));
                    if (networkConfig == null)
                    {
                        throw new DncNetworkManagementException(
                            networkID,
                            new ErrorInfo(ErrorCode.NetworkNotFound,
                                "Network configuration not found",
                                (int)HttpStatusCode.NotFound,
                                ErrorType.NonRetriable));
                    }

                    subnetConfiguration = await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        networkRepositoryManager.SubnetConfigReliableCollection,
                        SubnetConfiguration.GetSubnetConfigurationKey(networkID, subnetName));
                    if (subnetConfiguration == null)
                    {
                        throw new DncNetworkManagementException(
                            networkID,
                            new ErrorInfo(
                                ErrorCode.SubnetNotFound,
                                $"Subnet {subnetName} not found",
                                (int)HttpStatusCode.NotFound,
                                ErrorType.NonRetriable));
                    }

                    foreach (string reservationID in subnetConfiguration.Reservations.Keys)
                    {
                        ReservationConfiguration reservationConfiguration = await
                            RepositoryHelper.GetFromReliableDictionary(
                                tx,
                                networkRepositoryManager.ReservationReliableCollection,
                                ReservationConfiguration.GetReservationKey(networkID, reservationID));
                        if(reservationConfiguration != null)
                        {
                            reservationConfigurations.Add(reservationConfiguration);
                        }
                    }
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
        /// Get network status.
        /// </summary>
        /// <param name="networkID">Supply the ID of the network.</param>
        /// <returns>Network status.</returns>
        async Task<RequestStatus> IDNCService.GetNetworkStatusResponseAsync(
            string networkID)
        {
            using (ITransaction tx = RepositoryHelper.GetTransaction())
            {
                string networkKey = NetworkConfiguration.GetNetworkKey(networkID);
                NetworkConfiguration networkConfig;
                try
                {
                    networkConfig = await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        networkRepositoryManager.NetworkConfigReliableCollection,
                        networkKey);
                    if (networkConfig == null)
                    {
                        throw new DncNetworkManagementException(
                            networkID,
                            new ErrorInfo(ErrorCode.NetworkNotFound,
                                "Network configuration not found",
                                (int)HttpStatusCode.NotFound,
                                ErrorType.NonRetriable));
                    }
                }
                catch (Exception exception)
                {
                    ServiceEventSource.Current.Message(
                        $"Failed to get network {networkID} with exception:{exception}");
                    throw;
                }

                RequestStatus requestStatus =
                    new RequestStatus(
                        networkConfig.State.ToString(),
                        networkConfig.Status.ToString(),
                        networkConfig.FailureDetail);

                return (requestStatus);
            }
        }

        /// <summary>
        /// Get subnet status.
        /// </summary>
        /// <param name="networkID">Supply the ID of the network.</param>
        /// <param name="subnetName">Supply the subnet name.</param>
        /// <returns>subnet status.</returns>
        async Task<RequestStatus> IDNCService.GetSubnetStatusResponseAsync(
            string networkID,
            string subnetName)
        {
            using (ITransaction tx = RepositoryHelper.GetTransaction())
            {
                string networkKey = NetworkConfiguration.GetNetworkKey(networkID);
                SubnetConfiguration subnetConfig;
                RequestStatus requestStatus;

                try
                {
                    NetworkConfiguration networkConfig =
                        await RepositoryHelper.GetFromReliableDictionary(
                            tx,
                            networkRepositoryManager.NetworkConfigReliableCollection,
                            networkKey);
                    if (networkConfig == null)
                    {
                        throw new DncNetworkManagementException(
                            networkID,
                            new ErrorInfo(ErrorCode.NetworkNotFound,
                                "Network configuration not found",
                                (int)HttpStatusCode.NotFound,
                                ErrorType.NonRetriable));
                    }

                    string subnetKey =
                        SubnetConfiguration.GetSubnetConfigurationKey(networkID, subnetName);
                    subnetConfig = await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        networkRepositoryManager.SubnetConfigReliableCollection,
                        subnetKey);
                    if (subnetConfig == null)
                    {
                        throw new DncNetworkManagementException(
                            networkID,
                            new ErrorInfo(
                                ErrorCode.SubnetNotFound,
                                $"Subnet {subnetName} not found",
                                (int)HttpStatusCode.NotFound,
                                ErrorType.NonRetriable));
                    }
                }
                catch (Exception exception)
                {
                    ServiceEventSource.Current.Message(
                        $"Failed to get subnet {subnetName} in network {networkID} with exception:{exception}");
                    throw;
                }

                if (subnetConfig.IsJoined)
                {
                    requestStatus = new RequestStatus(
                        Enum.GetName(typeof(NetworkState), NetworkState.Join),
                        Enum.GetName(typeof(Status), Status.Completed),
                        null);
                }
                else
                {
                    requestStatus = new RequestStatus(
                        Enum.GetName(typeof(NetworkState), NetworkState.Unknown),
                        Enum.GetName(typeof(Status), Status.NotStarted),
                        null);
                }

                return (requestStatus);
            }
        }

        /// <summary>
        /// Retrieve the network container status.
        /// </summary>
        /// <param name="networkID">Supply the ID of the network.</param>
        /// <param name="networkContainerID">Supply the network container ID.</param>
        /// <returns>Network container status.</returns>
        async Task<RequestStatus> IDNCService.GetNetworkContainerStatusResponseAsync(
            string networkID,
            string networkContainerID)
        {
            ServiceEventSource.Current.Message($"GeNetworkContainerStatus with " +
                    $"network id :{networkID} ncid {networkContainerID}");
            using (ITransaction tx = RepositoryHelper.GetTransaction())
            {
                NCConfiguration ncConfig;
                try
                {
                    string ncKey = NCConfiguration.GetNCKey(networkID, networkContainerID);
                    ncConfig = await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        networkRepositoryManager.NCConfigReliableCollection,
                        ncKey);
                    if (ncConfig == null)
                    {
                        throw new DncNetworkContainerException(
                            networkID,
                            networkContainerID,
                            new ErrorInfo(ErrorCode.NetworkContainerNotFound, "Network Container not found",
                                (int)HttpStatusCode.NotFound, ErrorType.NonRetriable));
                    }
                }
                catch (Exception exception)
                {
                    ServiceEventSource.Current.Message(
                        $"Failed to get network container {networkContainerID} with exception:{exception}");
                    throw;
                }

                RequestStatus requestStatus =
                    new RequestStatus(
                        ncConfig.State.ToString(),
                        ncConfig.Status.ToString(),
                        ncConfig.FailureDetail);

                return (requestStatus);
            }
        }

        /// <summary>
        /// Retrieve the reservation status.
        /// </summary>
        /// <param name="networkID">Supply the ID of the network.</param>
        /// <param name="reservationID">Supply the reservation ID.</param>
        /// <returns>Network container status.</returns>
        async Task<RequestStatus> IDNCService.GetReservationStatusResponseAsync(
            string networkID,
            string reservationID)
        {
            using (ITransaction tx = RepositoryHelper.GetTransaction())
            {
                ReservationConfiguration reservationConfig;
                try
                {
                    string reservationKey = ReservationConfiguration.GetReservationKey(networkID, reservationID);
                    reservationConfig = await networkRepositoryManager.GetReservationConfiguration(
                        reservationKey,
                        tx);
                    if (reservationConfig == null)
                    {
                        throw new DncManagementException(
                            new ErrorInfo(
                                ErrorCode.ReservationNotFound,
                                "Reservation not found",
                                (int)HttpStatusCode.NotFound,
                                ErrorType.NonRetriable));
                    }
                }
                catch (Exception exception)
                {
                    DncManagementException dncManagementException =
                        (DncManagementException)exception;
                    return (new RequestStatus("", "", dncManagementException.ErrorInfo));
                }

                RequestStatus requestStatus =
                    new RequestStatus(
                        reservationConfig.ReservationState.ToString(),
                        reservationConfig.Status.ToString(),
                        reservationConfig.FailureDetail);

                return (requestStatus);
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
                NetworkConfiguration networkConfig;
                using (ITransaction tx = RepositoryHelper.GetTransaction())
                {
                    string networkKey = NetworkConfiguration.GetNetworkKey(networkID);
                    networkConfig = await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        networkRepositoryManager.NetworkConfigReliableCollection,
                        networkKey);

                    if (networkConfig == null)
                    {
                        networkConfig = new NetworkConfiguration(
                            "",
                            networkID,
                            networkInfoRequest,
                            apiVersion);
                    }
                    else
                    {
                        networkConfig.UpdateNetworkInfo(networkInfoRequest);
                    }

                    await RepositoryHelper.CreateOrUpdateReliableDictionary(
                        tx,
                        networkRepositoryManager.NetworkConfigReliableCollection,
                        networkKey,
                        networkConfig);

                    RequestContext requestContext = new RequestContext(
                        networkKey,
                        networkID,
                        "",
                        RequestType.VirtualNetworkRequest);
                    await RepositoryHelper.CreateOrUpdateReliableDictionary(
                        tx,
                        networkRepositoryManager.PendingRequestsReliableCollection,
                        networkKey,
                        requestContext);
                    
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
                NodeConfiguration nodeConfig;
                using (ITransaction tx = RepositoryHelper.GetTransaction())
                {
                    string nodeKey = NodeConfiguration.GetNodeKey(nodeID);
                    nodeConfig = await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        networkRepositoryManager.NodeConfigReliableCollection,
                        nodeKey);

                    await cnsClient.SetOrchestratorType(nodeInfoRequest);

                    if (nodeConfig == null)
                    {
                        nodeConfig = new NodeConfiguration(
                            nodeID,
                            apiVersion,
                            nodeInfoRequest);
                    }
                    else
                    {
                        nodeConfig.UpdateNodeInfo(nodeInfoRequest);
                    }

                    await RepositoryHelper.CreateOrUpdateReliableDictionary(
                        tx,
                        networkRepositoryManager.NodeConfigReliableCollection,
                        NodeConfiguration.GetNodeKey(nodeID),
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
                ServiceEventSource.Current.Message($"Creating network container with " +
                    $"network id :{networkID} ncid {networkContainerID}");
                if (ncRequest is NetworkContainerRequestV1)
                {
                    throw new DncNetworkContainerException(
                        networkID,
                        networkContainerID,
                        new ErrorInfo(ErrorCode.RequestNotsupported,
                        "NetworkcontainerRequestV1 currently not supported",
                        ErrorType.NonRetriable));
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
                    requestContext.SetSubnetName(ncRequestV2.AllocationRequest.SubnetName);

                    ncConfig = await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        networkRepositoryManager.NCConfigReliableCollection,
                        ncKey);

                    if (ncConfig != null)
                    {
                        goto Cleanup;
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
                            new ErrorInfo(ErrorCode.NetworkNotFound,
                                "Network associated with container not found",
                                ErrorType.NonRetriable));
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
                            $"Subnet {ncRequestV2.AllocationRequest.SubnetName} " +
                            $"associated with container request not found",
                            ErrorType.NonRetriable));
                    }

                    if (!subnetConfiguration.IsJoined)
                    {
                        throw new DncNetworkContainerException(
                            networkID,
                            networkContainerID,
                            new ErrorInfo(ErrorCode.SubnetIsNotJoined,
                            $"Subnet {ncRequestV2.AllocationRequest.SubnetName} " +
                            $"associated with container is not joined",
                            ErrorType.NonRetriable));
                    }

                    ReservationConfiguration reservationConfiguration = new ReservationConfiguration(
                        networkID,
                        Guid.NewGuid().ToString(),
                        ncRequestV2.AllocationRequest);

                    reservationConfiguration.AssociatedNCID = networkContainerID;
                    ncConfig = new NCConfiguration(
                        networkID,
                        networkContainerID,
                        ncRequest,
                        apiVersion);
                    ncConfig.ReservationID = reservationConfiguration.ReservationID;
                    ncConfig.ImplicitReservation = true;
                    ncConfig.State = NetworkContainerState.Reserve;
                    ncConfig.Status = Status.Pending;

                    await RepositoryHelper.CreateOrUpdateReliableDictionary(
                        tx,
                        networkRepositoryManager.NCConfigReliableCollection,
                        ncKey,
                        ncConfig);
                    ncConfig.CNSVersion = 0;

                    string reservationKey = ReservationConfiguration.GetReservationKey(
                        networkID,
                        reservationConfiguration.ReservationID);
                    await RepositoryHelper.CreateOrUpdateReliableDictionary(
                        tx,
                        networkRepositoryManager.ReservationReliableCollection,
                        reservationKey,
                        reservationConfiguration);

                    ServiceEventSource.Current.Message(
                        $"Reservation Key:{reservationKey}\n Reservation Config:{reservationConfiguration}");
                    ServiceEventSource.Current.Message(
                        $"Subnet Key:{subnetKey}\n Subnet Config:{subnetConfiguration}");

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

                    await RepositoryHelper.CommitTransactionAsync(tx);

                    ServiceEventSource.Current.Message($"NC Key:{ncKey}\nNC Config:{ncConfig}");
                    ServiceEventSource.Current.Message($"Request Key:{ncKey} Request Config:{requestContext}");
                    ServiceEventSource.Current.Message($"Network container creation request queued");
                }

                return;
            }
            catch (Exception exception)
            {
                ServiceEventSource.Current.Message($"CreateOrUpdateNetworkContainerAsync failed with exception: {exception}");
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
                    string networkKey = NetworkConfiguration.GetNetworkKey(networkID);
                    NetworkConfiguration networkConfiguration =
                        await RepositoryHelper.GetFromReliableDictionary(
                            tx,
                            networkRepositoryManager.NetworkConfigReliableCollection,
                            networkKey,
                            LockMode.Update);

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
                                    subnetKey,
                                    LockMode.Update);

                            if (subnetConfiguration != null && subnetConfiguration.Reservations.Count > 0)
                            {
                                throw new DncNetworkManagementException(
                                    networkID,
                                    new ErrorInfo(ErrorCode.NetworkInUse,
                                    $"Subnet {subnetConfiguration.SubnetName} has reservations in it",
                                    (int)HttpStatusCode.BadRequest,
                                    ErrorType.NonRetriable));
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
                    string nodeKey = NodeConfiguration.GetNodeKey(nodeID);
                    NodeConfiguration nodeConfiguration =
                        await RepositoryHelper.GetFromReliableDictionary(
                            tx,
                            networkRepositoryManager.NodeConfigReliableCollection,
                            nodeKey,
                            LockMode.Update);

                    if (nodeConfiguration != null)
                    {
                        if (nodeConfiguration.DeployedNCMap.Count != 0)
                        {
                            throw new DncNodeManagementException(
                                nodeID,
                                new ErrorInfo(ErrorCode.NodeInUse, "Node contains deployed NCs",
                                (int)HttpStatusCode.BadRequest,
                                ErrorType.NonRetriable));
                        }
                        else if (nodeConfiguration.constrainedReservations.Count != 0)
                        {
                            throw new DncNodeManagementException(
                                nodeID,
                                new ErrorInfo(ErrorCode.NodeInUse, "Node contains reservations",
                                (int)HttpStatusCode.BadRequest,
                                ErrorType.NonRetriable));
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
            ServiceEventSource.Current.Message($"Deleting network container with " +
                $"network id :{networkID} ncid {networkContainerID}");
            try
            { 
                using (ITransaction tx = RepositoryHelper.GetTransaction())
                {
                    string ncKey = NCConfiguration.GetNCKey(networkID, networkContainerID);
                    NCConfiguration ncConfiguration = await networkRepositoryManager.GetNetworkContainerConfiguration(
                        ncKey,
                        tx,
                        LockMode.Update);
                    if (ncConfiguration != null)
                    {
                        ReservationConfiguration reservationConfig = await networkRepositoryManager.GetReservationConfiguration(
                            ReservationConfiguration.GetReservationKey(networkID, ncConfiguration.ReservationID),
                            tx);
                        if (reservationConfig == null)
                        {
                            throw new DncManagementException(
                                new ErrorInfo(ErrorCode.ReservationNotFound,
                                    "Failed to delete NC due to failure to find reservation",
                                    ErrorType.NonRetriable));
                        }

                        RequestContext requestContext = new RequestContext(
                            ncKey,
                            networkID,
                            ncConfiguration.Request.AssociationInfo.NodeID,
                            RequestType.NetworkContainerRequest);
                        requestContext.SetSubnetName(reservationConfig.AllocationRequest.SubnetName);

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

                        await networkRepositoryManager.UpdateNetworkContainerConfigUntransacted(
                            ncKey,
                            ncConfiguration,
                            tx);

                        await RepositoryHelper.CommitTransactionAsync(tx);
                        ServiceEventSource.Current.Message($"DeleteNetworkContainerAsync: Adding new request with requestID:" +
                            $" {ncKey}, RequestContext:{requestContext}");
                    }
                }
            }
            catch (Exception exception)
            {
                ServiceEventSource.Current.Message($"Failed to delete network container {networkContainerID} in network " +
                    $"{networkID} due to exception {exception}");
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
                    NCConfiguration ncConfiguration = await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        networkRepositoryManager.NCConfigReliableCollection,
                        ncKey);
                    if (ncConfiguration == null)
                    {
                        throw new DncNetworkContainerException(
                            networkID,
                            networkContainerID,
                            new ErrorInfo(ErrorCode.NetworkContainerNotFound, "Container not found",
                                (int)HttpStatusCode.NotFound, ErrorType.NonRetriable));
                    }

                    string nodeKey = NodeConfiguration.GetNodeKey(ncConfiguration.Request.AssociationInfo.NodeID);
                    NodeConfiguration nodeConfiguration = await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        networkRepositoryManager.NodeConfigReliableCollection,
                        nodeKey);

                    string networkKey = NetworkConfiguration.GetNetworkKey(networkID);
                    NetworkConfiguration networkConfiguration = await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        networkRepositoryManager.NetworkConfigReliableCollection,
                        networkKey);
                    if (networkConfiguration == null)
                    {
                        throw new DncNetworkContainerException(
                            networkID,
                            networkContainerID,
                            new ErrorInfo(
                                ErrorCode.NetworkNotFound,
                                "Network associated with container not found",
                                ErrorType.NonRetriable));
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
                                $"Reservation {ncConfiguration.ReservationID} for Container not found",
                                ErrorType.NonRetriable));
                    }

                    string subnetKey = SubnetConfiguration.GetSubnetConfigurationKey(networkID,
                        reservationConfiguration.AllocationRequest.SubnetName);
                    SubnetConfiguration subnetConfiguration = await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        networkRepositoryManager.SubnetConfigReliableCollection,
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
                    string nodeKey = NodeConfiguration.GetNodeKey(nodeID);
                    List<NetworkContainerResponse> networkContainerResponses = new List<NetworkContainerResponse>();

                    NodeConfiguration nodeConfiguration =
                        await RepositoryHelper.GetFromReliableDictionary(
                            tx,
                            networkRepositoryManager.NodeConfigReliableCollection,
                            nodeKey);
                    if (nodeConfiguration == null)
                    {
                        throw new DncNodeManagementException(
                            nodeID,
                            new ErrorInfo(
                                ErrorCode.NodeNotFound,
                                "Node not found",
                                (int)HttpStatusCode.NotFound,
                                ErrorType.NonRetriable));
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

        async Task IDNCService.AddNetworkProfileAsync(
            string networkProfileID)
        {
            try
            {
                using (ITransaction tx = RepositoryHelper.GetTransaction())
                {
                    NetworkProfile networkProfile = new NetworkProfile(networkProfileID);
                    await RepositoryHelper.CreateOrUpdateReliableDictionary(
                        tx,
                        networkRepositoryManager.NetworkProfilesReliableCollection,
                        networkProfileID,
                        networkProfile);
                    await RepositoryHelper.CommitTransactionAsync(tx);
                    ServiceEventSource.Current.Message(
                        $"NetworkProfileID:{networkProfileID} NetworkProfile:{networkProfile}");
                }
            }
            catch (Exception exception)
            {
                ServiceEventSource.Current.Message($"Failed to add network profile with exception:{exception}");
                throw;
            }
        }

        async Task IDNCService.DeleteNetworkProfileAsync(
            string networkProfileID)
        {
            try
            {
                using (ITransaction tx = RepositoryHelper.GetTransaction())
                {
                    NetworkProfile networkProfile =
                        await RepositoryHelper.GetFromReliableDictionary(
                            tx,
                            networkRepositoryManager.NetworkProfilesReliableCollection,
                            networkProfileID);

                    if (networkProfile != null)
                    {
                        await RepositoryHelper.RemoveFromReliableDictionary(
                            tx,
                            networkRepositoryManager.NetworkProfilesReliableCollection,
                            networkProfileID);
                        await RepositoryHelper.CommitTransactionAsync(tx);
                    }

                }
            }
            catch (Exception exception)
            {
                ServiceEventSource.Current.Message($"Failed to delete network profile with exception:{exception}");
                throw;
            }
        }

        /// <summary>
        /// Create reservation.
        /// </summary>
        /// <param name="networkID">Supply the ID of the network.</param>
        /// <param name="reservationID">Supply the ID of reservation.</param>
        /// <param name="ipAllocationRequest">Supply the IP allocation request.</param>
        /// <returns></returns>
        async Task IDNCService.CreateReservationAsync(
            string networkID,
            string reservationID,
            IPAllocationRequest ipAllocationRequest)
        {
            ServiceEventSource.Current.Message($"CreateReservation: NetworkID: {networkID}, " +
                $"ReservationID: {reservationID}, Request: {ipAllocationRequest}");
            try
            {
                using (ITransaction tx = RepositoryHelper.GetTransaction())
                {
                    NetworkConfiguration networkConfig = await networkRepositoryManager.GetNetworkConfiguration(
                        NetworkConfiguration.GetNetworkKey(networkID),
                        tx);
                    if (networkConfig == null)
                    {
                        throw new DncManagementException(
                            new ErrorInfo(
                                ErrorCode.NetworkNotFound,
                                $"Network configuration for network {networkID} not found",
                                ErrorType.NonRetriable));
                    }

                    string reservationKey = ReservationConfiguration.GetReservationKey(networkID, reservationID);
                    ReservationConfiguration reservationConfig =
                        await networkRepositoryManager.GetReservationConfiguration(
                            reservationKey,
                            tx);
                    if (reservationConfig != null)
                    {
                        ServiceEventSource.Current.Message($"Found existing reservation for reservationID: {reservationID}");
                        return;
                    }

                    reservationConfig = new ReservationConfiguration(
                        networkID,
                        Guid.NewGuid().ToString(),
                        ipAllocationRequest);
                    reservationConfig.ReservationState = ReservationState.Allocate;
                    reservationConfig.Status = Status.Pending;
                    await networkRepositoryManager.UpdateReservationConfigurationUntransacted(
                        reservationKey,
                        reservationConfig,
                        tx);

                    RequestContext requestContext = new RequestContext(
                        reservationKey,
                        networkID,
                        ipAllocationRequest.NodeConstraint,
                        RequestType.ReservationRequest);
                    requestContext.SetSubnetName(ipAllocationRequest.SubnetName);

                    await RepositoryHelper.CreateOrUpdateReliableDictionary(
                        tx,
                        networkRepositoryManager.PendingRequestsReliableCollection,
                        reservationKey,
                        requestContext);

                    await RepositoryHelper.CommitTransactionAsync(tx);
                    ServiceEventSource.Current.Message($"Completed persisting request for reservation {reservationID}");
                }
            }
            catch (Exception exception)
            {
                ServiceEventSource.Current.Message($"Failed to process request for create reservation {reservationID} " +
                    $" due to exception {exception}");
                throw;
            }
        }

        /// <summary>
        /// Delete reservation.
        /// </summary>
        /// <param name="networkID">Supply the ID of the network.</param>
        /// <param name="reservationID">Supply the ID of reservation.</param>
        /// <returns></returns>
        async Task IDNCService.DeleteReservationAsync(
            string networkID,
            string reservationID)
        {
            ServiceEventSource.Current.Message($"DeleteReservation: NetworkID: {networkID}, " +
                $"ReservationID: {reservationID}");
            try
            {
                using (ITransaction tx = RepositoryHelper.GetTransaction())
                {
                    NetworkConfiguration networkConfig = await networkRepositoryManager.GetNetworkConfiguration(
                        NetworkConfiguration.GetNetworkKey(networkID),
                        tx);
                    if (networkConfig == null)
                    {
                        throw new DncManagementException(
                            new ErrorInfo(
                                ErrorCode.NetworkNotFound,
                                $"Network configuration for network {networkID} not found",
                                ErrorType.NonRetriable));
                    }

                    string reservationKey = ReservationConfiguration.GetReservationKey(networkID, reservationID);
                    ReservationConfiguration reservationConfig =
                        await networkRepositoryManager.GetReservationConfiguration(
                            reservationKey,
                            tx,
                            LockMode.Update);
                    if (reservationConfig == null)
                    {
                        throw new DncManagementException(
                            new ErrorInfo(
                                ErrorCode.ReservationNotFound,
                                $"Reservation configuration for reservation {reservationID} not found",
                                ErrorType.NonRetriable));
                    }
                    reservationConfig.ReservationState = ReservationState.Release;
                    reservationConfig.Status = Status.Pending;
                    reservationConfig.MarkedForDelete = true;
                    await networkRepositoryManager.UpdateReservationConfigurationUntransacted(
                        reservationKey,
                        reservationConfig,
                        tx);

                    RequestContext requestContext = new RequestContext(
                        reservationKey,
                        networkID,
                        "",
                        RequestType.ReservationRequest);
                    requestContext.SetSubnetName(reservationConfig.AllocationRequest.SubnetName);

                    await RepositoryHelper.CreateOrUpdateReliableDictionary(
                        tx,
                        networkRepositoryManager.PendingRequestsReliableCollection,
                        reservationKey,
                        requestContext);

                    await RepositoryHelper.CommitTransactionAsync(tx);
                    ServiceEventSource.Current.Message($"Completed persisting delete request for reservation {reservationID}");
                }
            }
            catch (Exception exception)
            {
                ServiceEventSource.Current.Message($"Failed to process request for delete reservation {reservationID} " +
                    $" due to exception {exception}");
                throw;
            }
        }
    }
}
