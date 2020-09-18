// --------------------------------------------------------------------------------------------------------------------
// <copyright file="DelegateRequestParameters.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;

    public class DelegateRequestParameters
    {
        [JsonProperty("controllerID")]
        public string ControllerId
        {
            get; set;
        }
    }

    public class DelegateSubnetRequest
    {
        [JsonProperty("delegateSubnetParameters")]
        public DelegateRequestParameters Parameters
        {
            get; set;
        }
    }
}
