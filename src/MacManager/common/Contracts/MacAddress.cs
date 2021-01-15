// -----------------------------------------------------------------------------
// <copyright file="MacAddress.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.MacManager.Common
{
    using System;
    using Newtonsoft.Json;

    public class MacAddress
    {
        [JsonProperty("id")]
        public string Address { get; set; }

        public string ReservedBy { get; set; }

        public DateTime ReservedDateTime { get; set; }

        public bool IsBlocked { get; set; }

        [JsonProperty("_etag")]
        internal string ETag { get; set; }
    }
}
