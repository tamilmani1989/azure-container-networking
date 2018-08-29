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
    /// Implements the request reset monitor.
    /// Monitors the failed requests and adds them back
    /// for processing after the back off time.
    /// </summary>
    public class RequestResetMonitor
    {
        /// <summary>
        /// Cached cancellation token.
        /// </summary>
        private CancellationToken cancellationToken;

        /// <summary>
        /// Handle to the network manager.
        /// </summary>
        private NetworkRepositoryManager networkRepsoitoryManager;

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="networkManager">Supply the network manager.</param>
        /// <param name="cancellationToken">Supply the cancellation token.</param>
        public RequestResetMonitor(
            NetworkRepositoryManager networkManager,
            CancellationToken cancellationToken)
        {
            this.cancellationToken = cancellationToken;
            this.networkRepsoitoryManager = networkManager;
        }

        /// <summary>
        /// Start monitor.
        /// </summary>
        public async void StartAsync()
        {
            while (true)
            {
                ICollection<string> requestIDs = await networkRepsoitoryManager.GetAllRequestIDs(false);
                foreach (string requestID in requestIDs)
                {
                    IReliableDictionary<string, RequestContext> pendingRequestContextReliableCollection =
                        await RepositoryHelper.GetReliableDictionaryAsync<string, RequestContext>(
                            Constants.PendingRequestsTableName);
                    IReliableDictionary<string, RequestContext> failedRequestContextReliableCollection =
                        await RepositoryHelper.GetReliableDictionaryAsync<string, RequestContext>(
                            Constants.FailedRequestsTableName);
                    cancellationToken.ThrowIfCancellationRequested();
                    using (var tx = RepositoryHelper.GetTransaction())
                    {
                        RequestContext requestContext = await RepositoryHelper.GetFromReliableDictionary(
                            tx,
                            failedRequestContextReliableCollection,
                            requestID,
                            LockMode.Update);

                        cancellationToken.ThrowIfCancellationRequested();
                        if (requestContext == null)
                        {
                            continue;
                        }

                        if (requestContext.LastRetriedDateTime.Add(requestContext.RetryInterval) < DateTime.UtcNow)
                        {
                            ServiceEventSource.Current.Message(
                                $"Moving request context from failed to pending queue:{requestContext}");
                            await RepositoryHelper.CreateOrUpdateReliableDictionary(
                                tx,
                                pendingRequestContextReliableCollection,
                                requestID,
                                requestContext);
                            cancellationToken.ThrowIfCancellationRequested();
                            await RepositoryHelper.RemoveFromReliableDictionary(
                                tx,
                                failedRequestContextReliableCollection,
                                requestID);
                            cancellationToken.ThrowIfCancellationRequested();
                            await RepositoryHelper.CommitTransactionAsync(tx);
                            cancellationToken.ThrowIfCancellationRequested();
                        }
                    }
                }
                await Task.Delay(TimeSpan.FromSeconds(1), cancellationToken);
            }
        }
    }
}
