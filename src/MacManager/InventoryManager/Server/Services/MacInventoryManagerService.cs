// -----------------------------------------------------------------------------
// <copyright file="MacInventoryManagerService.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.MacInventoryManager
{
    using System;
    using System.Collections.Generic;
    using System.Net;
    using System.Threading;
    using System.Threading.Tasks;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Azure.ContainerNetworking.MacInventoryManager.DataAccess;
    using Microsoft.Azure.ContainerNetworking.MacManager.Common;
    using Microsoft.Azure.Cosmos;
    using Microsoft.Extensions.Logging;

    public sealed class MacInventoryManagerService : IMacInventoryManagerService
    {
        private readonly ILogger<MacInventoryManagerService> logger;
        private readonly SemaphoreSlim rangeLock = new SemaphoreSlim(1);
        private readonly IMacAddressRangeComponent rangeComponent;
        private readonly IMacAddressComponent addressComponent;
        private readonly MacInventoryInitializationState initializationState;
        private readonly int macAddressStrLength = 12;

        public MacInventoryManagerService(
            ILogger<MacInventoryManagerService> logger,
            IMacAddressRangeComponent rangeComponent,
            IMacAddressComponent addressComponent,
            MacInventoryInitializationState initializationState)
        {
            this.logger = logger;
            this.rangeComponent = rangeComponent;
            this.addressComponent = addressComponent;
            this.initializationState = initializationState;
        }

        public async Task<MacAddressRange> UpdateRangeAsync(MacAddressRange range)
        {
            this.logger.CLogInformation($"Processing update of mac address range");
            try
            {
                this.EnsureInitialized();

                // validate input
                if (range == null)
                {
                    throw new ArgumentNullException(nameof(range));
                }

                range.Validate();

                this.logger.CLogInformation("MacAddress range validated successfully: {0}", range);

                await rangeLock.WaitAsync();

                try
                {
                    (long start, long end) = range.ConvertRanges();

                    // check if ranges overlap.
                    List<MacAddressRange> existingRanges = await GetRangesAsync();
                    existingRanges.Sort((MacAddressRange x, MacAddressRange y) =>
                    {
                        return string.Compare(x.RangeStart, y.RangeStart, StringComparison.OrdinalIgnoreCase);
                    });

                    MacAddressRange containingRange = null;
                    bool exactMatch = false;
                    for (int i = existingRanges.Count - 1; i >= 0; i--)
                    {
                        MacAddressRange existingRange = existingRanges[i];
                        (long existingStart, long existingEnd) = existingRange.ConvertRanges();

                        // operation needs to be idempotent, so we allow updates to existing ranges
                        if (start == existingStart && end == existingEnd)
                        {
                            containingRange = existingRange;
                            exactMatch = true;
                            break;
                        }

                        if (start >= existingStart && start <= existingEnd)
                        {
                            containingRange = existingRange;
                            break;
                        }

                        if (end >= existingStart && end <= existingEnd)
                        {
                            containingRange = existingRange;
                            break;
                        }
                    }

                    // if we are blocking, we must be under a containing range.
                    if (!exactMatch)
                    {
                        if (range.IsBlocked)
                        {
                            if (containingRange == null)
                            {
                                throw new InvalidOperationException($"Could not find containing range for {range}");
                            }
                            else if (containingRange.IsBlocked)
                            {
                                throw new InvalidOperationException($"{range} overlaps with existing range: {containingRange}");
                            }
                        }
                        else if (containingRange != null)
                        {
                            throw new InvalidOperationException($"{range} overlaps with existing range: {containingRange}");
                        }
                        else if (containingRange == null)
                        {
                            // check if we overlap a range.
                            for (int i = 0; i < existingRanges.Count; i++)
                            {
                                MacAddressRange existingRange = existingRanges[i];
                                (long existingStart, long existingEnd) = existingRange.ConvertRanges();

                                if (existingStart >= start && existingStart <= end)
                                {
                                    throw new InvalidOperationException($"{range} overlaps with existing range: {existingRange}");
                                }

                                if (existingEnd >= start && existingEnd <= end)
                                {
                                    throw new InvalidOperationException($"{range} overlaps with existing range: {existingRange}");
                                }
                            }
                        }
                    }

                    // add record.
                    await this.rangeComponent.UpsertRangeAsync(range);

                    // flatten.
                    List<string> macAddressesToProcess = new List<string>();

                    for (long i = start; i <= end; i++)
                    {
                        string address = i.ToString("X");

                        // Right in the beginning of this method, we validate that start and end MACs in MAC range are
                        // valid MAC addresses. Then these start/end MACs gets converted from string to long and it can lose its length
                        // if it has zeros in front, in this conversion. For example - if MAC is 00125A00B000 in string format, after conversion
                        // to long it will be 125A00B000. Add the missing octets here.
                        if (address.Length != macAddressStrLength)
                        {
                            int diff = macAddressStrLength - address.Length;
                            if (diff < 0)
                            {
                                this.logger.CLogInformation("Skip flattening MAC {0} as it is not of valid length {1}", address, macAddressStrLength);
                                continue;
                            }

                            string zeros = new String('0', diff);
                            address = string.Format("{0}{1}", zeros, address);
                        }

                        macAddressesToProcess.Add(address);
                    }

                    while (macAddressesToProcess.Count > 0)
                    {
                        Dictionary<string, MacAddress> existingItems = new Dictionary<string, MacAddress>(StringComparer.OrdinalIgnoreCase);
                        foreach (MacAddress address in await addressComponent.GetMacAddressesAsync(range.RangeStart, range.RangeEnd))
                        {
                            existingItems.Add(address.Address, address);
                        }

                        List<string> failures = new List<string>();
                        List<Task> tasks = new List<Task>();
                        foreach (string address in macAddressesToProcess)
                        {
                            bool update = false;
                            if (!existingItems.TryGetValue(address, out MacAddress macAddress))
                            {
                                macAddress = new MacAddress()
                                {
                                    Address = address,
                                    IsBlocked = range.IsBlocked
                                };
                                update = true;
                            }
                            else if (macAddress.IsBlocked != range.IsBlocked)
                            {
                                macAddress.IsBlocked = range.IsBlocked;
                                update = true;
                            }

                            if (update)
                            {
                                tasks.Add(UpsertMacAddressWithRetryAsync(macAddress, failures));
                            }
                        }

                        await Task.WhenAll(tasks);
                        macAddressesToProcess = failures;
                    }

                    // set flattened to true
                    range.IsFlattened = true;
                    await this.rangeComponent.UpsertRangeAsync(range);
                }
                finally
                {
                    rangeLock.Release();
                }

                this.logger.CLogInformation("MacAddress range updated successfully");

                return range;
            }
            catch (Exception ex)
            {
                this.logger.CLogError("Error updating range: {0}", ex);
                throw;
            }
        }

        private void EnsureInitialized()
        {
            if (!this.initializationState.IsInitialized)
            {
                throw new InvalidOperationException("MacInventoryInitializationService has not been initialized");
            }
        }

        private async Task UpsertMacAddressWithRetryAsync(MacAddress macAddress, List<string> failures)
        {
            try
            {
                await addressComponent.UpsertMacAddressAsync(macAddress);
            }
            catch (CosmosException ex) when (ex.StatusCode == HttpStatusCode.PreconditionFailed)
            {
                this.logger.CLogInformation($"Hit precondition failed when updating mac address {macAddress.Address}");
                failures.Add(macAddress.Address);
            }
        }

        public Task<List<MacAddressRange>> GetRangesAsync()
        {
            this.EnsureInitialized();
            return this.rangeComponent.GetRangesAsync();
        }

        public async Task<IEnumerable<MacAddressStats>> GetMacAddressStatsAsync()
        {
            this.EnsureInitialized();

            List<MacAddressRange> ranges = await rangeComponent.GetRangesAsync();
            List<MacAddressStats> stats = new List<MacAddressStats>(ranges.Count);

            foreach (MacAddressRange range in ranges)
            {
                MacAddressStats stat = new MacAddressStats();
                stat.BlockedReason = range.BlockedReason;
                stat.RangeStart = range.RangeStart;
                stat.RangeEnd = range.RangeEnd;

                (long start, long end) = range.ConvertRanges();
                stat.TotalCount = end - start + 1;
                stat.ReservedCount = await addressComponent.GetReservedCountAsync(range.RangeStart, range.RangeEnd);

                stats.Add(stat);
            }

            return stats;
        }
    }
}
