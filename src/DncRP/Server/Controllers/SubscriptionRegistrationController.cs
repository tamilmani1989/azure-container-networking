// --------------------------------------------------------------------------------------------------------------------
// <copyright file="SubscriptionRegistrationController.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.IO;
    using System.Net;
    using System.Threading.Tasks;
    using Microsoft.AspNetCore.Authorization;
    using Microsoft.AspNetCore.Http.Extensions;
    using Microsoft.AspNetCore.Mvc;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Configuration;
    using Microsoft.Extensions.Logging;


    [Route("subscriptions/{subscriptionId}")]
    public class SubscriptionRegistrationController : DncRpControllerBase<SubscriptionRegistrationController>
    {
        private AzSubscriptionBroker azSubscriptionBroker;

        public SubscriptionRegistrationController(
            ILogger<SubscriptionRegistrationController> logger,
            ITableOperations tableOperations,
            IHttpUtility httpUtility)
            : base(logger, tableOperations)
        {
            this.azSubscriptionBroker = new AzSubscriptionBroker(httpUtility, logger);
            logger.CLogInformation("SubscriptionRegistrationController initialized");
        }

        /// <summary>
        /// Customer will first Register Subscription with DNCRP
        /// </summary>
        /// <returns></returns>
        [HttpPut]
        public async Task<ContentResult> PutSubscriptionRegistration(string subscriptionId)
        {
            this.logger.CLogInformation($"PutSubscriptionRegistration URI: {HttpContext.Request.GetDisplayUrl()}");

            if (HttpContext.Request.Body == null)
            {
                throw new EmptyRequestBody("Empty request body");
            }

            try
            {
                string jsonBody = await AddOrUpdateSubscriptionAsync(subscriptionId);
                return new ContentResult { 
                    StatusCode = (int)HttpStatusCode.OK, 
                    Content = jsonBody, 
                    ContentType = "application/json" 
                };
            }
            catch (Exception ex)
            {
                string message = $"Failed to process PutSubscriptionRegistration call.";
                int statusCode = (int)HttpStatusCode.InternalServerError;

                ErrorResponseMessageException errorException = ex as ErrorResponseMessageException;
                if (errorException != null)
                {
                    statusCode = (int)errorException.HttpStatus;
                    message += errorException.Message + $"InternalErrorCode: {errorException.ErrorCode}";
                }

                message += $"InternalException: {ex}";

                return ConvertToExceptionResult(message, statusCode);
            }
        }

        private async Task<string> AddOrUpdateSubscriptionAsync(string subscriptionId)
        {
            SubscriptionRegistrationRequest request = null;
            string jsonBody = String.Empty;
            using (StreamReader stream = new StreamReader(HttpContext.Request.Body))
            {
                jsonBody = await stream.ReadToEndAsync();
                this.logger.CLogInformation("PutSubscriptionRegistration body: {0}", jsonBody);

                request = JsonUtility.Deserialize<SubscriptionRegistrationRequest>(jsonBody);
            }

            // Check if Subscription is already registered
            switch (SubscriptionRegistrationRequest.ParseRegistrationState(request.State))
            {
                case SubscriptionRegistrationState.Registered:
                    {
                        bool registered = await tableOperations.SubscriptionOperation.IsSubscriptionRegisteredAsync(subscriptionId);
                        if (!registered)
                        {
                            string tenantId = await azSubscriptionBroker.GetTenantIdFromSubscriptionIdAsync(subscriptionId);
                            await tableOperations.SubscriptionOperation.RegisterSubAsync(subscriptionId, tenantId, jsonBody);
                        }
                        break;
                    }
                case SubscriptionRegistrationState.Unregistered:
                case SubscriptionRegistrationState.Deleted:
                    {
                        await tableOperations.SubscriptionOperation.UnregisterSubAsync(subscriptionId);
                        break;
                    }
                default:
                    throw new NotSupportedException($"PutSubscription failed, state :{request.State} not supported");
            }

            return jsonBody;
        }
    }
}
