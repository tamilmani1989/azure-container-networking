// -----------------------------------------------------------------------------
// <copyright file="MacAddressRangeComponent.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.MacInventoryManager.DataAccess
{
    using System;
    using System.Collections.Generic;
    using System.Threading.Tasks;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Azure.ContainerNetworking.MacManager.Common;
    using Microsoft.Azure.Cosmos;

    internal sealed class MacAddressRangeComponent : IMacAddressRangeComponent
    {
        private readonly Container container;

        public MacAddressRangeComponent(CosmosClientProvider clientProvider)
        {
            this.container = clientProvider.CosmosClient.GetContainer(MacManagerConstants.DatabaseName, MacManagerConstants.MacAddressRangesContainerName);
        }

        public async Task UpsertRangeAsync(MacAddressRange range)
        {
            range.LastUpdated = DateTime.UtcNow;
            await this.container.UpsertItemAsync(range);
        }

        public async Task<List<MacAddressRange>> GetRangesAsync()
        {
            List<MacAddressRange> ranges = new List<MacAddressRange>();

            using (FeedIterator<MacAddressRange> iterator = this.container.GetItemQueryIterator<MacAddressRange>())
            {
                while (iterator.HasMoreResults)
                {
                    ranges.AddRange((await iterator.ReadNextAsync()));
                }
            }

            return ranges;
        }
    }
}
