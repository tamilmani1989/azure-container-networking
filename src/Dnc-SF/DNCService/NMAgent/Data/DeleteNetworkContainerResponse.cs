using Newtonsoft.Json;
using System;
using System.Collections.Generic;
using System.Text;

namespace NMAgent.Data
{
    class DeleteNetworkContainerResponse
    {
        [JsonProperty("httpStatusCode")]
        public string ResponseCode;

        public DeleteNetworkContainerResponse(string responseCode)
        {
            this.ResponseCode = responseCode;
        }
        public override string ToString()
        {
            return $"[ResponseCode:{this.ResponseCode}]";
        }
    }
}
