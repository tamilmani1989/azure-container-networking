// -----------------------------------------------------------------------------
// <copyright file="MacAddressStatsController.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.MacInventoryManager.Controllers
{
    using System;
    using System.Collections.Generic;
    using System.Threading.Tasks;
    using Microsoft.AspNetCore.Mvc;

    [ApiController]
    [Route("/macinventory/macaddressstats")]
    public class MacAddressStatsController : ControllerBase
    {
        private readonly IMacInventoryManagerService managerService;

        public MacAddressStatsController(IMacInventoryManagerService managerService)
        {
            this.managerService = managerService;
        }

        /// <summary>
        /// Gets stats from the store.
        /// Example: https://localhost:5004/macinventory/macaddressstats
        /// </summary>
        /// <returns></returns>
        public async Task<IEnumerable<MacAddressStats>> GetStatsAsync()
        {
            return await this.managerService.GetMacAddressStatsAsync();
        }
    }
}
