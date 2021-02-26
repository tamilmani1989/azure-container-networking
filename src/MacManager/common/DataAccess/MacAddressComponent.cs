// -----------------------------------------------------------------------------
// <copyright file="MacAddressComponent.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.MacManager.Common
{
    using System.Collections.Generic;
    using System.Linq;
    using System.Threading;
    using System.Threading.Tasks;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Azure.Cosmos;
    
    internal sealed class MacAddressComponent : IMacAddressComponent
    {
        private readonly CosmosClientProvider clientProvider;
        private Container container;

        public MacAddressComponent(CosmosClientProvider clientProvider)
        {
            this.clientProvider = clientProvider;
        }

        private async Task<Container> GetContainerAsync()
        {
            if (this.container == null)
            {
                await clientProvider.InitializeAsync(CancellationToken.None);
                this.container = clientProvider.CosmosClient.GetContainer(MacManagerConstants.DatabaseName, MacManagerConstants.MacAddressContainerName);
            }
            return this.container;
        }

        public async Task<List<MacAddress>> GetMacAddressesAsync(string rangeStart, string rangeEnd)
        {
            QueryDefinition query = new QueryDefinition($"SELECT * FROM {MacManagerConstants.MacAddressContainerName} m WHERE m.id >= @rangeStart AND m.id <= @rangeEnd")
                .WithParameter("@rangeStart", rangeStart)
                .WithParameter("@rangeEnd", rangeEnd);

            List<MacAddress> addresses = new List<MacAddress>();
            using (FeedIterator<MacAddress> iterator = (await this.GetContainerAsync()).GetItemQueryIterator<MacAddress>(query))
            {
                while (iterator.HasMoreResults)
                {
                    addresses.AddRange(await iterator.ReadNextAsync());
                }
            }

            return addresses;
        }

        public async Task<List<MacAddress>> GetMacAddressesAsync(QueryDefinition query)
        {
            List<MacAddress> addresses = new List<MacAddress>();
            using (FeedIterator<MacAddress> iterator = (await this.GetContainerAsync()).GetItemQueryIterator<MacAddress>(query))
            {
                while (iterator.HasMoreResults)
                {
                    addresses.AddRange(await iterator.ReadNextAsync());
                }
            }

            return addresses;
        }

        public async Task<long> GetReservedCountAsync(string rangeStart, string rangeEnd)
        {
            QueryDefinition query = new QueryDefinition($"SELECT VALUE COUNT(1) FROM {MacManagerConstants.MacAddressContainerName} m WHERE m.id >= @rangeStart AND m.id <= @rangeEnd AND m.ReservedBy <> null")
            .WithParameter("@rangeStart", rangeStart)
            .WithParameter("@rangeEnd", rangeEnd);

            List<MacAddress> addresses = new List<MacAddress>();
            using (FeedIterator<long> iterator = (await this.GetContainerAsync()).GetItemQueryIterator<long>(query))
            {
                while (iterator.HasMoreResults)
                {
                    return (await iterator.ReadNextAsync()).First();
                }
            }

            return 0;
        }

        public async Task<long> GetCountAsync(string rangeStart, string rangeEnd)
        {
            QueryDefinition query = new QueryDefinition($"SELECT VALUE COUNT(1) FROM {MacManagerConstants.MacAddressContainerName} m WHERE m.id >= @rangeStart AND m.id <= @rangeEnd")
            .WithParameter("@rangeStart", rangeStart)
            .WithParameter("@rangeEnd", rangeEnd);

            List<MacAddress> addresses = new List<MacAddress>();
            using (FeedIterator<long> iterator = (await this.GetContainerAsync()).GetItemQueryIterator<long>(query))
            {
                while (iterator.HasMoreResults)
                {
                    return (await iterator.ReadNextAsync()).First();
                }
            }

            return 0;
        }

        public async Task UpsertMacAddressAsync(MacAddress address)
        {
            await (await this.GetContainerAsync()).UpsertItemAsync(address, requestOptions: new ItemRequestOptions { IfMatchEtag = address.ETag });
        }
    }
}
