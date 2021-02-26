// -----------------------------------------------------------------------------
// <copyright file="MacDeallocateController.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.MacAllocator.Controller
{
    using System.Threading.Tasks;
    using Microsoft.AspNetCore.Mvc;

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
        /// Example: PUT/POST https://localhost:5012/allocate 
        /// With JSON body: 
        /// { "AllocationId": "asdasasdasdsa", Type: "abcd", Tag:"mdnc" }
        /// </summary>
        /// <param name="request"></param>
        /// <returns>
        /// { MAC: "000a959d6816" }
        /// </returns>
        [HttpPost]
        [HttpPut]
        [Route("/macallocator/allocate")]
        public async Task<MacAllocatorResponse> AllocateMacAsync(MacAllocatorRequest request)
        {
            return await service.AllocateMacAddressAsync(request);
        }

        /// <summary>
        /// Allocates a MAC address from the store.
        /// Example: PUT/POST https://localhost:5012/deallocate 
        /// With JSON body: 
        /// { "AllocationId": "asdasasdasdsa", Type: "", Tag:"" }
        /// </summary>
        /// <param name="request"></param>
        [HttpPost]
        [HttpPut]
        [Route("/macallocator/deallocate")]
        public async Task<MacAllocatorResponse> DeallocateMacAsync(MacAllocatorRequest request)
        {
            return await service.DeallocateMacAddressAsync(request);
        }
    }
}
