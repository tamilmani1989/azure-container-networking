// -----------------------------------------------------------------------------
// <copyright file="MacAllocationRequest.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.MacAllocator
{
    public sealed class MacAllocatorRequest
    {
        /// <summary>
        /// Represents the DNC ID requesting to allocate/deallocate MAC.
        /// </summary>
        public string AllocationId { get; set; }

        /// <summary>
        /// MAC allocation type specified by DNC.
        /// </summary>
        public string Type { get; set; }

        /// <summary>
        /// MAC allocation tag specified by DNC.
        /// </summary>
        public string Tag { get; set; }
    }
}
