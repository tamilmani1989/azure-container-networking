// -----------------------------------------------------------------------------
// <copyright file="MacAddressStats.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.MacInventoryManager
{
    public class MacAddressStats
    {
        /// <summary>
        /// Start of the range.
        /// </summary>
        public string RangeStart { get; set; }

        /// <summary>
        /// End of the range
        /// </summary>
        public string RangeEnd { get; set; }

        /// <summary>
        /// If set the range will not be used.
        /// </summary>
        public string BlockedReason { get; set; }

        /// <summary>
        /// The total count of mac addresses in the range
        /// </summary>
        public long TotalCount { get; set; }

        /// <summary>
        /// The reserved count of mac addresses in the range
        /// </summary>
        public long ReservedCount { get; set; }
    }
}
