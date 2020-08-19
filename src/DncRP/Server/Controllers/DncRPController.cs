// --------------------------------------------------------------------------------------------------------------------
// <copyright file="DncRPController.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Net;
    using Microsoft.AspNetCore.Authorization;
    using Microsoft.AspNetCore.Mvc;
    using Microsoft.Extensions.Logging;

    [ApiController]
    [Route("/{**routePath}")]
    [Authorize]
    public class DncRPController : ControllerBase
    {
        private readonly ILogger<DncRPController> _logger;

        public DncRPController(
            ILogger<DncRPController> logger)
        {
            _logger = logger;
        }

        [Route("/subregistration")]
        public ContentResult HandleSubRegistration()
        {
            return new ContentResult { StatusCode = (int)HttpStatusCode.OK, Content = "DncRP testing", ContentType = "text/html" };
        }
    }
}
