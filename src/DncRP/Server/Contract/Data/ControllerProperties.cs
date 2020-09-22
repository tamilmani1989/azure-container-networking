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
        [JsonProperty("controllerProperties")]
        public ControllerPropertiesDetails PropertiesList
        {
            get; set;
        }
    }

    public class ControllerPropertiesDetails
    {
        [JsonProperty("appId")]
        public string DncAppId
        {
            get; set;
        }

        [JsonProperty("endpoint")]
        public string DncEndpoint
        {
            get; set;
        }
    }
}
