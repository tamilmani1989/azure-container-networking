// --------------------------------------------------------------------------------------------------------------------
// <copyright file="ControllerDetails.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;

    public class ControllerDetails
    {
        [JsonProperty(PropertyName = "id")]
        public string Id { get; set; }
    }
}
