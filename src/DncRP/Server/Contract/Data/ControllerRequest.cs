// -----------------------------------------------------------------------------
// <copyright file="ControllerRequest.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;
    public class ControllerRequest
    {
        [JsonProperty("location")]
        public string Location
        {
            get; set;
        }
    }

}
