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
        private readonly CosmosClientProvider clientProvider;
        private Container container;

        public MacAddressRangeComponent(CosmosClientProvider clientProvider)
        {
            this.clientProvider = clientProvider;
        }

        private Container GetContainer()
        {
            if (this.container == null)
            {
                this.container = clientProvider.CosmosClient.GetContainer(MacManagerConstants.DatabaseName, MacManagerConstants.MacAddressRangesContainerName);
            }

            return container;
        }

        public async Task UpsertRangeAsync(MacAddressRange range)
        {
            range.LastUpdated = DateTime.UtcNow;
            await this.GetContainer().UpsertItemAsync(range);
        }

        public async Task<List<MacAddressRange>> GetRangesAsync()
        {
            List<MacAddressRange> ranges = new List<MacAddressRange>();

            using (FeedIterator<MacAddressRange> iterator = this.GetContainer().GetItemQueryIterator<MacAddressRange>())
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
