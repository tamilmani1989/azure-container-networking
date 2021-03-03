// -----------------------------------------------------------------------------
// <copyright file="MacDeallocateController.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.MacAllocator.Controller
{
    using System.Net;
    using System.Threading.Tasks;
    using Microsoft.AspNetCore.Mvc;
    using Microsoft.Azure.ContainerNetworking.CoreLib;

    [ApiController]
    public class MacAllocatorController
    {
        private readonly IMacAllocatorService service;

        public MacAllocatorController(IMacAllocatorService service)
        {
            this.service = service;
        }

        /// <summary>
        /// Allocates a MAC address from the store.
        /// Example: PUT/POST https://localhost:5012/macallocator/allocate
        /// With JSON body: 
        /// { "AllocationId": "soumyaptest1", Type: "", Tag:"" }
        /// </summary>
        /// <param name="request"></param>
        /// <returns>
        /// { "allocationId": "soumyaptest1", "macAddress": "125A00B000", "errorMessage": null, "shouldRetry": false, "isSuccess": true }
        /// </returns>
        [HttpPost]
        [HttpPut]
        [Route("/macallocator/allocate")]
        public async Task<ContentResult> AllocateMacAsync(MacAllocatorRequest request)
        {
            MacAllocatorResponse response = await service.AllocateMacAddressAsync(request);
            return this.CreateResult(response);
        }

        /// <summary>
        /// Allocates a MAC address from the store.
        /// Example: PUT/POST https://localhost:5012/macallocator/deallocate 
        /// With JSON body: 
        /// { "AllocationId": "soumyaptest1", Type: "", Tag:"" }
        /// </summary>
        /// <param name="request"></param>
        /// <returns>
        /// { "allocationId": "soumyaptest1", "macAddress": null, "errorMessage": null, "shouldRetry": false, "isSuccess": true }
        /// </returns>
        [HttpPost]
        [HttpPut]
        [Route("/macallocator/deallocate")]
        public async Task<ContentResult> DeallocateMacAsync(MacAllocatorRequest request)
        {
            MacAllocatorResponse response = await service.DeallocateMacAddressAsync(request);
            return this.CreateResult(response);
        }

        /// <summary>
        /// Method to create the ContentResult from MacAllocatorResponse
        /// </summary>
        private ContentResult CreateResult(MacAllocatorResponse response)
        {
            if (response.IsSuccess)
            {
                return new ContentResult
                {
                    StatusCode = (int)HttpStatusCode.OK,
                    Content = JsonUtility.Serialize(response),
                    ContentType = "application/json"
                };
            }
            else
            {
                return new ContentResult
                {
                    StatusCode = response.ShouldRetry ? (int)HttpStatusCode.InternalServerError : (int)HttpStatusCode.BadRequest,
                    Content = JsonUtility.Serialize(response),
                    ContentType = "application/json"
                };
            }
        }
    }
}
