//-----------------------------------------------------------------------
// <copyright file="RequestWatcher.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// 
//-----------------------------------------------------------------------

using Cns;
using DNCService;
using DNCService.API;
using Microsoft.ServiceFabric.Data;
using Microsoft.ServiceFabric.Data.Collections;
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
        /// Configuration for DNC
        /// </summary>
        private DncConfiguration dncConfiguration;

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

        private TagsCacheManager tagsCacheManager;

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
        /// <param name="tagsCacheManager">Handle to tags cache manager.</param>
        /// <param name="dncConfiguration">Supply the Dnc configuration.</param>
        public RequestWatcher(
            HostAgentClient hostAgentClient,
            CnsClient cnsClient,
            TagsCacheManager tagsCacheManager,
            DncConfiguration dncConfiguration)
        {
            this.hostAgentClient = hostAgentClient;
            this.cnsClient = cnsClient;
            this.dncConfiguration = dncConfiguration;
            this.tagsCacheManager = tagsCacheManager;
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
            VirtualNetworkResponse virtualNetworkResponse = await hostAgentClient.JoinAndGetNetwork(
                                    requestContext.NetworkID);
            await networkRepositoryManager.UpdateNetworkConfiguration(
                requestID,
                requestContext.NetworkID,
                virtualNetworkResponse);
            tagsCacheManager.CheckAndAddTagsCache(virtualNetworkResponse);
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
                RequestContext requestContext = null;
                try
                {
                    requestContext = await networkRepositoryManager.GetRequestContext(requestID);
                    if (requestContext == null)
                    {
                        ServiceEventSource.Current.Message($"Request context associated with request: " +
                            $"{requestID} is null. Deleting the request context");
                        await networkRepositoryManager.DeleteRequestContext(requestID);
                    }
                    ServiceEventSource.Current.Message($"ProcessRequest for RequestContext: {requestContext}");

                    switch (requestContext.Type)
                    {
                        case RequestType.VirtualNetworkRequest:
                            await ProcessJoinRequestAsync(requestID, requestContext);
                            break;

                        case RequestType.NetworkContainerRequest:
                            await ProcessNetworkContainerRequest(requestID, requestContext);
                            break;

                        case RequestType.ReservationRequest:
                            await ProcessReservationRequest(requestID, requestContext);
                            break;

                        default:
                            throw new DncManagementException(
                                new ErrorInfo(ErrorCode.StatusInternalServerError,
                                    $"Failed to process request: {requestContext} due to invalid request type: " +
                                    $"{requestContext.Type}",
                                    ErrorType.NonRetriable));
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

                            await networkRepositoryManager.DeleteRequestContextUntransacted(
                                requestContext.RequestID,
                                tx);

                            await RepositoryHelper.CommitTransactionAsync(tx);
                        }
                    }
                }
            }).ConfigureAwait(false);
        }

        /// <summary>
        /// Process IP reservation request.
        /// </summary>
        /// <param name="requestID">Supply the request ID</param>
        /// <param name="requestContext">Supply the request context.</param>
        /// <returns></returns>
        private async Task ProcessReservationRequest(
            string requestID,
            RequestContext requestContext)
        {
            ReservationConfiguration reservationConfig;
            using (var tx = RepositoryHelper.GetTransaction())
            {
                reservationConfig = await networkRepositoryManager.GetReservationConfiguration(
                    requestContext.RequestID,
                    tx);
                if (reservationConfig == null)
                {
                    throw new DncManagementException(
                        new ErrorInfo(ErrorCode.ReservationNotFound,
                            "Failed to process reservation request due to failure to find reservation",
                            ErrorType.NonRetriable));
                }
            }

            if (reservationConfig.Status == Status.Failed)
            {
                ServiceEventSource.Current.Message($"Failed to process reservation request: " +
                    $"{requestContext} due to invalid state {reservationConfig.Status}");
                return;
            }

            ServiceEventSource.Current.Message($"########### ReservationState: {reservationConfig.ReservationState}" +
                $" ### Key: {requestID} ###########");

            switch (reservationConfig.ReservationState)
            {
                case ReservationState.Allocate:
                    await CreateReservationRequest(requestContext);
                    break;

                case ReservationState.Release:
                    await DeleteReservationRequest(requestContext);
                    break;

                default:
                    throw new DncManagementException(
                        new ErrorInfo(ErrorCode.StatusInternalServerError,
                            $"Failed to process reservation request: {requestContext} due to invalid state" +
                            $" {reservationConfig.ReservationState}",
                            ErrorType.NonRetriable));
            }
        }

        /// <summary>
        /// Create IP reservation request.
        /// </summary>
        /// <param name="requestContext">Supply the request context.</param>
        /// <returns></returns>
        private async Task CreateReservationRequest(
            RequestContext requestContext)
        {
            using (var tx = RepositoryHelper.GetTransaction())
            {
                NodeConfiguration nodeConfig = null;
                if (!string.IsNullOrWhiteSpace(requestContext.NodeID))
                {
                    string nodeKey = NodeConfiguration.GetNodeKey(requestContext.NodeID);
                    nodeConfig = await RepositoryHelper.GetFromReliableDictionary(
                        tx,
                        networkRepositoryManager.NodeConfigReliableCollection,
                        nodeKey,
                        LockMode.Update);
                    if (nodeConfig == null)
                    {
                        throw new DncNetworkContainerException(
                            requestContext.NetworkID,
                            requestContext.NodeID,
                            new ErrorInfo(ErrorCode.NodeNotFound, "Node not found", ErrorType.NonRetriable));
                    }
                }

                string subnetKey = SubnetConfiguration.GetSubnetConfigurationKey(
                requestContext.NetworkID,
                requestContext.SubnetName);
                SubnetConfiguration subnetConfig = await networkRepositoryManager.GetSubnetConfiguration(
                    subnetKey,
                    tx,
                    LockMode.Update);
                if (subnetConfig == null)
                {
                    throw new DncManagementException(
                        new ErrorInfo(ErrorCode.SubnetNotFound,
                            "Failed to process reservation request due to failure to find subnet configuration",
                            ErrorType.NonRetriable));
                }

                ReservationConfiguration reservationConfig = await networkRepositoryManager.GetReservationConfiguration(
                    requestContext.RequestID,
                    tx,
                    LockMode.Update);
                if (reservationConfig == null)
                {
                    throw new DncManagementException(
                        new ErrorInfo(ErrorCode.ReservationNotFound,
                            "Failed to process create reservation request due to failure to find reservation",
                            ErrorType.NonRetriable));
                }

                if ((reservationConfig.ReservationState != ReservationState.Allocate) &&
                    (reservationConfig.Status != Status.Pending))
                {
                    ServiceEventSource.Current.Message($"Incorrect Reservation state: {reservationConfig.ReservationState} " +
                        $"status: {reservationConfig.Status}. Expected pending allocate.");
                    return;
                }

                await ReserveIPUntransacted(
                    subnetKey,
                    subnetConfig,
                    requestContext.RequestID,
                    reservationConfig,
                    nodeConfig,
                    tx);

                await RepositoryHelper.CommitTransactionAsync(tx);
            }
        }

        /// <summary>
        /// Delete IP reservation request.
        /// </summary>
        /// <param name="requestContext">Supply the request context.</param>
        /// <returns></returns>
        private async Task DeleteReservationRequest(RequestContext requestContext)
        {
            using (var tx = RepositoryHelper.GetTransaction())
            {
                ReservationConfiguration reservationConfig = await networkRepositoryManager.GetReservationConfiguration(
                    requestContext.RequestID,
                    tx,
                    LockMode.Update);
                if (reservationConfig == null)
                {
                    throw new DncManagementException(
                        new ErrorInfo(ErrorCode.ReservationNotFound,
                            "Failed to process delete reservation request due to failure to find reservation",
                            ErrorType.NonRetriable));
                }

                if ((reservationConfig.ReservationState != ReservationState.Release) &&
                    (reservationConfig.Status != Status.Pending))
                {
                    ServiceEventSource.Current.Message($"Incorrect Reservation state: {reservationConfig.ReservationState} " +
                        $"status: {reservationConfig.Status}. Expected pending release.");
                    return;
                }

                if (!string.IsNullOrWhiteSpace(reservationConfig.AssociatedNCID))
                {
                    ServiceEventSource.Current.Message($"Failed to delete reservation {requestContext.RequestID}. " +
                        $"Reservation is in use with container {reservationConfig.AssociatedNCID}");
                    throw new DncManagementException(
                        new ErrorInfo(ErrorCode.ReservationNotFound,
                            $"Failed to process delete reservation request as reservation is in use with the" +
                            $" container {reservationConfig.AssociatedNCID}",
                            ErrorType.NonRetriable));
                }

                // Delete the IP allocation from azure storage
                if (isSharedSubnetDelegationEnabled())
                {
                    string subnetKey = SubnetConfiguration.GetSubnetConfigurationKey(
                        requestContext.NetworkID,
                        reservationConfig.AllocationRequest.SubnetName);

                    await azureStorageClient.DeleteAllocatedIP(
                        subnetKey,
                        azureStorageClient.EncodeIPAllocationKey(reservationConfig.AllocatedSubnet),
                        requestContext.RequestID);
                }

                await networkRepositoryManager.DeleteReservationUntransacted(
                    reservationConfig,
                    requestContext,
                    tx);

                await RepositoryHelper.CommitTransactionAsync(tx);
            }
        }

        /// <summary>
        /// Reserve IP.
        /// </summary>
        /// <param name="subnetKey">Subney key</param>
        /// <param name="subnetConfig">Subnet configuration</param>
        /// <param name="reservationKey">Reservation key</param>
        /// <param name="reservationConfig">Reservation configuration</param>
        /// <param name="nodeConfig">Node configuration</param>
        /// <param name="tx">transaction object</param>
        /// <returns></returns>
        private async Task ReserveIPUntransacted(
            string subnetKey,
            SubnetConfiguration subnetConfig,
            string reservationKey,
            ReservationConfiguration reservationConfig,
            NodeConfiguration nodeConfig,
            ITransaction tx)
        {
            if (!string.IsNullOrWhiteSpace(reservationConfig.AllocatedSubnet))
            {
                ServiceEventSource.Current.Message($"Reservation request {reservationConfig.AllocationRequest.ToString()}" +
                    $" is already processed and has an allocated IP Address: {reservationConfig.AllocatedSubnet}");
                return;
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
                            $"{existingSubnetAllocation} for current reservation");
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
                string ipConstraint = reservationConfig.AllocationRequest.IPConstraint;
                if (string.IsNullOrWhiteSpace(ipConstraint))
                {
                    // Allocate a new IP from base subnet
                    reservationConfig.AllocatedSubnet = AllocationHelper.AllocateSubnetFromBaseSubnet(
                        IPSubnet.Parse(subnetConfig.IPSubnet),
                        allocatedSubnets,
                        32).ToString();
                }
                else
                {
                    IPSubnet allocatedSubnet = AllocationHelper.GetIPSubnetForIPAllocationConstraint(ipConstraint);
                    IPSubnet baseSubnet = IPSubnet.Parse(subnetConfig.IPSubnet);
                    if (allocatedSubnets.Contains(allocatedSubnet) ||
                        !AllocationHelper.IsSubnetContainedInBaseSubnet(baseSubnet, allocatedSubnet))
                    {
                        throw new DncManagementException(
                            new ErrorInfo(ErrorCode.AllocationFailed,
                                $"Failed to allocate the constraint IP: {ipConstraint}",
                                ErrorType.NonRetriable));
                    }
                    reservationConfig.AllocatedSubnet = allocatedSubnet.ToString();
                }

                if (isSharedSubnetDelegationEnabled())
                {
                    // Try to save new allocation in IPAllocationsTable
                    bool insertIPAllocationSuccess = await azureStorageClient.TryInsertAllocatedIP(
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

            if (nodeConfig != null)
            {
                nodeConfig.AddConstrainedReservation(reservationConfig.ReservationID, reservationConfig.NetworkID);
                await networkRepositoryManager.UpdateNodeConfigurationUntransacted(
                    NodeConfiguration.GetNodeKey(reservationConfig.AllocationRequest.NodeConstraint),
                    nodeConfig,
                    tx);
            }

            ServiceEventSource.Current.Message($"Reserved IP: {reservationConfig.AllocatedSubnet} for key: {reservationKey}");

            reservationConfig.ReservationState = ReservationState.Allocate;
            reservationConfig.Status = Status.Completed;

            await networkRepositoryManager.UpdateReservationConfigurationUntransacted(
                reservationKey,
                reservationConfig,
                tx);

            // Update subnet config with the new allocated subnet and save the
            // updated subnet config.
            subnetConfig.AddReservation(
                reservationConfig.ReservationID,
                reservationConfig.AllocatedSubnet);

            await networkRepositoryManager.UpdateSubnetConfigurationUntransacted(
                subnetKey,
                subnetConfig,
                tx);
        }

        /// <summary>
        /// Process network container request.
        /// </summary>
        /// <param name="requestID">Supply the request ID</param>
        /// <param name="requestContext">Supply the request context.</param>
        /// <returns></returns>
        private async Task ProcessNetworkContainerRequest(
            string requestID,
            RequestContext requestContext)
        {
            NCConfiguration ncConfiguration = await networkRepositoryManager.GetNetworkContainerConfiguration(requestID);
            if (ncConfiguration == null)
            {
                throw new DncManagementException(
                    new ErrorInfo(ErrorCode.NetworkContainerNotFound,
                        "Failed to process network container request due to failure to find configuration",
                        ErrorType.NonRetriable));
            }

            if (ncConfiguration.Status == Status.Failed)
            {
                ServiceEventSource.Current.Message($"Failed to process network container request: " +
                    $"{requestContext} due to invalid container state {ncConfiguration.State}");
                return;
            }

            ServiceEventSource.Current.Message($"########### NCState: {ncConfiguration.State} ## Key: {requestID} ###########");

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

                default:
                    throw new DncManagementException(
                        new ErrorInfo(ErrorCode.StatusInternalServerError,
                            $"Failed to process network container request: {requestContext} due to invalid container state" +
                            $" {ncConfiguration.State}",
                            ErrorType.NonRetriable));
            }
        }

        /// <summary>
        /// Process network container disassociate request.
        /// </summary>
        /// <param name="requestContext">Supply the request context.</param>
        /// <returns></returns>
        private async Task ProcessNetworkContainerDisAssociate(
            RequestContext requestContext)
        {
            using (var tx = RepositoryHelper.GetTransaction())
            {
                NodeConfiguration nodeConfig = await networkRepositoryManager.GetNodeConfiguration(
                        requestContext.NodeID,
                        tx);
                if (nodeConfig == null)
                {
                    ServiceEventSource.Current.Message($"Failed to retrieve node configuration for NC: " +
                        $"{requestContext.RequestID}");
                    return;
                }

                NCConfiguration ncConfig = await networkRepositoryManager.GetNetworkContainerConfiguration(
                    requestContext.RequestID,
                    tx,
                    LockMode.Update);
                if (ncConfig == null)
                {
                    throw new DncManagementException(
                        new ErrorInfo(ErrorCode.NetworkContainerNotFound,
                            "Failed to disassociate NC due to failure to retrieve NC configuration",
                            ErrorType.NonRetriable));
                }

                // Abort the operation if the state of the network container is altered by another thread.
                if (ncConfig.State != NetworkContainerState.DisAssociate)
                {
                    ServiceEventSource.Current.Message($"NC state is {ncConfig.State} instead of disassociate." +
                        $" Aborting the operation");
                    return;
                }

                string nodeInterfaceID =
                    string.IsNullOrWhiteSpace(ncConfig.Request.AssociationInfo.InterfaceID) ?
                        nodeConfig.NodeInfo.IPAddresses[0] :
                        ncConfig.Request.AssociationInfo.InterfaceID;
                string networkContainerID = NCConfiguration.GetNetworkContainerIDFromKey(requestContext.RequestID);
                await cnsClient.DeleteNetworkContainer(
                    networkContainerID,
                    nodeInterfaceID);

                await networkRepositoryManager.UpdateNetworkContainerStateUntransacted(
                    ncConfig,
                    requestContext,
                    NetworkContainerState.DisAssociate,
                    NetworkContainerState.UnPublish,
                    Status.Pending,
                    false,
                    tx);

                await RepositoryHelper.CommitTransactionAsync(tx);
            }
        }

        /// <summary>
        /// Process network container revert request.
        /// </summary>
        /// <param name="requestContext">Supply the request context.</param>
        /// <returns></returns>
        private async Task ProcessNetworkContainerRevert(
            RequestContext requestContext)
        {
            using (var tx = RepositoryHelper.GetTransaction())
            {
                NodeConfiguration nodeConfig = await networkRepositoryManager.GetNodeConfiguration(
                    requestContext.NodeID,
                    tx,
                    LockMode.Update);

                string subnetKey = SubnetConfiguration.GetSubnetConfigurationKey(
                    requestContext.NetworkID,
                    requestContext.SubnetName);
                SubnetConfiguration subnetConfig = await networkRepositoryManager.GetSubnetConfiguration(
                    subnetKey,
                    tx,
                    LockMode.Update);

                NCConfiguration ncConfig = await networkRepositoryManager.GetNetworkContainerConfiguration(
                    requestContext.RequestID,
                    tx,
                    LockMode.Update);
                if (ncConfig == null)
                {
                    throw new DncManagementException(
                        new ErrorInfo(ErrorCode.NetworkContainerNotFound,
                            "Failed to delete NC due to failure to retrieve NC configuration",
                            ErrorType.NonRetriable));
                }

                // Abort the operation if the state of the network container is altered by another thread.
                if (ncConfig.State != NetworkContainerState.Revert)
                {
                    ServiceEventSource.Current.Message($"NC state is {ncConfig.State} instead of revert." +
                        $" Aborting the operation");
                    return;
                }

                string reservationKey = ReservationConfiguration.GetReservationKey(
                    requestContext.NetworkID,
                    ncConfig.ReservationID);
                ReservationConfiguration reservationConfig = await networkRepositoryManager.GetReservationConfiguration(
                    reservationKey,
                    tx,
                    LockMode.Update);
                if (reservationConfig == null)
                {
                    throw new DncManagementException(
                        new ErrorInfo(ErrorCode.ReservationNotFound,
                            "Failed to delete NC due to failure to find reservation",
                            ErrorType.NonRetriable));
                }

                try
                {
                    // Delete the IP allocation from azure storage if NC was created with
                    // implicit reservation
                    if (ncConfig.ImplicitReservation &&
                        isSharedSubnetDelegationEnabled())
                    {
                        await azureStorageClient.DeleteAllocatedIP(
                            subnetKey,
                            azureStorageClient.EncodeIPAllocationKey(reservationConfig.AllocatedSubnet),
                            reservationKey);
                    }

                    await networkRepositoryManager.DeleteNetworkContainerUntransacted(
                        nodeConfig,
                        subnetConfig,
                        reservationConfig,
                        ncConfig,
                        requestContext,
                        true,
                        tx);

                    await RepositoryHelper.CommitTransactionAsync(tx);
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
        }

        /// <summary>
        /// Process network container program request.
        /// </summary>
        /// <param name="requestContext">Supply the request context.</param>
        /// <returns></returns>
        private async Task ProcessNetworkContainerProgram(
            RequestContext requestContext)
        {
            using (var tx = RepositoryHelper.GetTransaction())
            {
                NCConfiguration ncConfig = await networkRepositoryManager.GetNetworkContainerConfiguration(
                    requestContext.RequestID,
                    tx,
                    LockMode.Update);
                if (ncConfig == null)
                {
                    throw new DncManagementException(
                        new ErrorInfo(ErrorCode.NetworkContainerNotFound,
                            "Failed to program NC due to failure to retrieve NC configuration",
                            ErrorType.NonRetriable));
                }

                // Abort the operation if the state of the network container is altered by another thread.
                if (ncConfig.State != NetworkContainerState.Program)
                {
                    ServiceEventSource.Current.Message($"NC state is {ncConfig.State} instead of program." +
                        $" Aborting the operation");
                    return;
                }

                await networkRepositoryManager.UpdateNetworkContainerStateUntransacted(
                    ncConfig,
                    requestContext,
                    NetworkContainerState.Program,
                    NetworkContainerState.Program,
                    Status.Completed,
                    true,
                    tx);

                await RepositoryHelper.CommitTransactionAsync(tx);
            }
        }

        /// <summary>
        /// Process network container unpublish request.
        /// </summary>
        /// <param name="requestContext">Supply the request context.</param>
        /// <returns></returns>
        private async Task ProcessNetworkContainerUnPublish(
            RequestContext requestContext)
        {
            using (var tx = RepositoryHelper.GetTransaction())
            {
                NodeConfiguration nodeConfig = await networkRepositoryManager.GetNodeConfiguration(
                    requestContext.NodeID,
                    tx);
                if (nodeConfig == null)
                {
                    ServiceEventSource.Current.Message($"Failed to retrieve node configuration for NC: " +
                        $"{requestContext.RequestID}");
                    return;
                }

                SubnetConfiguration subnetConfig = await networkRepositoryManager.GetSubnetConfiguration(
                    SubnetConfiguration.GetSubnetConfigurationKey(requestContext.NetworkID, requestContext.SubnetName),
                    tx);
                if (subnetConfig == null)
                {
                    throw new DncNetworkContainerException(
                        requestContext.NetworkID,
                        NCConfiguration.GetNetworkContainerIDFromKey(requestContext.RequestID),
                        new ErrorInfo(ErrorCode.SubnetNotFound,
                        $"Subnet {requestContext.SubnetName} " +
                        $"associated with container not found",
                        ErrorType.NonRetriable));
                }

                NCConfiguration ncConfig = await networkRepositoryManager.GetNetworkContainerConfiguration(
                    requestContext.RequestID,
                    tx,
                    LockMode.Update);
                if (ncConfig == null)
                {
                    throw new DncManagementException(
                        new ErrorInfo(ErrorCode.NetworkContainerNotFound,
                            "Failed to unpublish NC due to failure to retrieve NC configuration",
                            ErrorType.NonRetriable));
                }

                // Abort the operation if the state of the network container is altered by another thread.
                if (ncConfig.State != NetworkContainerState.UnPublish)
                {
                    ServiceEventSource.Current.Message($"NC state is {ncConfig.State} instead of unpublish." +
                        $" Aborting the operation");
                    return;
                }

                string nodeInterfaceID =
                    string.IsNullOrWhiteSpace(ncConfig.Request.AssociationInfo.InterfaceID) ?
                        nodeConfig.NodeInfo.IPAddresses[0] :
                        ncConfig.Request.AssociationInfo.InterfaceID;
                string networkContainerID = NCConfiguration.GetNetworkContainerIDFromKey(requestContext.RequestID);
                await hostAgentClient.DeleteNetworkContainer(
                    networkContainerID,
                    nodeInterfaceID,
                    subnetConfig.AccessToken);

                await networkRepositoryManager.UpdateNetworkContainerStateUntransacted(
                    ncConfig,
                    requestContext,
                    NetworkContainerState.UnPublish,
                    NetworkContainerState.Revert,
                    Status.Pending,
                    false,
                    tx);

                await RepositoryHelper.CommitTransactionAsync(tx);
            }
        }

        /// <summary>
        /// Process network container publish request.
        /// </summary>
        /// <param name="requestContext">Supply the request context.</param>
        /// <returns></returns>
        private async Task ProcessNetworkContainerPublish(
            RequestContext requestContext)
        {
            NetworkContainerInfo networkContainerInfo =
                await networkRepositoryManager.GetNetworkContainerInfo(requestContext);
            if (networkContainerInfo == null)
            {
                throw new DncManagementException(
                    new ErrorInfo(ErrorCode.NetworkContainerNotFound,
                        "Failed to publish NC due to failure to retrieve NC configuration",
                        ErrorType.NonRetriable));
            }

            using (var tx = RepositoryHelper.GetTransaction())
            {
                NCConfiguration ncConfig = await networkRepositoryManager.GetNetworkContainerConfiguration(
                    requestContext.RequestID,
                    tx,
                    LockMode.Update);
                if (ncConfig == null)
                {
                    throw new DncManagementException(
                        new ErrorInfo(ErrorCode.NetworkContainerNotFound,
                            "Failed to publish NC due to failure to retrieve NC configuration",
                            ErrorType.NonRetriable));
                }

                // Abort the operation if the state of the network container is altered by another thread.
                if (ncConfig.State != NetworkContainerState.Publish)
                {
                    ServiceEventSource.Current.Message($"NC state is {ncConfig.State} instead of publish." +
                        $" Aborting the operation");
                    return;
                }

                await hostAgentClient.CreateNetworkContainer(networkContainerInfo);

                await networkRepositoryManager.UpdateNetworkContainerStateUntransacted(
                    ncConfig,
                    requestContext,
                    NetworkContainerState.Publish,
                    NetworkContainerState.AllocateLocalIP,
                    Status.Pending,
                    false,
                    tx);

                await RepositoryHelper.CommitTransactionAsync(tx);
            }
        }

        /// <summary>
        /// Process network container associate request.
        /// </summary>
        /// <param name="requestContext">Supply the request context.</param>
        /// <returns></returns>
        private async Task ProcessNetworkContainerAssociate(
            RequestContext requestContext)
        {
            NetworkContainerInfo networkContainerInfo =
                await networkRepositoryManager.GetNetworkContainerInfo(requestContext);
            if (networkContainerInfo == null)
            {
                throw new DncManagementException(
                    new ErrorInfo(ErrorCode.NetworkContainerNotFound,
                        "Failed to associate NC due to failure to retrieve NC configuration",
                        ErrorType.NonRetriable));
            }

            using (var tx = RepositoryHelper.GetTransaction())
            {
                NCConfiguration ncConfig = await networkRepositoryManager.GetNetworkContainerConfiguration(
                    requestContext.RequestID,
                    tx,
                    LockMode.Update);
                if (ncConfig == null)
                {
                    throw new DncManagementException(
                        new ErrorInfo(ErrorCode.NetworkContainerNotFound,
                            "Failed to associate NC due to failure to retrieve NC configuration",
                            ErrorType.NonRetriable));
                }

                // Abort the operation if the state of the network container is altered by another thread.
                if ((ncConfig.State != NetworkContainerState.Associate) &&
                    (ncConfig.State != NetworkContainerState.RefreshAssociation))
                {
                    ServiceEventSource.Current.Message($"NC state is {ncConfig.State} instead of associate or" +
                        $" refresh associate. Aborting the operation");
                    return;
                }

                cnsClient.ParseTagsIntoCnetAddresses(networkContainerInfo, ncConfig);

                await cnsClient.CreateNetworkContainer(networkContainerInfo);

                await networkRepositoryManager.UpdateNetworkContainerConfigUntransacted(
                    requestContext.RequestID,
                    ncConfig,
                    tx);

                await networkRepositoryManager.UpdateNetworkContainerStateUntransacted(
                    ncConfig,
                    requestContext,
                    networkContainerInfo.NetworkContainerState,
                    NetworkContainerState.Program,
                    Status.Pending,
                    false,
                    tx);

                await RepositoryHelper.CommitTransactionAsync(tx);
            }
        }

        /// <summary>
        /// Process network container allocate local IP request.
        /// </summary>
        /// <param name="requestContext">Supply the request context.</param>
        /// <returns></returns>
        private async Task ProcessNetworkContainerAllocateLocalIP(
            RequestContext requestContext)
        {
            using (var tx = RepositoryHelper.GetTransaction())
            {
                List<IPSubnet> allocatedLocalSubnets = new List<IPSubnet>();
                NCConfiguration ncConfig;
                try
                {
                    NodeConfiguration nodeConfiguration = await networkRepositoryManager.GetNodeConfiguration(
                        requestContext.NodeID,
                        tx,
                        LockMode.Update);
                    if (nodeConfiguration == null)
                    {
                        ServiceEventSource.Current.Message($"Failed to retrieve node configuration for NC: " +
                            $"{requestContext.RequestID}, during ncConfig.State");
                        return;
                    }

                    ncConfig = await networkRepositoryManager.GetNetworkContainerConfiguration(
                        requestContext.RequestID,
                        tx,
                        LockMode.Update);
                    if (ncConfig == null)
                    {
                        throw new DncManagementException(
                            new ErrorInfo(ErrorCode.NetworkContainerNotFound,
                                "Failed to allocate local IP for NC due to failure to retrieve NC configuration",
                                ErrorType.NonRetriable));
                    }

                    // Abort the operation if the state of the network container is altered by another thread.
                    if (ncConfig.State != NetworkContainerState.AllocateLocalIP)
                    {
                        ServiceEventSource.Current.Message($"NC state is {ncConfig.State} instead of AllocateLocalIP." +
                            $" Aborting the operation");
                        return;
                    }

                    if (nodeConfiguration.DeployedNCMap.Count != 0)
                    {
                        // Compile a list of allocated local addresses from the returned node configuration
                        foreach (KeyValuePair<string, IReadOnlyDictionary<string, string>> networkNCKvp
                            in nodeConfiguration.DeployedNCMap)
                        {
                            string networkID = networkNCKvp.Key;
                            foreach (string networkContainerID in networkNCKvp.Value.Keys)
                            {
                                string ncKey = NCConfiguration.GetNCKey(networkID, networkContainerID);
                                if (requestContext.RequestID != ncKey)
                                {
                                    NCConfiguration ncConfiguration =
                                        await networkRepositoryManager.GetNetworkContainerConfiguration(
                                            ncKey,
                                            tx);
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
                }
                catch (Exception exception)
                {
                    throw new DncManagementException(
                        new ErrorInfo(ErrorCode.AllocationFailed,
                            $"Failed to process ProcessNetworkContainerAllocateLocalIP due to exception {exception}",
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
                    await networkRepositoryManager.SetNetworkContainerLocalIPUntransacted(
                        ncConfig,
                        requestContext,
                        localIP.ToString(),
                        tx);
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

                await networkRepositoryManager.UpdateNetworkContainerStateUntransacted(
                    ncConfig,
                    requestContext,
                    NetworkContainerState.AllocateLocalIP,
                    NetworkContainerState.Associate,
                    Status.Pending,
                    false,
                    tx);

                await RepositoryHelper.CommitTransactionAsync(tx);
            }
        }

        /// <summary>
        /// Process reserve an IP for network container.
        /// </summary>
        /// <param name="requestContext">Supply the request context.</param>
        /// <returns></returns>
        private async Task ProcessNetworkContainerReserveIP(
            RequestContext requestContext)
        {
            using (var tx = RepositoryHelper.GetTransaction())
            {
                string nodeKey = NodeConfiguration.GetNodeKey(requestContext.NodeID);
                NodeConfiguration nodeConfig = await RepositoryHelper.GetFromReliableDictionary(
                    tx,
                    networkRepositoryManager.NodeConfigReliableCollection,
                    nodeKey,
                    LockMode.Update);
                if (nodeConfig == null)
                {
                    throw new DncNetworkContainerException(
                        requestContext.NetworkID,
                        requestContext.NodeID,
                        new ErrorInfo(ErrorCode.NodeNotFound, "Node not found", ErrorType.NonRetriable));
                }

                int exhaustedNCCapacity = nodeConfig.DeployedNCMap.Count;
                int nodeNCCapcity = await getNodeNCLimit(nodeConfig.NodeInfo.IPAddresses[0]);

                if(exhaustedNCCapacity >= nodeNCCapcity)
                {
                    throw new DncNetworkContainerException(
                        requestContext.NetworkID,
                        requestContext.NodeID,
                        new ErrorInfo(ErrorCode.NodeContainerCapacityExhausted,
                        $"Node NC Capacity {nodeNCCapcity} exhausted",
                        ErrorType.NonRetriable));
                }

                nodeConfig.AddNetworkContainer(requestContext.NetworkID,
                    NCConfiguration.GetNetworkContainerIDFromKey(requestContext.RequestID));

                await RepositoryHelper.CreateOrUpdateReliableDictionary(
                    tx,
                    networkRepositoryManager.NodeConfigReliableCollection,
                    nodeKey,
                    nodeConfig);

                string subnetKey = SubnetConfiguration.GetSubnetConfigurationKey(
                    requestContext.NetworkID,
                    requestContext.SubnetName);
                SubnetConfiguration subnetConfig = await networkRepositoryManager.GetSubnetConfiguration(
                    subnetKey,
                    tx,
                    LockMode.Update);
                if (subnetConfig == null)
                {
                    throw new DncManagementException(
                        new ErrorInfo(ErrorCode.SubnetNotFound,
                            "Failed to reserve IP for NC due to failure to find subnet configuration",
                            ErrorType.NonRetriable));
                }

                NCConfiguration ncConfig = await networkRepositoryManager.GetNetworkContainerConfiguration(
                    requestContext.RequestID,
                    tx,
                    LockMode.Update);
                if (ncConfig == null)
                {
                    throw new DncManagementException(
                        new ErrorInfo(ErrorCode.NetworkContainerNotFound,
                            "Failed to reserve IP for NC due to failure to retrieve NC configuration",
                            ErrorType.NonRetriable));
                }

                // Abort the operation if the state of the network container is altered by another thread.
                if (ncConfig.State != NetworkContainerState.Reserve)
                {
                    ServiceEventSource.Current.Message($"NC state is {ncConfig.State} instead of Reserve. " +
                        $"Aborting the operation");
                    return;
                }

                string reservationKey = ReservationConfiguration.GetReservationKey(
                    requestContext.NetworkID,
                    ncConfig.ReservationID);
                ReservationConfiguration reservationConfig = await networkRepositoryManager.GetReservationConfiguration(
                    reservationKey,
                    tx,
                    LockMode.Update);
                if (reservationConfig == null)
                {
                    throw new DncManagementException(
                        new ErrorInfo(ErrorCode.ReservationNotFound,
                            "Failed to reserve IP for NC due to failure to find reservation",
                            ErrorType.NonRetriable));
                }

                await ReserveIPUntransacted(
                    subnetKey,
                    subnetConfig,
                    reservationKey,
                    reservationConfig,
                    nodeConfig,
                    tx);

                ServiceEventSource.Current.Message($"Reserved IP: {reservationConfig.AllocatedSubnet} for ncKey {requestContext.RequestID}");

                await networkRepositoryManager.UpdateNetworkContainerStateUntransacted(
                    ncConfig,
                    requestContext,
                    NetworkContainerState.Reserve,
                    NetworkContainerState.Publish,
                    Status.Pending,
                    false,
                    tx);

                await RepositoryHelper.CommitTransactionAsync(tx);
            }
        }

        /// <summary>
        /// Returns if the shared subnet delegation is enabled
        /// </summary>
        /// <returns>True if enabled, false otherwise.</returns>
        private bool isSharedSubnetDelegationEnabled()
        {
            return azureStorageClient.IsSharedSubnetDelegationEnabled();
        }

        /// <summary>
        /// Returns number of containers allowed on a node.
        /// </summary>
        /// <returns> (number of logical cores of a node - 1) </returns>
        private async Task<int> getNodeNCLimit(string nodeInterfaceID)
        {
            try
            {
                int numCPUCores = await cnsClient.GetNumOfCPUCores(nodeInterfaceID);
                return (numCPUCores - 1);
            }
            catch(Exception ex)
            {
                ServiceEventSource.Current.Message(
                    $"Failed to get number of cpu cores on node {nodeInterfaceID} due to exception {ex}");

                throw ex;
            }
        }
    }
}
