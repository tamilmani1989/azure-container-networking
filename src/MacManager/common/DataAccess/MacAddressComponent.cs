// -----------------------------------------------------------------------------
// <copyright file="MacAddressComponent.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.MacManager.Common
{
    using System.Collections.Generic;
    using System.Threading.Tasks;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Azure.Cosmos;
    
    internal sealed class MacAddressComponent : IMacAddressComponent
    {
        private readonly Container container;

        public MacAddressComponent(CosmosClientProvider clientProvider)
        {
            this.container = clientProvider.CosmosClient.GetContainer(MacManagerConstants.DatabaseName, MacManagerConstants.MacAddressContainerName);
        }

        public async Task<List<MacAddress>> GetMacAddressesAsync(string rangeStart, string rangeEnd)
        {
            QueryDefinition query = new QueryDefinition($"SELECT * FROM {MacManagerConstants.MacAddressContainerName} m WHERE m.id >= @rangeStart AND m.id <= @rangeEnd")
                .WithParameter("@rangeStart", rangeStart)
                .WithParameter("@rangeEnd", rangeEnd);

            List<MacAddress> addresses = new List<MacAddress>();
            using (FeedIterator<MacAddress> iterator = this.container.GetItemQueryIterator<MacAddress>(query))
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
            using (FeedIterator<MacAddress> iterator = this.container.GetItemQueryIterator<MacAddress>(query))
            {
                while (iterator.HasMoreResults)
                {
                    addresses.AddRange(await iterator.ReadNextAsync());
                }
            }

            return addresses;
        }

        public Task UpsertMacAddressAsync(MacAddress address)
        {
            return this.container.UpsertItemAsync(address, requestOptions: new ItemRequestOptions { IfMatchEtag = address.ETag });
        }
    }
}
