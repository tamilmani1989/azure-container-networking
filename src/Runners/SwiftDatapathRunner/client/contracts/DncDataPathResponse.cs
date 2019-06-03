//--------------------------------------------------------------------------
// <copyright file="DNCService.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
//   DncDataPathResponse is a contract (of datapath test results) 
//   between runner client and server.
// </summary>
//--------------------------------------------------------------------------

namespace SwiftRunner
{
    using Newtonsoft.Json;
    
    namespace contracts
    {
        /// <summary>
        /// Defines the response.
        /// </summary>
        public class DncDataPathResponse
        {
            /// <summary>
            /// ServiceTunnel
            /// </summary>
            [JsonProperty("ServiceTunnel")]
            public bool ServiceTunnel { get; private set; }

            /// <summary>
            /// Nic2Nic
            /// </summary>
            [JsonProperty("Nic2Nic")]
            public bool Nic2Nic { get; private set; }

            /// <summary>
            /// Nic2VM
            /// </summary>
            [JsonProperty("Nic2VM")]
            public bool Nic2VM { get; private set; }

            /// <summary>
            /// Vnet2VnetGW
            /// </summary>
            [JsonProperty("Vnet2VnetGW")]
            public bool Vnet2VnetGW { get; private set; }

            /// <summary>
            /// Vnet2VnetPeering
            /// </summary>
            [JsonProperty("Vnet2VnetPeering")]
            public bool Vnet2VnetPeering { get; private set; }

            /// <summary>
            /// LastError
            /// </summary>
            [JsonProperty("LastError")]
            public string LastError { get; private set; }

            /// <summary>
            /// Default Constructor.
            /// </summary>
            public DncDataPathResponse()
            {
                ServiceTunnel = false;
                Nic2Nic = false;
                Nic2VM = false;
                Vnet2VnetGW =  false;
                Vnet2VnetPeering = false;
                LastError = "";
            }

            /// <summary>
            /// Copy Constructor.
            /// </summary>
            /// <param name="response"></param>
            public DncDataPathResponse(DncDataPathResponse response)
            {
                ServiceTunnel = response.ServiceTunnel;
                Nic2Nic = response.Nic2Nic;
                Nic2VM = response.Nic2VM;
                Vnet2VnetGW = response.Vnet2VnetGW;
                Vnet2VnetPeering = response.Vnet2VnetPeering;
                LastError = response.LastError;
            }

            /// <summary>
            /// Constructor.
            /// </summary>
            /// <param name="serviceTunnel"></param>
            /// <param name="nic2Nic"></param>
            /// <param name="nic2VM"></param>
            /// <param name="vnet2VnetGW"></param>
            /// <param name="lastError"></param>
            public DncDataPathResponse(bool serviceTunnel, bool nic2Nic, bool nic2VM, bool vnet2VnetGW, bool vnet2VnetPeering, string lastError)
            {
                ServiceTunnel = serviceTunnel;
                Nic2Nic = nic2Nic;
                Nic2VM = nic2VM;
                Vnet2VnetGW = vnet2VnetGW;
                Vnet2VnetPeering = vnet2VnetPeering;
                LastError = lastError;
            }

            public bool Healthy()
            {
                return ServiceTunnel && Nic2Nic && Nic2VM && Vnet2VnetGW && Vnet2VnetPeering;
            }

            /// <summary>
            /// ToString implementation
            /// </summary>
            /// <returns>String representation of the object.</returns>
            public override string ToString()
            {
                return $"[serviceTunnel:{ServiceTunnel} nic2Nic:{Nic2Nic} nic2VM:{Nic2VM} vnet2VnetGW:{Vnet2VnetGW} vnet2VnetPeering:{Vnet2VnetPeering} lastError:{LastError}]";
            }
        }
    }
}