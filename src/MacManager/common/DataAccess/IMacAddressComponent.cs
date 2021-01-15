// -----------------------------------------------------------------------------
// <copyright file="IMacAddressComponent.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.MacManager.Common
{
    using System.Collections.Generic;
    using System.Threading.Tasks;

    public interface IMacAddressComponent
    {
        Task<List<MacAddress>> GetMacAddressesAsync(string rangeStart, string rangeEnd);
        Task UpsertMacAddressAsync(MacAddress address);
    }
}
