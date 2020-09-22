// --------------------------------------------------------------------------------------------------------------------
// <copyright file="OrchestratorProperties.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;

    public class OrchestratorProperties : TopLevelResourceProperties
    {
        [JsonProperty("orchestratorInstanceDetails")]
        public OrchestratorInstanceDetails orchestratorInstanceDetails
        {
            get; set;
        }

        [JsonProperty("controllerDetails")]
        public ControllerDetails ControllerDetails
        {
            get; set;
        }
    }
}
