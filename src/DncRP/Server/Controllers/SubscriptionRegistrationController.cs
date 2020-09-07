// --------------------------------------------------------------------------------------------------------------------
// <copyright file="SubscriptionRegistrationController.cs" company="Microsoft">
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

    public class SubscriptionRegistrationController : DncRpControllerBase
    {
        public SubscriptionRegistrationController(
            ILogger<SubscriptionRegistrationController> logger,
            ITableOperations tableOperations)
            : base(logger, tableOperations)
        {
            logger.LogInformation("SubscriptionRegistrationController initialized");
        }

        /// <summary>
        /// Customer will first Register Subscription with DNCRP
        /// </summary>
        /// <returns></returns>
        [Route("subscriptions/{subscriptionId}")]
        public ContentResult HandleSubRegistration(string subscriptionId)
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
