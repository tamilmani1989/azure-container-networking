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
    using System.Linq;
    using System.Text;
    using System.Threading;
    using System.Threading.Tasks;

    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Azure.ContainerNetworking.MacManager.Common;
    using Microsoft.Azure.Cosmos;
    using Microsoft.Extensions.Logging;
    using Microsoft.Extensions.Options;

    public sealed class MacAllocatorService : IMacAllocatorService
    {
        private const string ServiceUpMetric = "MacAllocatorServiceUp";

        private readonly ILogger<MacAllocatorService> logger;
        private readonly IMacAddressComponent addressComponent;
        private readonly MacAllocatorSetting macAllocatorSetting;
        private readonly IMetricsProvider metricsProvider;
        private readonly Timer timer;

        public MacAllocatorService(
            ILogger<MacAllocatorService> logger,
            IMacAddressComponent addressComponent,
            IOptions<MacAllocatorSetting> macAllocatorSetting,
            IMetricsProvider metricsProvider)
        {
            this.logger = logger;
            this.addressComponent = addressComponent;
            this.macAllocatorSetting = macAllocatorSetting.Value;
            this.metricsProvider = metricsProvider;

            this.timer = new Timer(this.SendServiceUpMetric, null, TimeSpan.Zero, TimeSpan.FromSeconds(60));
        }

        public async Task<MacAllocatorResponse> AllocateMacAddressAsync(MacAllocatorRequest request)
        {
            this.logger.CLogInformation($"Allocating MAC address for {request.AllocationId}");
            return await this.AllocateAsync(request, Convert.ToInt32(this.macAllocatorSetting.AllocationRetryCount));
        }

        public async Task<MacAllocatorResponse> DeallocateMacAddressAsync(MacAllocatorRequest request)
        {
            this.logger.CLogInformation($"Deallocating MAC address for {request.AllocationId}");

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
                    logger.CLogError($"Failed to deallocate MAC address {a.Address} for id {request.AllocationId}. Reason: {e.ToString()}");
                }
            }

            if (exceptions.Any())
            {
                string msg = $"Failed to deallocate {exceptions.Count} MAC addresses for Id {request.AllocationId}";
                logger.CLogError(msg);

                StringBuilder messageBuilder = new StringBuilder();
                messageBuilder.AppendLine(msg);

                for (int i=0; i < exceptions.Count; ++i)
                {
                    messageBuilder.AppendLine($"Exception {i}: {exceptions[i].Message}");
                }

                return new MacAllocatorResponse(
                    allocationId: request.AllocationId,
                    errorMessage: messageBuilder.ToString(),
                    shouldRetry: true);
            }
            else
            {
                logger.CLogInformation($"Successfully deallocated {addresses.Count} MAC addresses for id {request.AllocationId}.");
                return new MacAllocatorResponse(
                    allocationId: request.AllocationId,
                    macAddress: null);
            }
        }

        private async Task<MacAllocatorResponse> AllocateAsync(MacAllocatorRequest request, int macCount)
        {
            if (request == null || string.IsNullOrEmpty(request.AllocationId))
            {
                return new MacAllocatorResponse(
                    allocationId: request?.AllocationId,
                    errorMessage: "Invalid MAC allocation request.",
                    shouldRetry: false);
            }

            // If MAC address is already reserved by the allocation ID then return that and dont allocate new MAC address.
            // Only 1 MAC address can be allocated for 1 allocation Id.
            List<MacAddress> reservedAddresses = await this.GetMacAddressesReservedById(request.AllocationId);
            if (reservedAddresses != null && reservedAddresses.Any())
            {
                if (reservedAddresses.Count > 1)
                {
                    return new MacAllocatorResponse(
                        allocationId: request.AllocationId,
                        errorMessage: $"{reservedAddresses.Count} MAC addresses found for allocation id {request.AllocationId}. Possible MAC leak detected.",
                        shouldRetry: false);
                }

                return new MacAllocatorResponse(
                    allocationId: request.AllocationId,
                    macAddress: reservedAddresses.First().Address);
            }

            // Allocate MAC if there is no MAC address allocated for the allocation Id.
            QueryDefinition query = new QueryDefinition($"SELECT TOP @macCount * FROM {MacManagerConstants.MacAddressContainerName} m WHERE m.ReservedBy = null OR m.ReservedBy = ''")
                .WithParameter("@macCount", macCount);
            List<MacAddress> addresses = await this.addressComponent.GetMacAddressesAsync(query);
            if (addresses == null || !addresses.Any())
            {
                // shouldRetry is true because retrying later might free up a mac address.
                return new MacAllocatorResponse(
                    allocationId: request.AllocationId,
                    errorMessage: $"No empty MAC address found to allocate for Id {request.AllocationId}. Please assign new MAC addresses.",
                    shouldRetry: true);
            }

            List<MacAddress> shuffledAddresses = addresses.OrderBy(a => Guid.NewGuid()).ToList();
            foreach (MacAddress address in shuffledAddresses)
            {
                try
                {
                    if (!string.IsNullOrEmpty(address.ReservedBy) || !string.IsNullOrWhiteSpace(address.ReservedBy))
                    {
                        return new MacAllocatorResponse(
                            allocationId: request.AllocationId,
                            errorMessage: $"MAC address {address.Address} is already reserved by {address.ReservedBy}",
                            shouldRetry: true);
                    }
                    
                    address.ReservedBy = request.AllocationId;
                    address.ReservedDateTime = DateTime.UtcNow;
                    await this.addressComponent.UpsertMacAddressAsync(address);

                    this.logger.CLogInformation($"Successfully reserved MAC address {address.Address} for Allocation Id {request.AllocationId}");
                    return new MacAllocatorResponse(
                        allocationId: request.AllocationId,
                        macAddress: address.Address);
                }
                catch (Exception e)
                {
                    this.logger.CLogError($"Failed to reserve MAC address {address.Address} for id {request.AllocationId}. Trying to reserve another MAC address. Reason: {e.ToString()}");
                }
            }

            string msg = $"Failed to reserve MAC address after trying {shuffledAddresses.Count} times for id {request.AllocationId}.";
            this.logger.CLogError(msg);

            return new MacAllocatorResponse(
                allocationId: request.AllocationId,
                errorMessage: msg,
                shouldRetry: true);
        }

        private async Task<List<MacAddress>> GetMacAddressesReservedById(string allocationId)
        {
            QueryDefinition query = new QueryDefinition($"SELECT * FROM {MacManagerConstants.MacAddressContainerName} m WHERE m.ReservedBy = @macReservedBy")
                .WithParameter("@macReservedBy", allocationId);
            return await this.addressComponent.GetMacAddressesAsync(query);
        }

        private void SendServiceUpMetric(object state)
        {
            try
            {
                this.metricsProvider.EmitMetric(ServiceUpMetric, 1);
            }
            catch (Exception ex)
            {
                this.logger.CLogError("Caught error during sending metric: {0}", ex);
            }
        }
    }
}
