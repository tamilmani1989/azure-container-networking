// -----------------------------------------------------------------------------
// <copyright file="IMacAllocatorService.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.MacAllocator
{
    using System.Threading.Tasks;

    public interface IMacAllocatorService
    {
        Task<MacAllocatorResponse> AllocateMacAddressAsync(MacAllocatorRequest request);
        Task<MacAllocatorResponse> DeallocateMacAddressAsync(MacAllocatorRequest request);
    }
}
