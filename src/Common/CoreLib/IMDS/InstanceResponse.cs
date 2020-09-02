// -----------------------------------------------------------------------------
// <copyright file="ImdsResponse.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using System.Text.Json.Serialization;

    public class ComputeResponse
    {
        [JsonPropertyName("azEnvironment")]
        public string AzureEnvironment
        {
            get; set;
        }

        [JsonPropertyName("location")]
        public string AzureRegion
        {
            get; set;
        }
    }

    /// <summary>
    /// Object type that is returned when following IMDS API in invoked
    /// http://169.254.169.254/metadata/instance?api-version=2019-08-15&format=json
    /// </summary>
    public class InstanceResponse
    {
        [JsonPropertyName("compute")]
        public ComputeResponse Compute
        {
            get; set;
        }
    }
}
