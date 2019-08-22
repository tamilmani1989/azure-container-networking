//
//     Copyright (c) Microsoft Corporation.  All rights reserved.
//

using Newtonsoft.Json;

namespace Cns.Data
{
    /// <summary>
    /// Defines the CNS response for Number of CPU cores request.
    /// </summary>
    public class NumberOfCPUCoresResponse
    {
        [JsonProperty("Response")]
        public Response Response { get; set; }

        [JsonProperty("NumOfCPUCores")]
        public int NumOfCPUCores { get; set; }

        public NumberOfCPUCoresResponse(Response response, int numOfCPUCores)
        {
            Response = response;
            NumOfCPUCores = numOfCPUCores;
        }

        public override string ToString()
        {
            return $"[Response:{this.Response.ToString()}, NumOfCPUCores:{this.NumOfCPUCores}";
        }
    }
}
