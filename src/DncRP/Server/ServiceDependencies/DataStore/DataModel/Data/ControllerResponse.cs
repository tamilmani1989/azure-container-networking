// -----------------------------------------------------------------------------
// <copyright file="ControllerResponse.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Net;
    using System.Text.Json.Serialization;

    public class ResponseError
    {
        public HttpStatusCode StatusCode
        {
            get; set;
        }

        public string Error
        {
            get; set;
        }
    }

    public enum RequestState
    {
        Succeeded = 0,
        Provisioning
    }

    public class ControllerResponseProperties
    {
        [JsonPropertyName("state")]
        public RequestState State
        {
            get; set;
        }
    }

    public class ControllerResponse
    {
        [JsonPropertyName("id")]
        public string Id
        {
            get; set;
        }

        [JsonPropertyName("location")]
        public string Region
        {
            get; set;
        }

        [JsonPropertyName("name")]
        public string ResourceName
        {
            get; set;
        }

        [JsonPropertyName("type")]
        public string ResourceType
        {
            get; set;
        }

        [JsonPropertyName("properties")]
        public ControllerResponseProperties Properties
        {
            get; set;
        }

        [JsonIgnore]
        public ResponseError Error
        {
            get; set;
        }
    }
}