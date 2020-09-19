// --------------------------------------------------------------------------------------------------------------------
// <copyright file="SubnetDetails.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;

    public class SubnetDetails
    {
        [JsonProperty(PropertyName = "id")]
        public string Id { get; set; }
    }
}
