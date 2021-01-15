// -----------------------------------------------------------------------------
// <copyright file="IMacAddressRangeComponent.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.MacInventoryManager.DataAccess
{
    using System.Collections.Generic;
    using System.Threading.Tasks;

    public interface IMacAddressRangeComponent
    {
        Task<List<MacAddressRange>> GetRangesAsync();
        Task UpsertRangeAsync(MacAddressRange range);
    }
}