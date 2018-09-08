using Newtonsoft.Json;
using System;
using System.Collections.Generic;
using System.Text;

namespace NMAgent.Data
{
    class CreateNetworkContainerResponse
    {
        [JsonProperty("httpStatusCode")]
        public string ResponseCode;

        [JsonProperty("networkContainerId")]
        public string NetworkContainerID;

        [JsonProperty("version")]
        public string Version;

        public CreateNetworkContainerResponse(string responseCode, string networkContainerID, string version)
        {
            this.ResponseCode = responseCode;
            this.NetworkContainerID = networkContainerID;
            this.Version = version;
        }
        public override string ToString()
        {
            return $"[ResponseCode:{this.ResponseCode}, NetworkContainerId:{this.NetworkContainerID}" +
                $"Version:{this.Version}]";
        }
    }
}
