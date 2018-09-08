//-----------------------------------------------------------------------
// <copyright file="RequestWatcher.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// 
//-----------------------------------------------------------------------

using Cns;
using DNCService;
using Microsoft.ServiceFabric.Data.Notifications;
using NetworkManager.DataModel;
using NMAgent;
using NMAgent.Data;
using System;
using System.Collections.Generic;
using System.Threading.Tasks;

namespace NetworkManager
{
    /// <summary>
    /// Class definiton for request watcher.
    /// Monitors and processes pending requests.
    /// </summary>
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
        /// Constructor.
        /// </summary>
        /// <param name="hostAgentClient">Handle to the host agent client.</param>
        /// <param name="cnsClient">Handle to the cns client.</param>
        public RequestWatcher(HostAgentClient hostAgentClient, CnsClient cnsClient)
        {
            this.hostAgentClient = hostAgentClient;
            this.cnsClient = cnsClient;
            syncLock = new object();
        }

        /// <summary>
        /// Initialize.
        /// </summary>
        /// <param name="networkManager">Supply handle to the network repository manager.</param>
        public void Init(NetworkRepositoryManager networkManager)
        {
            this.networkRepositoryManager = networkManager;
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
                }
                catch (Exception exception)
                {
                    ServiceEventSource.Current.Message("Failed to process Network request {0}. Exception:{1}",
                        requestContext != null ? requestContext.ToString(): "null",
                        exception);

                    await networkRepositoryManager.MoveRequestToFailedQueue(
                        requestID,
                        hostAgentClient.DncConfiguration.DNCSettings.RequestRetryBaseInterval,
                        exception);
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

            ServiceEventSource.Current.Message($"###########NCState:{ncConfiguration.State}");

            switch (ncConfiguration.State)
            {
                case NetworkContainerState.Associate:
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

                    //case NetworkContainerState.RefreshAssociation:
                    //    ProcessNetworkContainerRefreshAssociation(requestContext, ncConfiguration);
                    //    break;
            }
        }

        private void ProcessNetworkContainerRefreshAssociation(RequestContext requestContext, NCConfiguration ncConfiguration)
        {
            throw new NotImplementedException();
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
                await networkRepositoryManager.DeleteRequestContext(requestContext.RequestID);
                return;
            }

            if (networkContainerInfo.NetworkContainerState != NetworkContainerState.DisAssociate)
            {
                return;
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
            NetworkContainerInfo networkContainerInfo = await networkRepositoryManager.GetNetworkContainerInfo(
                requestContext.RequestID);
            if (networkContainerInfo == null)
            {
                await networkRepositoryManager.DeleteRequestContext(requestContext.RequestID);
                return;
            }
            if (networkContainerInfo.NetworkContainerState != NetworkContainerState.Revert)
            {
                return;
            }

            await networkRepositoryManager.DeleteNetworkContainer(
                networkContainerInfo,
                true);
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
                await networkRepositoryManager.DeleteRequestContext(requestContext.RequestID);
                return;
            }
            if (networkContainerInfo.NetworkContainerState != NetworkContainerState.Program)
            {
                return;
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
                await networkRepositoryManager.DeleteRequestContext(requestContext.RequestID);
                return;
            }
            if (networkContainerInfo.NetworkContainerState != NetworkContainerState.UnPublish)
            {
                return;
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
                await networkRepositoryManager.DeleteRequestContext(requestContext.RequestID);
                return;
            }
            if (networkContainerInfo.NetworkContainerState != NetworkContainerState.Publish)
            {
                return;
            }

            await hostAgentClient.CreateNetworkContainer(networkContainerInfo);
            await networkRepositoryManager.UpdateNetworkContainerState(
                requestContext,
                NetworkContainerState.Publish,
                NetworkContainerState.Associate,
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
            NetworkContainerInfo networkContainerInfo = await networkRepositoryManager.GetNetworkContainerInfo(
                requestContext.RequestID);
            if (networkContainerInfo == null)
            {
                await networkRepositoryManager.DeleteRequestContext(requestContext.RequestID);
                return;
            }

            if (networkContainerInfo.NetworkContainerState != NetworkContainerState.Associate)
            {
                return;
            }

            await cnsClient.CreateNetworkContainer(networkContainerInfo);
            await networkRepositoryManager.UpdateNetworkContainerState(
                requestContext,
                NetworkContainerState.Associate,
                NetworkContainerState.Program,
                Status.Pending,
                false);
        }
    }
}
