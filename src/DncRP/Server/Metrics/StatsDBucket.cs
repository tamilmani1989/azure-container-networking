// -----------------------------------------------------------------------------
// <copyright file="StatsDBucket.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Collections.Generic;
    using Newtonsoft.Json;

    public class StatsDBucket
    {
        [JsonProperty(PropertyName = "Namespace")]
        public string Namespace
        {
            get; set;
        }

        [JsonProperty(PropertyName = "Metric")]
        public string Metric
        {
            get; set;
        }

        [JsonProperty(PropertyName = "Dims")]
        public Dictionary<string, string> Dims
        {
            get; set;
        }
    }
}
