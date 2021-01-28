// -----------------------------------------------------------------------------
// <copyright file="MacInventoryInitializationService.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.MacInventoryManager
{
    using System;
    using System.Threading;
    using System.Threading.Tasks;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Azure.ContainerNetworking.MacManager.Common;
    using Microsoft.Azure.Cosmos;
    using Microsoft.Extensions.Hosting;
    using Microsoft.Extensions.Logging;

    internal sealed class MacInventoryInitializationService : IHostedService
    {
        private readonly CosmosClientProvider clientProvider;
        private CancellationTokenSource stopTokenSource;
        private readonly CancellationToken stopToken;
        private readonly ILogger<MacInventoryManagerService> logger;
        private readonly MacInventoryInitializationState initializationState;
        private readonly IMetricsProvider metricsProvider;

        public MacInventoryInitializationService(
            CosmosClientProvider clientProvider,
            ILogger<MacInventoryManagerService> logger,
            MacInventoryInitializationState initializationState,
            IMetricsProvider metricsProvider)
        {
            this.clientProvider = clientProvider;
            this.stopTokenSource = new CancellationTokenSource();
            this.stopToken = this.stopTokenSource.Token;
            this.logger = logger;
            this.initializationState = initializationState;
            this.metricsProvider = metricsProvider;
        }

        public Task StartAsync(CancellationToken cancellationToken)
        {
            Task initializeTask = Task.Run(InitializeAsync);
            Task serviceUpMetricTask = Task.Run(SendServiceUpMetricAsync);
            return Task.CompletedTask;
        }

        private async Task InitializeAsync()
        {
            while (!this.initializationState.IsInitialized && !this.stopToken.IsCancellationRequested)
            {
                try
                {
                    this.logger.CLogInformation("Initializing client provider");
                    await this.clientProvider.InitializeAsync(stopToken);

                    this.logger.CLogInformation("Creating database if not exists {0}", MacManagerConstants.DatabaseName);
                    Database database = await clientProvider.CosmosClient.CreateDatabaseIfNotExistsAsync(MacManagerConstants.DatabaseName);

                    this.logger.CLogInformation("Creating address container if not exists");
                    ContainerProperties addressContainer = new ContainerProperties(
                        MacManagerConstants.MacAddressContainerName,
                        MacManagerConstants.MacAddressContainerPartitionKeyPath);

                    await database.CreateContainerIfNotExistsAsync(addressContainer, cancellationToken: stopToken);

                    this.logger.CLogInformation("Creating range container if not exists");
                    ContainerProperties rangeContainer = new ContainerProperties(
                        MacManagerConstants.MacAddressRangesContainerName,
                        MacManagerConstants.MacAddressRangePartitionKeyPath);

                    await database.CreateContainerIfNotExistsAsync(rangeContainer, cancellationToken: stopToken);
                    this.initializationState.IsInitialized = true;
                }
                catch (TaskCanceledException)
                {
                    this.logger.CLogInformation("Initialization task was stopped");
                    break;
                }
                catch (Exception ex)
                {
                    this.logger.CLogError("Caught error during initialization: {0}", ex);
                    await Task.Delay(5000);
                }
            }
        }

        private async Task SendServiceUpMetricAsync()
        {
            while (!this.stopToken.IsCancellationRequested)
            {
                try
                {
                    this.metricsProvider.EmitMetric(Constants.ServiceUpMetric, 1);
                    await Task.Delay(TimeSpan.FromSeconds(60), this.stopToken);
                }
                catch (TaskCanceledException)
                {
                    this.logger.CLogInformation($"{nameof(SendServiceUpMetricAsync)} task was stopped");
                    break;
                }
                catch (Exception ex)
                {
                    this.logger.CLogError("Caught error during sending metric: {0}", ex);
                    await Task.Delay(TimeSpan.FromSeconds(60), this.stopToken);
                }
            }
        }

        public Task StopAsync(CancellationToken cancellationToken)
        {
            if (this.stopTokenSource != null)
            {
                this.stopTokenSource.Cancel();
                this.stopTokenSource.Dispose();
                this.stopTokenSource = null;
            }
            return Task.CompletedTask;
        }
    }
}
