//-----------------------------------------------------------------------
// <copyright file="DeleteNetworkContainerRequest.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// 
//-----------------------------------------------------------------------

using Newtonsoft.Json;

namespace Cns.Data
{
    /// <summary>
    /// Defines the delete network container request.
    /// </summary>
    class DeleteNetworkContainerRequest
    {
        /// <summary>
        /// Id of the network container.
        /// </summary>
        [JsonProperty("NetworkContainerid")]
        public string NetworkContainerID { get; private set; }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="networkcontainerID"></param>
        public DeleteNetworkContainerRequest(string networkcontainerID)
        {
            NetworkContainerID = "Swift_" + networkcontainerID;
        }

        /// <summary>
        /// ToString implementation
        /// </summary>
        /// <returns>String representation of the object.</returns>
        public override string ToString()
        {
            return $"NetworkContainerID:{NetworkContainerID}";
        }
    }
}
