// -----------------------------------------------------------------------------
// <copyright file="IMacAllocatorService.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.MacAllocator
{
    using System;
    using System.Collections.Generic;
    using System.Configuration;
    using System.Linq;
    using System.Net;
    using System.Threading.Tasks;

    using Microsoft.Azure.ContainerNetworking.MacManager.Common;
    using Microsoft.Azure.Cosmos;
    using Microsoft.Extensions.Configuration;
    using Microsoft.Extensions.Logging;
    using Microsoft.Extensions.Options;

    public sealed class MacAllocatorService : IMacAllocatorService
    {
        private readonly ILogger<MacAllocatorService> logger;
        private readonly IMacAddressComponent addressComponent;
        private readonly MacAllocatorSetting macAllocatorSetting;

        public MacAllocatorService(
            ILogger<MacAllocatorService> logger,
            IMacAddressComponent addressComponent,
            IOptions<MacAllocatorSetting> macAllocatorSetting)
        {
            this.logger = logger;
            this.addressComponent = addressComponent;
            this.macAllocatorSetting = macAllocatorSetting.Value;
        }

        public async Task<MacAddress> AllocateMacAddressAsync(MacAllocatorRequest request)
        {
            this.logger.LogInformation($"Allocating MAC address for {request.AllocationId}");
            MacAddress address = await this.AllocateAsync(request, Convert.ToInt32(this.macAllocatorSetting.AllocationRetryCount));
            return address;
        }

        public async Task DeallocateMacAddressAsync(MacAllocatorRequest request)
        {
            this.logger.LogInformation($"Deallocating MAC address for {request.AllocationId}");

            List<MacAddress> addresses = await this.GetMacAddressesReservedById(request.AllocationId);
            List<Exception> exceptions = new List<Exception>();
            foreach (MacAddress a in addresses)
            {
                try
                {
                    a.ReservedBy = string.Empty;
                    a.ReservedDateTime = DateTime.MinValue;
                    await this.addressComponent.UpsertMacAddressAsync(a);
                }
                catch (Exception e)
                {
                    exceptions.Add(e);
                    logger.LogError($"Failed to deallocate MAC address {a.Address} for id {request.AllocationId}. Reason: {e.ToString()}");
                }
            }

            if (exceptions.Any())
            {
                string msg = $"Failed to deallocate {exceptions.Count} MAC addresses for Id {request.AllocationId}";
                logger.LogError(msg);
                throw new AggregateException(msg, exceptions);
            }
            else
            {
                logger.LogInformation($"Successfully deallocated {addresses.Count} MAC addresses for id {request.AllocationId}.");
            }
        }

        private async Task<MacAddress> AllocateAsync(MacAllocatorRequest request, int macCount)
        {
            if (request == null || string.IsNullOrEmpty(request.AllocationId))
            {
                throw new InvalidOperationException("Invalid MAC allocation request.");
            }

            // If MAC address is already reserved by the allocation ID then return that and dont allocate new MAC address.
            // Only 1 MAC address can be allocated for 1 allocation Id.
            List<MacAddress> reservedAddresses = await this.GetMacAddressesReservedById(request.AllocationId);
            if (reservedAddresses != null && reservedAddresses.Any())
            {
                if (reservedAddresses.Count > 1)
                {
                    throw new InvalidOperationException($"{reservedAddresses.Count} MAC addresses found for allocation id {request.AllocationId}. Possible MAC leak detected.");
                }

                return reservedAddresses.First();
            }

            // Allocate MAC if there is no MAC address allocated for the allocation Id.
            QueryDefinition query = new QueryDefinition($"SELECT TOP @macCount * FROM {MacManagerConstants.MacAddressContainerName} m WHERE m.ReservedBy IS NULL OR m.ReservedBy = ''")
                .WithParameter("@macCount", macCount);
            List<MacAddress> addresses = await this.addressComponent.GetMacAddressesAsync(query);
            if (addresses == null || !addresses.Any())
            {
                throw new InvalidOperationException($"No empty MAC address found to allocate for Id {request.AllocationId}. Please assign new MAC addresses.");
            }

            List<MacAddress> shuffledAddresses = addresses.OrderBy(a => Guid.NewGuid()).ToList();
            foreach (MacAddress address in shuffledAddresses)
            {
                try
                {
                    if (!string.IsNullOrEmpty(address.ReservedBy) || !string.IsNullOrWhiteSpace(address.ReservedBy))
                    {
                        throw new InvalidOperationException($"MAC address {address.Address} is already reserved by {address.ReservedBy}");
                    }
                    
                    address.ReservedBy = request.AllocationId;
                    address.ReservedDateTime = DateTime.UtcNow;
                    await this.addressComponent.UpsertMacAddressAsync(address);

                    this.logger.LogInformation($"Successfully reserved MAC address {address.Address} for Allocation Id {request.AllocationId}");
                    return address;
                }
                catch (Exception e)
                {
                    this.logger.LogError($"Failed to reserve MAC address {address.Address} for id {request.AllocationId}. Trying to reserve another MAC address. Reason: {e.ToString()}");
                }
            }

            string msg = $"Failed to reserve MAC address after trying {shuffledAddresses.Count} times for id {request.AllocationId}.";
            this.logger.LogError(msg);
            throw new InvalidOperationException(msg);
        }

        private async Task<List<MacAddress>> GetMacAddressesReservedById(string allocationId)
        {
            QueryDefinition query = new QueryDefinition($"SELECT * FROM {MacManagerConstants.MacAddressContainerName} m WHERE m.ReservedBy = @macReservedBy")
                .WithParameter("@macReservedBy", allocationId);
            return await this.addressComponent.GetMacAddressesAsync(query);
        }
    }
}
