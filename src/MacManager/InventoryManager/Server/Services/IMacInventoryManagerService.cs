// -----------------------------------------------------------------------------
// <copyright file="IMacInventoryManagerService.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.MacInventoryManager
{
    using System.Collections.Generic;
    using System.Threading.Tasks;

    public interface IMacInventoryManagerService
    {
        Task<List<MacAddressRange>> GetRangesAsync();
        Task<MacAddressRange> UpdateRangeAsync(MacAddressRange range);
        Task<IEnumerable<MacAddressStats>> GetMacAddressStatsAsync();
    }
}