// -----------------------------------------------------------------------------
// <copyright file="IMacAllocatorService.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.MacAllocator
{
    using System.Threading.Tasks;

    using Microsoft.Azure.ContainerNetworking.MacManager.Common;

    public interface IMacAllocatorService
    {
        Task<MacAddress> AllocateMacAddressAsync(MacAllocatorRequest request);
        Task DeallocateMacAddressAsync(MacAllocatorRequest request);
    }
}
