// -----------------------------------------------------------------------------
// <copyright file="MacManagerConstants.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.MacManager.Common
{
    public static class MacManagerConstants
    {
        public const string DatabaseName = "MacManagerDatabase";

        public const string MacAddressRangesContainerName = "MacAddressRange";
        public const string MacAddressRangePartitionKeyPath = "/RangeStart";

        public const string MacAddressContainerName = "MacAddress";
        public const string MacAddressContainerPartitionKeyPath = "/Address";
    }
}
