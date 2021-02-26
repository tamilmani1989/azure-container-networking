// -----------------------------------------------------------------------------
// <copyright file="MacAllocatorResponse.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.MacAllocator
{
    public sealed class MacAllocatorResponse
    {
        public MacAllocatorResponse(string allocationId, string macAddress)
        {
            AllocationId = allocationId;
            MacAddress = macAddress;
            ShouldRetry = false;
        }

        public MacAllocatorResponse(string allocationId, string errorMessage, bool shouldRetry)
        {
            AllocationId = allocationId;
            ErrorMessage = errorMessage;
            ShouldRetry = shouldRetry;
        }

        /// <summary>
        /// The AllocationID of the request
        /// </summary>
        public string AllocationId { get; private set; }

        /// <summary>
        /// The mac address if successful and is an allocation request.
        /// Null for a deallocation request.
        /// </summary>
        public string MacAddress { get; private set; }

        /// <summary>
        /// The error message if not successful
        /// </summary>
        public string ErrorMessage { get; private set; }

        /// <summary>
        /// Whether to retry the request or not.
        /// </summary>
        public bool ShouldRetry { get; private set; }

        /// <summary>
        /// Whether the operation was successful or not.
        /// </summary>
        public bool IsSuccess
        {
            get
            {
                return string.IsNullOrEmpty(ErrorMessage);
            }
        }
    }
}
