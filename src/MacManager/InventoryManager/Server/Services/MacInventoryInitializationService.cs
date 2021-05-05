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
    using Microsoft.Azure.ContainerNetworking.MacInventoryManager.DataAccess;
    using Microsoft.Azure.ContainerNetworking.MacManager.Common;
    using Microsoft.Azure.Cosmos;
    using Microsoft.Extensions.Hosting;
    using Microsoft.Extensions.Logging;

    internal sealed class MacInventoryInitializationService : IHostedService
    {
        private static readonly TimeSpan delay1Minute = TimeSpan.FromSeconds(60);
        private static readonly TimeSpan delay5Minute = TimeSpan.FromSeconds(300);
        private static readonly TimeSpan delay1Hour = TimeSpan.FromSeconds(3600);
        private const string rangeStart = "";
        private const string rangeEnd = "FFFFFFFFFFFF";
        private const int macAddressContainerThoughput = 5000; // RUs
        private readonly CosmosClientProvider clientProvider;
        private CancellationTokenSource stopTokenSource;
        private readonly CancellationToken stopToken;
        private readonly ILogger<MacInventoryInitializationService> logger;
        private readonly MacInventoryInitializationState initializationState;
        private readonly IMetricsProvider metricsProvider;
        private readonly IMacAddressComponent addressComponent;

        public MacInventoryInitializationService(
            CosmosClientProvider clientProvider,
            ILogger<MacInventoryInitializationService> logger,
            MacInventoryInitializationState initializationState,
            IMetricsProvider metricsProvider,
            IMacAddressComponent addressComponent)
        {
            this.clientProvider = clientProvider;
            this.stopTokenSource = new CancellationTokenSource();
            this.stopToken = this.stopTokenSource.Token;
            this.logger = logger;
            this.initializationState = initializationState;
            this.metricsProvider = metricsProvider;
            this.addressComponent = addressComponent;
        }

        public Task StartAsync(CancellationToken cancellationToken)
        {
            Task initializeTask = Task.Run(InitializeAsync);
            Task serviceUpMetricTask = Task.Run(SendServiceUpMetricAsync);
            Task sendMetricTask = Task.Run(SendMacMetricsAsync);
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

                    await database.CreateContainerIfNotExistsAsync(
                        addressContainer, throughput: macAddressContainerThoughput, cancellationToken: stopToken);

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
                    await Task.Delay(delay1Minute, this.stopToken);
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
                    await Task.Delay(delay1Minute, this.stopToken);
                }
                catch (TaskCanceledException)
                {
                    this.logger.CLogInformation($"{nameof(SendServiceUpMetricAsync)} task was stopped");
                    break;
                }
                catch (Exception ex)
                {
                    this.logger.CLogError("Caught error during sending metric: {0}", ex);
                    await Task.Delay(delay1Minute, this.stopToken);
                }
            }
        }

        private async Task SendMacMetricsAsync()
        {
            while (!this.stopToken.IsCancellationRequested)
            {
                try
                {
                    if (!this.initializationState.IsInitialized)
                    {
                        this.logger.CLogInformation($"{nameof(SendMacMetricsAsync)}: Initialization not complete, will retry in a bit");
                        await Task.Delay(delay1Minute, this.stopToken);
                        continue;
                    }

                    // it is cheaper to do 2 queries on CosmosDB rather than doing a group by
                    // for 65K addresses it was 1200 RUs vs 6 RUs
                    long totalAddresses = await this.addressComponent.GetCountAsync(rangeStart, rangeEnd);
                    long reservedAddresses = await this.addressComponent.GetReservedCountAsync(rangeStart, rangeEnd);

                    this.logger.CLogInformation("Sending metrics - totalAddresses: {0}, reservedAddresses: {1}", totalAddresses, reservedAddresses);

                    this.metricsProvider.EmitMetric(Constants.TotalAddresses, totalAddresses);
                    this.metricsProvider.EmitMetric(Constants.ReservedAddresses, reservedAddresses);
                    await Task.Delay(delay1Hour, this.stopToken);
                }
                catch (TaskCanceledException)
                {
                    this.logger.CLogInformation($"{nameof(SendMacMetricsAsync)} task was stopped");
                    break;
                }
                catch (Exception ex)
                {
                    this.logger.CLogError("Caught error during sending mac metrics: {0}", ex);
                    await Task.Delay(delay5Minute, this.stopToken);
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
