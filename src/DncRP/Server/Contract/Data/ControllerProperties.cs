// -----------------------------------------------------------------------------
// <copyright file="ControllerProperties.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;

    public class ControllerProperties : TopLevelResourceProperties
    {
        [JsonProperty("dncAppId")]
        public string DncAppId
        {
            get; set;
        }

        [JsonProperty("dncEndpoint")]
        public string DncEndpoint
        {
            get; set;
        }
    }
}
