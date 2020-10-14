// --------------------------------------------------------------------------------------------------------------------
// <copyright file="TableStoreInitHostedService.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Threading;
    using System.Threading.Tasks;
    using Microsoft.Extensions.DependencyInjection;
    using Microsoft.Extensions.Hosting;

    public class TableStoreInitHostedService : IHostedService
    {
        private readonly IServiceProvider serviceProvider;

        public TableStoreInitHostedService(IServiceProvider serviceProvider)
        {
            this.serviceProvider = serviceProvider;
        }

        public async Task StartAsync(CancellationToken cancellationToken)
        {
            Console.WriteLine("TableStoreInitHostedService started");

            using (var scope = serviceProvider.CreateScope())
            {
                // Init cosmosDb store
                ITableDataStore cosmosStore = scope.ServiceProvider.GetRequiredService<ITableDataStore>();
                Console.WriteLine("TableStoreInitHostedService: Init ITableDataStore");
                await cosmosStore.InitAsync();

                // Init ITableOperations
                Console.WriteLine("TableStoreInitHostedService: Init ITableOperations");
                ITableOperations tableOperations = scope.ServiceProvider.GetRequiredService<ITableOperations>();
            }

            Console.WriteLine("TableStoreInitHostedService done");
        }

        // no-op for now
        public Task StopAsync(CancellationToken cancellationToken) => Task.CompletedTask;
    }
}
