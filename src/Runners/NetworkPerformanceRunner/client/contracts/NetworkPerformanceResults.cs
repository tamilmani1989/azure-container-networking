//--------------------------------------------------------------------------
// <copyright file="DNCService.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
//   DncDataPathResponse is a contract (of datapath test results) 
//   between runner client and server.
// </summary>
//--------------------------------------------------------------------------

namespace NetworkPerformanceRunner
{
    using Newtonsoft.Json;
    
    namespace contracts
    {
        /// <summary>
        /// Defines the response.
        /// </summary>
        public class NetworkPerformanceResults
        {
            /// <summary>
            /// CPUSender
            /// </summary>
            [JsonProperty("CPUSender")]
            public int CPUSender { get; private set; }

            /// <summary>
            /// CPUReceiver
            /// </summary>
            [JsonProperty("CPUReceiver")]
            public int CPUReceiver { get; private set; }

            /// <summary>
            /// Throughput
            /// </summary>
            [JsonProperty("Throughput")]
            public long Throughput { get; private set; }

            /// <summary>
            /// LatencyMin
            /// </summary>
            [JsonProperty("LatencyMin")]
            public int LatencyMin { get; private set; }

            /// <summary>
            /// LatencyMax
            /// </summary>
            [JsonProperty("LatencyMax")]
            public int LatencyMax { get; private set; }

            /// <summary>
            /// LatencyAvg
            /// </summary>
            [JsonProperty("LatencyAvg")]
            public int LatencyAvg { get; private set; }

            /// <summary>
            /// Default Constructor.
            /// </summary>
            public NetworkPerformanceResults()
            {
                CPUSender = 0;
                CPUReceiver = 0;
                Throughput = 0;
                LatencyMin =  0;
                LatencyMax = 0;
                LatencyAvg = 0;
            }

            /// <summary>
            /// Copy Constructor.
            /// </summary>
            /// <param name="response"></param>
            public NetworkPerformanceResults(NetworkPerformanceResults response)
            {
                CPUSender = response.CPUSender;
                CPUReceiver = response.CPUReceiver;
                Throughput = response.Throughput;
                LatencyMin = response.LatencyMin;
                LatencyMax = response.LatencyMax;
                LatencyAvg = response.LatencyAvg;
            }

            /// <summary>
            /// Constructor.
            /// </summary>
            /// <param name="serviceTunnel"></param>
            /// <param name="nic2Nic"></param>
            /// <param name="nic2VM"></param>
            /// <param name="vnet2VnetGW"></param>
            public NetworkPerformanceResults(int cpuSender, int cpuReceiver, long throughput, int latencyMin, int latencyMax, int latencyAvg)
            {
                CPUSender = cpuSender;
                CPUReceiver = cpuReceiver;
                Throughput = throughput;
                LatencyMin = latencyMin;
                LatencyMax = latencyMax;
                LatencyAvg = latencyAvg;
            }

            /// <summary>
            /// ToString implementation
            /// </summary>
            /// <returns>String representation of the object.</returns>
            public override string ToString()
            {
                return $"[CPUSender:{CPUSender}% CPUReceiver:{CPUReceiver}% Throughput:{Throughput}bits/s LatencyMin:{LatencyMin}us LatencyMax:{LatencyMax}us LatencyAvg:{LatencyAvg}us]";
            }
        }
    }
}