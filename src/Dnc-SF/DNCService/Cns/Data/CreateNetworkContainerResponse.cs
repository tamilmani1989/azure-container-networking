//-----------------------------------------------------------------------
// <copyright file="CreateNetworkContainerResponse.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// 
//-----------------------------------------------------------------------

using Newtonsoft.Json;
using System;
using System.Collections.Generic;
using System.Runtime.Serialization;
using System.Text;

namespace Cns.Data
{
    /// <summary>
    /// Defines the create network container response.
    /// </summary>
    public class CreateNetworkContainerResponse
    {
        [JsonProperty("Response")]
        public Response Response { get; set; }

        public CreateNetworkContainerResponse(int returncode, string message)
        {
            Response = new Response(returncode, message);
        }

        public CreateNetworkContainerResponse()
        {
        }

        public CreateNetworkContainerResponse(Response response)
        {
            Response = response;
        }
    }
}
