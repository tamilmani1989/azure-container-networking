// --------------------------------------------------------------------------------------------------------------------
// <copyright file="DncRPController.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Net;
    using Microsoft.AspNetCore.Authorization;
    using Microsoft.AspNetCore.Http.Extensions;
    using Microsoft.AspNetCore.Mvc;
    using Microsoft.Extensions.Logging;

    [ApiController]
    [Route("/{**routePath}")]
    [Authorize]
    public class DncRPController : ControllerBase
    {
        private readonly ILogger<DncRPController> logger;

        public DncRPController(
            ILogger<DncRPController> logger)
        {
            this.logger = logger;
        }

        [Route("/subregistration")]
        public ContentResult HandleSubRegistration()
        {
            this.logger.LogInformation($"HandleSubRegistration URI: {HttpContext.Request.GetDisplayUrl()}");
            return new ContentResult { StatusCode = (int)HttpStatusCode.OK, Content = "DncRP testing", ContentType = "text/html" };
        }

        [Route("/{*url}")]
        public ContentResult CatchAll()
        {
            this.logger.LogInformation($"CatchAll URI: {HttpContext.Request.GetDisplayUrl()}");
            return new ContentResult { StatusCode = (int)HttpStatusCode.OK, Content = "DncRP testing (CatchAll)", ContentType = "text/html" };
        }
    }
}
