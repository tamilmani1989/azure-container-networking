// -----------------------------------------------------------------------------
// <copyright file="ControllerRequest.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Collections.Generic;

    public class Controller : TopLevelResource
    {
        /// <summary>
        /// Gets the container group type.
        /// </summary>
        private const string ControllerResourceType = "Microsoft.DelegatedNetwork/controller";

        public Controller()
        {
        }

        /// <summary>
        /// Initializes a new instance of the <see cref="Controller"/> class.
        /// </summary>
        public Controller(string subscriptionId, string resourceGroup, string resourceName)
        {
            this.SubscriptionId = subscriptionId;
            this.ResourceGroup = resourceGroup;
            this.Name = resourceName;

            this.Type = ControllerResourceType;
        }

        /// <summary>
        /// Initializes a new instance of the <see cref="Controller"/> class.
        /// </summary>
        public Controller(string subscriptionId, string resourceGroup, string resourceName, string location, ControllerProperties properties) :
            this(subscriptionId, resourceGroup, resourceName)
        {
            this.Location = location;
            this.Properties = properties;
        }

        public static Controller CreateErrorResponse(string errorMessage, int httpStatusCode)
        {
            return new Controller
            {
                Error = new TopLevelResourceError
                {
                    Error = errorMessage,
                    StatusCode = httpStatusCode
                }
            };
        }

        public static Controller CreateErrorResponse(string errorMessage, int httpStatusCode, ErrorResponseCode apiErrorCode)
        {
            return new Controller
            {
                Error = new TopLevelResourceError
                {
                    Error = errorMessage,
                    StatusCode = httpStatusCode,
                    ErrorResponseCode = apiErrorCode
                }
            };
        }

        /// <summary>
        /// Get controller resouce by parsing ARM resource id
        /// </summary>
        /// <param name="armResourceId"></param>
        /// <param name="controller"></param>
        /// <returns></returns>
        public static bool TryParseController(string armResourceId, out Controller controller)
        {
            Dictionary<string, string> dict = null;
            if (!TryParseResourceId(armResourceId, out dict))
            {
                controller = null;
                return false;
            }

            controller = new Controller
            {
                SubscriptionId = dict[ResourceConstants.SubscriptionId],
                ResourceGroup = dict[ResourceConstants.ResourceGroup],
                Name = dict[ResourceConstants.ResourceName],
                Type = dict[ResourceConstants.ProviderType]
            };

            return true;
        }
    }
}
