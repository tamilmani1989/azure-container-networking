// --------------------------------------------------------------------------------------------------------------------
// <copyright file="ResourceDefinition.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------


namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;
    using System;

    public class TopLevelResourceProperties
    {
        /// <summary>
        /// Gets or sets the type of the resource.
        /// </summary>
        [JsonProperty(Required = Required.Default, PropertyName = "resourceGuid")]
        public string ResourceGuid { get; set; }

        /// <summary>
        /// Gets or sets the type of the resource.
        /// </summary>
        [JsonProperty(Required = Required.Default, PropertyName = "provisioningState")]
        public ProvisioningState State { get; set; }

        /// <summary>
        /// Convert state to enum
        /// </summary>
        /// <param name="state"></param>
        /// <returns></returns>
        public static ProvisioningState ParseProvisioningState(string state)
        {
            return (ProvisioningState)Enum.Parse(typeof(ProvisioningState), state, true);
        }
    }
}
