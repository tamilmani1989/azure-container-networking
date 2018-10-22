//-----------------------------------------------------------------------
// <copyright file="RequestWatcher.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// 
//-----------------------------------------------------------------------

using Cns;
using DNCService;
using DNCService.API;
using Microsoft.ServiceFabric.Data.Notifications;
using NetworkManager.DataModel;
using NMAgent;
using NMAgent.Data;
using System;
using System.Collections.Generic;
using System.Threading;
using System.Threading.Tasks;

namespace NetworkManager
{
    /// <summary>
    /// Constants for local IP subnet.
    /// </summary>
    ///
    public class LocalIPSubnetConstants
    {
        public const string LocalIPSubnet = "169.254.0.0/16";
        public const string ImdsSubnet = "169.254.169.254/32";
        public const string LocalIPGateway = "169.254.0.1";
        public const int LocalIPSubnetPrefix = 16;
    }

    /// <summary>
    /// Class definition for request watcher.
    /// Monitors and processes pending requests.
    /// </summary>
    ///
    public class RequestWatcher
    {
        /// <summary>
        /// Handle to the host agent client.
        /// </summary>
        public HostAgentClient hostAgentClient;

        /// <summary>
        /// Handle to the cns client.
        /// </summary>
        public CnsClient cnsClient { get; private set; }

        /// <summary>
        /// Handle to the network manager repository.
        /// </summary>
        private NetworkRepositoryManager networkRepositoryManager;

        /// <summary>
        /// True if initialized.
        /// </summary>
        private bool isInitialized;

        /// <summary>
        /// Synchronization lock.
        /// </summary>
        private object syncLock;

        /// <summary>
        /// Azure storage client.
        /// </summary>
        private AzureStorageClient azureStorageClient;

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="hostAgentClient">Handle to the host agent client.</param>
        /// <param name="cnsClient">Handle to the cns client.</param>
        /// <param name="dncConfiguration">Supply the Dnc configuration.</param>
        public RequestWatcher(
            HostAgentClient hostAgentClient,
            CnsClient cnsClient,
            DncConfiguration dncConfiguration
            )
        {
            this.hostAgentClient = hostAgentClient;
            this.cnsClient = cnsClient;
            azureStorageClient = new AzureStorageClient(dncConfiguration);
            syncLock = new object();
        }

        /// <summary>
        /// Initialize.
        /// </summary>
        /// <param name="networkManager">Supply handle to the network repository manager.</param>
        /// <param name="cancellationToken">Cancellation token.</param>
        public async Task Init(
            NetworkRepositoryManager networkManager,
            CancellationToken cancellationToken)
        {
            networkRepositoryManager = networkManager;
            await azureStorageClient.Init(cancellationToken);
        }

        /// <summary>
        /// Helper to check if watcher is initialized.
        /// </summary>
        /// <returns>True if initialized, false otherwise.</returns>
        private bool IsInitialized()
        {
            lock(syncLock)
            {
                return isInitialized;
            }
        }

        /// <summary>
        /// Starts request watcher.
        /// </summary>
        public async void Start()
        {
            lock (syncLock)
            {
                isInitialized = true;
            }
            ICollection<string> requestIDs = await networkRepositoryManager.GetAllRequestIDs();
            foreach (string requestID in requestIDs)
            {
                ProcessRequest(requestID);
            }
        }

        /// <summary>
        /// Request dictionary change handler.
        /// </summary>
        /// <param name="sender">Supply the sender.</param>
        /// <param name="e">Supply the change event args.</param>
        public void OnDictionaryChangedHandler(object sender, NotifyDictionaryChangedEventArgs<string, DataModel.RequestContext> e)
        {
            switch (e.Action)
            {
                case NotifyDictionaryChangedAction.Add:
                    var addEvent = e as NotifyDictionaryItemAddedEventArgs<string, DataModel.RequestContext>;
                    this.ProcessAddOrUpdateNotification(addEvent.Key, addEvent.Value);
                    return;

                case NotifyDictionaryChangedAction.Update:
                    var updateEvent = e as NotifyDictionaryItemUpdatedEventArgs<string, DataModel.RequestContext>;
                    this.ProcessAddOrUpdateNotification(updateEvent.Key, updateEvent.Value);
                    return;

                default:
                    break;
            }
        }

        /// <summary>
        /// Process network join request.
        /// </summary>
        /// <param name="requestID">Supply the request ID</param>
        /// <param name="requestContext">Supply the request context.</param>
        /// <returns></returns>
        private async Task ProcessJoinRequestAsync(string requestID, RequestContext requestContext)
        {
            VirtualNetworkResponse virtualNetworkresponse = await hostAgentClient.JoinAndGetNetwork(
                                    requestContext.NetworkID);
            await networkRepositoryManager.UpdateNetworkConfiguration(
                requestID,
                requestContext.NetworkID,
                virtualNetworkresponse);
        }

        /// <summary>
        /// Process add or update request change notification.
        /// </summary>
        /// <param name="requestID">Supply the request ID</param>
        /// <param name="requestContext">Supply the request context.</param>
        /// <returns></returns>

        void ProcessAddOrUpdateNotification(string requestID, DataModel.RequestContext requestContext)
        {
            if(!IsInitialized())
            {
                return;
            }

            ServiceEventSource.Current.Message("Received update notification:{0}, RequestContext:{1}",
                requestID,
                requestContext);
            ProcessRequest(requestID);
        }

        /// <summary>
        /// Process a request.
        /// </summary>
        /// <param name="requestID">Supply the request ID</param>
        /// <returns></returns>
        private void ProcessRequest(string requestID)
        {
            Task.Run(async () =>
            {
                DataModel.RequestContext requestContext = null;
                try
                {
                    requestContext = await networkRepositoryManager.GetRequestContext(requestID);
                    if (requestContext == null)
                    {
                        ServiceEventSource.Current.Message("Request context associated with request:{0} is null. Deleting the request context",
                            requestID);
                        await networkRepositoryManager.DeleteRequestContext(requestID);
                    }
                    ServiceEventSource.Current.Message("Processing:{0}, RequestContext:{1}",
                        requestID,
                        requestContext);

                    switch (requestContext.Type)
                    {
                        case RequestType.VirtualNetworkRequest:
                            await ProcessJoinRequestAsync(requestID, requestContext);
                            break;

                        case RequestType.NetworkContainerRequest:
                            await ProcessNetworkContainerRequest(requestID, requestContext);
                            break;
                    }

                    // Reset the failure detail
                    await networkRepositoryManager.UpdateFailureDetail(
                        requestContext, null);
                }
                catch (Exception exception)
                {
                    ServiceEventSource.Current.Message("Failed to process Request {0}. Exception:{1}",
                        ((requestContext != null) ? requestContext.ToString() : "null"),
                        exception);

                    DncManagementException dncManagementException = (DncManagementException)exception;
                    bool isErrorRetriable = (dncManagementException.ErrorType == ErrorType.Retriable);

                    // Update the failure detail
                    await networkRepositoryManager.UpdateFailureDetail(
                        requestContext, dncManagementException.ErrorInfo, isErrorRetriable);

                    // Add to the failed queue only if the error is retriable
                    if (isErrorRetriable)
                    {
                        await networkRepositoryManager.MoveRequestToFailedQueue(
                            requestID,
                            hostAgentClient.DncConfiguration.DNCSettings.RequestRetryBaseInterval,
                            dncManagementException.ErrorInfo);
                    }
                    else
                    {
                        using (var tx = RepositoryHelper.GetTransaction())
                        {
                            // Remove the entry from pending queue for non-retriable error
                            await RepositoryHelper.RemoveFromReliableDictionary(
                                tx,
                                networkRepositoryManager.PendingRequestsReliableCollection,
                                requestID);

                            await networkRepositoryManager.DeleteRequestContext(requestContext.RequestID);

                            await RepositoryHelper.CommitTransactionAsync(tx);
                        }
                    }
                }
            }).ConfigureAwait(false);
        }

        /// <summary>
        /// Process network container request.
        /// </summary>
        /// <param name="requestID">Supply the request ID</param>
        /// <param name="requestContext">Supply the request context.</param>
        /// <returns></returns>

        private async Task ProcessNetworkContainerRequest(string requestID, RequestContext requestContext)
        {
            DataModel.NCConfiguration ncConfiguration = await networkRepositoryManager.GetNCConfigurationAsync(requestID);
            if(ncConfiguration.Status == DataModel.Status.Failed)
            {
                return;
            }

            ServiceEventSource.Current.Message($"########### NCState:{ncConfiguration.State} ###########");

            switch (ncConfiguration.State)
            {
                case NetworkContainerState.Reserve:
                    await ProcessNetworkContainerReserveIP(requestContext);
                    break;

                case NetworkContainerState.Associate:
                case NetworkContainerState.RefreshAssociation:
                    await ProcessNetworkContainerAssociate(requestContext);
                    break;

                case NetworkContainerState.Publish:
                    await ProcessNetworkContainerPublish(requestContext);
                    break;

                case NetworkContainerState.UnPublish:
                    await ProcessNetworkContainerUnPublish(requestContext);
                    break;

                case NetworkContainerState.Program:
                    await ProcessNetworkContainerProgram(requestContext);
                    break;

                case NetworkContainerState.Revert:
                    await ProcessNetworkContainerRevert(requestContext);
                    break;

                case NetworkContainerState.DisAssociate:
                    await ProcessNetworkContainerDisAssociate(requestContext);
                    break;

                case NetworkContainerState.AllocateLocalIP:
                    await ProcessNetworkContainerAllocateLocalIP(requestContext);
                    break;
            }
        }

        /// <summary>
        /// Process network container disassociate request.
        /// </summary>
        /// <param name="requestContext">Supply the request context.</param>
        /// <returns></returns>
        private async Task ProcessNetworkContainerDisAssociate(RequestContext requestContext)
        {
            NetworkContainerInfo networkContainerInfo = await networkRepositoryManager.GetNetworkContainerInfo(
                requestContext.RequestID);
            if (networkContainerInfo == null)
            {
                throw new DncManagementException(
                    new ErrorInfo(ErrorCode.NetworkContainerNotFound,
                        "Failed to disassociate NC due to failure to retrieve NC configuration",
                        ErrorType.NonRetriable));
            }

            await cnsClient.DeleteNetworkContainer(networkContainerInfo);

            await networkRepositoryManager.UpdateNetworkContainerState(
                requestContext,
                NetworkContainerState.DisAssociate,
                NetworkContainerState.UnPublish,
                Status.Pending,
                false);
        }

        /// <summary>
        /// Process network container revert request.
        /// </summary>
        /// <param name="requestContext">Supply the request context.</param>
        /// <returns></returns>
        private async Task ProcessNetworkContainerRevert(RequestContext requestContext)
        {
            NetworkContainerInfo networkContainerInfo =
                await networkRepositoryManager.GetNetworkContainerInfo(requestContext.RequestID);
            if (networkContainerInfo == null)
            {
                throw new DncManagementException(
                    new ErrorInfo(ErrorCode.NetworkContainerNotFound,
                        "Failed to delete NC due to failure to retrieve NC configuration",
                        ErrorType.NonRetriable));
            }

            try
            {
                // Delete the IP allocation from azure storage
                if (isSharedSubnetDelegationEnabled())
                {
                    NCConfiguration ncConfig =
                        await networkRepositoryManager.GetNCConfigurationAsync(requestContext.RequestID);
                    if (ncConfig == null)
                    {
                        throw new DncManagementException(
                            new ErrorInfo(ErrorCode.NetworkContainerNotFound,
                                "Failed to delete NC due to failure to find network container config",
                                ErrorType.NonRetriable));
                    }

                    string reservationKey = ReservationConfiguration.GetReservationKey(
                        networkContainerInfo.NetworkID,
                        ncConfig.ReservationID);
                    ReservationConfiguration reservationConfig =
                        await networkRepositoryManager.GetReservationConfiguration(reservationKey);
                    if (reservationConfig == null)
                    {
                        throw new DncManagementException(
                            new ErrorInfo(ErrorCode.ReservationNotFound,
                                "Failed to delete NC due to failure to find reservation",
                                ErrorType.NonRetriable));
                    }

                    string subnetKey = SubnetConfiguration.GetSubnetConfigurationKey(
                        requestContext.NetworkID,
                        reservationConfig.AllocationRequest.SubnetName);

                    await azureStorageClient.DeleteAllocatedIP(
                        subnetKey,
                        azureStorageClient.EncodeIPAllocationKey(reservationConfig.AllocatedSubnet),
                        reservationKey);
                }

                await networkRepositoryManager.DeleteNetworkContainer(networkContainerInfo, true);
            }
            catch (Exception exception)
            {
                string errorMsg = (exception.InnerException == null) ?
                    exception.Message : exception.InnerException.Message;
                throw new DncManagementException(
                    new ErrorInfo(ErrorCode.NetworkContainerNotFound,
                        $"Failed to delete NC due to exception {errorMsg}",
                        ErrorType.Retriable));
            }
        }

        /// <summary>
        /// Process network container program request.
        /// </summary>
        /// <param name="requestContext">Supply the request context.</param>
        /// <returns></returns>
        private async Task ProcessNetworkContainerProgram(RequestContext requestContext)
        {
            NetworkContainerInfo networkContainerInfo = await networkRepositoryManager.GetNetworkContainerInfo(
                requestContext.RequestID);
            if (networkContainerInfo == null)
            {
                throw new DncManagementException(
                    new ErrorInfo(ErrorCode.NetworkContainerNotFound,
                        "Failed to program NC due to failure to retrieve NC configuration",
                        ErrorType.NonRetriable));
            }

            await networkRepositoryManager.UpdateNetworkContainerState(
                requestContext,
                NetworkContainerState.Program,
                NetworkContainerState.Program,
                Status.Completed,
                true);
        }

        /// <summary>
        /// Process network container unpublish request.
        /// </summary>
        /// <param name="requestContext">Supply the request context.</param>
        /// <returns></returns>
        private async Task ProcessNetworkContainerUnPublish(RequestContext requestContext)
        {
            NetworkContainerInfo networkContainerInfo = await networkRepositoryManager.GetNetworkContainerInfo(
                requestContext.RequestID);
            if (networkContainerInfo == null)
            {
                throw new DncManagementException(
                    new ErrorInfo(ErrorCode.NetworkContainerNotFound,
                        "Failed to unpublish NC due to failure to retrieve NC configuration",
                        ErrorType.NonRetriable));
            }

            await hostAgentClient.DeleteNetworkContainer(networkContainerInfo);

            await networkRepositoryManager.UpdateNetworkContainerState(
                requestContext,
                NetworkContainerState.UnPublish,
                NetworkContainerState.Revert,
                Status.Pending,
                false);
        }

        /// <summary>
        /// Process network container publish request.
        /// </summary>
        /// <param name="requestContext">Supply the request context.</param>
        /// <returns></returns>
        private async Task ProcessNetworkContainerPublish(RequestContext requestContext)
        {
            NetworkContainerInfo networkContainerInfo = await networkRepositoryManager.GetNetworkContainerInfo(
                requestContext.RequestID);

            if (networkContainerInfo == null)
            {
                throw new DncManagementException(
                    new ErrorInfo(ErrorCode.NetworkContainerNotFound,
                        "Failed to publish NC due to failure to retrieve NC configuration",
                        ErrorType.NonRetriable));
            }

            await hostAgentClient.CreateNetworkContainer(networkContainerInfo);

            await networkRepositoryManager.UpdateNetworkContainerState(
                requestContext,
                NetworkContainerState.Publish,
                NetworkContainerState.AllocateLocalIP,
                Status.Pending,
                false);
        }

        /// <summary>
        /// Process network container associate request.
        /// </summary>
        /// <param name="requestContext">Supply the request context.</param>
        /// <returns></returns>
        private async Task ProcessNetworkContainerAssociate(RequestContext requestContext)
        {
            NetworkContainerInfo networkContainerInfo =
                await networkRepositoryManager.GetNetworkContainerInfo(requestContext.RequestID);
            NCConfiguration ncConfig =
                await networkRepositoryManager.GetNCConfigurationAsync(requestContext.RequestID);

            if ((networkContainerInfo == null) ||
                (ncConfig == null))
            {
                throw new DncManagementException(
                    new ErrorInfo(ErrorCode.NetworkContainerNotFound,
                        "Failed to associate NC due to failure to retrieve NC configuration",
                        ErrorType.NonRetriable));
            }

            cnsClient.ParseTagsIntoCnetAddresses(ref networkContainerInfo, ref ncConfig);

            await cnsClient.CreateNetworkContainer(networkContainerInfo);

            // Retrieve the latest NC config as current copy could have been updated while calling CNS
            NCConfiguration ncConfigLatest =
                await networkRepositoryManager.GetNCConfigurationAsync(requestContext.RequestID);
            if (ncConfigLatest == null)
            {
                throw new DncManagementException(
                    new ErrorInfo(ErrorCode.NetworkContainerNotFound,
                        "Failed to associate NC due to failure to retrieve NC configuration",
                        ErrorType.NonRetriable));
            }

            ncConfig.MarkedForDelete = ncConfigLatest.MarkedForDelete;

            // Update NC configuration
            await networkRepositoryManager.UpdateNCConfiguration(
                requestContext,
                ncConfig);

            await networkRepositoryManager.UpdateNetworkContainerState(
                requestContext,
                networkContainerInfo.NetworkContainerState,
                NetworkContainerState.Program,
                Status.Pending,
                false);
        }

        /// <summary>
        /// Process network container allocate local IP request.
        /// </summary>
        /// <param name="requestContext">Supply the request context.</param>
        /// <returns></returns>
        private async Task ProcessNetworkContainerAllocateLocalIP(RequestContext requestContext)
        {
            NetworkContainerInfo networkContainerInfo = await networkRepositoryManager.GetNetworkContainerInfo(
                requestContext.RequestID);

            if (networkContainerInfo == null)
            {
                throw new DncManagementException(
                    new ErrorInfo(ErrorCode.NetworkContainerNotFound,
                        "Failed to allocate host local IP to NC due to failure to retrieve NC configuration",
                        ErrorType.NonRetriable));
            }

            List<IPSubnet> allocatedLocalSubnets = new List<IPSubnet>();
            try
            {
                NodeConfiguration nodeConfiguration =
                    await networkRepositoryManager.GetNodeConfiguration(requestContext.NodeID);

                if (nodeConfiguration == null)
                {
                    return;
                }

                if (nodeConfiguration.DeployedNCMap.Count != 0)
                {
                    // Compile a list of allocated local addresses from the returned node configuration
                    foreach (KeyValuePair<string, IReadOnlyDictionary<string, string>> networkNCKvp
                        in nodeConfiguration.DeployedNCMap)
                    {
                        string networkID = networkNCKvp.Key;
                        foreach (string ncID in networkNCKvp.Value.Keys)
                        {
                            NCConfiguration ncConfiguration =
                                await networkRepositoryManager.GetNetworkContainerConfiguration(
                                    networkID,
                                    ncID);

                            if (ncConfiguration == null)
                            {
                                continue;
                            }

                            if (!string.IsNullOrWhiteSpace(ncConfiguration.LocalIP))
                            {
                                allocatedLocalSubnets.Add(IPSubnet.Parse(ncConfiguration.LocalIP));
                            }
                        }
                    }
                }
            }
            catch (Exception exception)
            {
                string errorMsg = (exception.InnerException == null) ?
                    exception.Message : exception.InnerException.Message;
                throw new DncManagementException(
                    new ErrorInfo(ErrorCode.AllocationFailed,
                        $"Failed to process ProcessNetworkContainerAllocateLocalIP due to excpetion {errorMsg}",
                        ErrorType.Retriable));
            }

            try
            {
                // Add 169.254.169.254 to the allocated list
                allocatedLocalSubnets.Add(IPSubnet.Parse(LocalIPSubnetConstants.ImdsSubnet));

                IPSubnet localIP = AllocationHelper.AllocateSubnetFromBaseSubnet(
                    IPSubnet.Parse(LocalIPSubnetConstants.LocalIPSubnet),
                    allocatedLocalSubnets,
                    32);

                await networkRepositoryManager.SetNetworkContainerLocalIP(requestContext, localIP.ToString());
            }
            catch (Exception exception)
            {
                string errorMsg = (exception.InnerException == null) ?
                    exception.Message : exception.InnerException.Message;
                throw new DncManagementException(
                    new ErrorInfo(ErrorCode.AllocationFailed,
                        $"Failed to process ProcessNetworkContainerAllocateLocalIP due to excpetion {errorMsg}",
                        ErrorType.NonRetriable));
            }

            await networkRepositoryManager.UpdateNetworkContainerState(
                requestContext,
                NetworkContainerState.AllocateLocalIP,
                NetworkContainerState.Associate,
                Status.Pending,
                false);
        }

        /// <summary>
        /// Process reserve an IP for network container.
        /// </summary>
        /// <param name="requestContext">Supply the request context.</param>
        /// <returns></returns>
        private async Task ProcessNetworkContainerReserveIP(RequestContext requestContext)
        {
            NetworkContainerInfo networkContainerInfo =
                await networkRepositoryManager.GetNetworkContainerInfo(requestContext.RequestID);

            if (networkContainerInfo == null)
            {
                throw new DncManagementException(
                    new ErrorInfo(ErrorCode.NetworkContainerNotFound,
                        "Failed to reserve IP for NC due to failure to retrieve NC configuration",
                        ErrorType.NonRetriable));
            }

            NCConfiguration ncConfig =
                await networkRepositoryManager.GetNCConfigurationAsync(requestContext.RequestID);
            if (ncConfig == null)
            {
                throw new DncManagementException(
                    new ErrorInfo(ErrorCode.NetworkContainerNotFound,
                        "Failed to reserve IP for NC due to failure to retrieve NC configuration",
                        ErrorType.NonRetriable));
            }

            string reservationKey = ReservationConfiguration.GetReservationKey(
                requestContext.NetworkID,
                ncConfig.ReservationID);
            ReservationConfiguration reservationConfig =
                await networkRepositoryManager.GetReservationConfiguration(reservationKey);
            if (reservationConfig == null)
            {
                throw new DncManagementException(
                    new ErrorInfo(ErrorCode.ReservationNotFound,
                        "Failed to reserve IP for NC due to failure to find reservation",
                        ErrorType.NonRetriable));
            }

            string subnetKey = SubnetConfiguration.GetSubnetConfigurationKey(
                requestContext.NetworkID,
                reservationConfig.AllocationRequest.SubnetName);
            SubnetConfiguration subnetConfig =
                await networkRepositoryManager.GetSubnetConfiguration(subnetKey);
            if (subnetConfig == null)
            {
                throw new DncManagementException(
                    new ErrorInfo(ErrorCode.SubnetNotFound,
                        "Failed to reserve IP for NC due to failure to find subnet configuration",
                        ErrorType.NonRetriable));
            }

            string existingSubnetAllocation = null;
            List<IPSubnet> allocatedSubnets = new List<IPSubnet>();

            // If sharedSubnetDelegation is enabled call azureStorage to get all the allocated subnets
            if (isSharedSubnetDelegationEnabled())
            {
                IReadOnlyDictionary<string, string> allocatedSubnetStrings =
                    await azureStorageClient.GetAllocatedSubnets(subnetKey);

                foreach (KeyValuePair<string, string> allocatedSubnetKvp in allocatedSubnetStrings)
                {
                    var subnetAllocation = IPSubnet.Parse(
                        azureStorageClient.DecodeIPAllocationKey(allocatedSubnetKvp.Key));
                    if (reservationKey.Equals(allocatedSubnetKvp.Value))
                    {
                        existingSubnetAllocation = subnetAllocation.ToString();
                        ServiceEventSource.Current.Message($"Found allocated IP " +
                            $"{reservationConfig.AllocatedSubnet} for current reservation");
                        break;
                    }
                    allocatedSubnets.Add(subnetAllocation);
                }
            }
            else
            {
                foreach (KeyValuePair<string, string> reservationKvp in subnetConfig.Reservations)
                {
                    allocatedSubnets.Add(IPSubnet.Parse(reservationKvp.Value));
                    ServiceEventSource.Current.Message($"Reservation Id:{reservationKvp.Key} " +
                        $"Value:{reservationKvp.Value}");
                }
            }

            if (string.IsNullOrWhiteSpace(existingSubnetAllocation))
            {
                // Allocate a new IP from base subnet
                reservationConfig.AllocatedSubnet =
                    AllocationHelper.AllocateSubnetFromBaseSubnet(
                        IPSubnet.Parse(subnetConfig.IPSubnet),
                        allocatedSubnets,
                        32).ToString();

                if (isSharedSubnetDelegationEnabled())
                {
                    // Try to save new allocation in IPAllocationsTable
                    bool insertIPAllocationSuccess =
                        await azureStorageClient.TryInsertAllocatedIP(
                            subnetKey,
                            azureStorageClient.EncodeIPAllocationKey(reservationConfig.AllocatedSubnet),
                            reservationKey);
                    if (!insertIPAllocationSuccess)
                    {
                        throw new DncManagementException(
                            new ErrorInfo(ErrorCode.AllocationFailed,
                                "Failed to insert allocated IP into the table. " +
                                $"Possible IP contention. Retrying...",
                                ErrorType.Retriable));
                    }
                }
            }
            else
            {
                // Assign the existing subnet allocation
                reservationConfig.AllocatedSubnet = existingSubnetAllocation;
            }

            ServiceEventSource.Current.Message($"Reserved IP: {reservationConfig.AllocatedSubnet}");

            await networkRepositoryManager.UpdateReservationConfiguration(
                reservationKey,
                reservationConfig);

            // Update subnet config with the new allocated subnet and save the
            // updated subnet config.
            subnetConfig.AddReservation(
                reservationConfig.ReservationID,
                reservationConfig.AllocatedSubnet);

            await networkRepositoryManager.UpdateSubnetConfiguration(
                subnetKey,
                subnetConfig);

            await networkRepositoryManager.UpdateNetworkContainerState(
                requestContext,
                NetworkContainerState.Reserve,
                NetworkContainerState.Publish,
                Status.Pending,
                false);
        }

        /// <summary>
        /// Returns if the shared subnet delegation is enabled
        /// </summary>
        /// <returns>True if enabled, false otherwise.</returns>
        private bool isSharedSubnetDelegationEnabled()
        {
            return azureStorageClient.IsSharedSubnetDelegationEnabled();
        }
    }
}
