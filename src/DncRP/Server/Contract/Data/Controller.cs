// -----------------------------------------------------------------------------
// <copyright file="ControllerRequest.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
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

            this.isExtensionResource = false;
            this.Type = ControllerResourceType;
        }

        /// <summary>
        /// Initializes a new instance of the <see cref="Controller"/> class.
        /// </summary>
        public Controller(string subscriptionId, string resourceGroup, string resourceName, ControllerProperties properties) :
            this(subscriptionId, resourceGroup, resourceName)
        {
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
    }
}
