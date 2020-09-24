// -----------------------------------------------------------------------------
// <copyright file="Orchestrator.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    public class Orchestrator : TopLevelResource
    {
        /// <summary>
        /// Gets the type of orchestrator
        /// </summary>
        private const string OrchestratorResourceType = "Microsoft.DelegatedNetwork/orchestratorInstances";

        public Orchestrator()
        {
        }

        /// <summary>
        /// Initializes a new instance of the <see cref="Orchestrator"/> class.
        /// </summary>
        public Orchestrator(string subscriptionId, string resourceGroup, string resourceName)
        {
            this.SubscriptionId = subscriptionId;
            this.ResourceGroup = resourceGroup;
            this.Name = resourceName;

            this.Type = OrchestratorResourceType;
        }

        /// <summary>
        /// Initializes a new instance of the <see cref="Orchestrator"/> class.
        /// </summary>
        public Orchestrator(string subscriptionId, string resourceGroup, string resourceName, string location, OrchestratorProperties properties) :
            this(subscriptionId, resourceGroup, resourceName)
        {
            this.Properties = properties;
            this.Location = location;
        }

        public static Orchestrator CreateErrorResponse(string errorMessage, int httpStatusCode)
        {
            return CreateErrorResponse(errorMessage, httpStatusCode, ErrorResponseCode.None);
        }

        public static Orchestrator CreateErrorResponse(string errorMessage, int httpStatusCode, ErrorResponseCode apiErrorCode)
        {
            return new Orchestrator
            {
                Error = new TopLevelResourceError
                {
                    Error = errorMessage,
                    StatusCode = httpStatusCode,
                    ErrorResponseCode = apiErrorCode
                }
            };
        }
    }
}
