// -----------------------------------------------------------------------------
// <copyright file="Orchestrator.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;

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
        public Orchestrator(
            string subscriptionId,
            string resourceGroup,
            string resourceName,
            string location,
            OrchestratorKind kind,
            OrchestratorProperties properties,
            MSIIdentityResponse orchestratorIdentity) :
            this(subscriptionId, resourceGroup, resourceName)
        {
            this.Properties = properties;
            this.Location = location;
            this.Kind = kind;
            this.OrchestratorIdentity = orchestratorIdentity;
        }

        [JsonProperty("kind")]
        public OrchestratorKind Kind
        {
            get; set;
        }

        [JsonProperty("identity")]
        public MSIIdentityResponse OrchestratorIdentity
        {
            get; set;
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
