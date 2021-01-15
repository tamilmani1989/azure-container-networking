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
    [Route("/macaddressstats")]
    public class MacAddressStatsController : ControllerBase
    {
        public Task<IEnumerable<MacAddressStats>> GetStatsAsync()
        {
            throw new NotImplementedException();
        }
    }
}
