//-----------------------------------------------------------------------
// <copyright file="DeleteNetworkContainerResponse.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// 
//-----------------------------------------------------------------------
using Newtonsoft.Json;

namespace Cns.Data
{
    /// <summary>
    /// Defines the delete container response,
    /// </summary>
    public class DeleteNetworkContainerResponse
    {
        [JsonProperty("Response")]
        public Response Response { get; set; }

        public DeleteNetworkContainerResponse(int returncode, string message)
        {
            Response = new Response(returncode, message);
        }

        public DeleteNetworkContainerResponse()
        {
        }

        public DeleteNetworkContainerResponse(Response response)
        {
            Response = response;
        }
    }
}
