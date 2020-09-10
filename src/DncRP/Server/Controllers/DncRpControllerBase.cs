// --------------------------------------------------------------------------------------------------------------------
// <copyright file="DncRpControllerBase.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Net;
    using Microsoft.AspNetCore.Authorization;
    using Microsoft.AspNetCore.Http.Extensions;
    using Microsoft.AspNetCore.Mvc;
    using Microsoft.Extensions.Configuration;
    using Microsoft.Extensions.Logging;

    [ApiController]
    [Authorize]
    public class DncRpControllerBase<T> : ControllerBase
    {
        protected readonly ILogger<T> logger;
        protected readonly ITableOperations tableOperations;

        public DncRpControllerBase(
            ILogger<T> logger,
            ITableOperations tableOperations)
        {
            this.logger = logger;
            this.tableOperations = tableOperations;
        }
    }
}
