// -----------------------------------------------------------------------------
// <copyright file="MacAddressRangeControllers.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.MacInventoryManager.Controllers
{
    using System.Collections.Generic;
    using System.Threading.Tasks;
    using Microsoft.AspNetCore.Mvc;

    [ApiController]
    [Route("/macinventory/macaddressranges")]
    public class MacAddressRangesController : ControllerBase
    {
        private readonly IMacInventoryManagerService service;

        public MacAddressRangesController(IMacInventoryManagerService service)
        {
            this.service = service;
        }

        /// <summary>
        /// Writes a MacAddressRange to the store and flattens it out, so it's available
        /// to the allocator. 
        /// Example: PUT/POST https://localhost:5004/macaddressranges 
        /// With JSON body: 
        /// { "RangeStart": "AAAAAAAAAAAA", "RangeEnd": "AAAAAAAAAAAA", "BlockedReason": "" }
        /// </summary>
        /// <param name="range"></param>
        /// <returns></returns>
        [HttpPost]
        [HttpPut]
        public async Task<MacAddressRange> PutRangeAsync(MacAddressRange range)
        {
            return await service.UpdateRangeAsync(range);
        }

        /// <summary>
        /// Gets the ranges from the store.
        /// Example: GET https://localhost:5004/macaddressranges
        /// </summary>
        /// <returns></returns>
        [HttpGet]
        public async Task<IEnumerable<MacAddressRange>> GetRangesAsync()
        {
            return await service.GetRangesAsync();
        }
    }
}
