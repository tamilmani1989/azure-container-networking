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
            /// Nc2Nc
            /// </summary>
            [JsonProperty("Nc2Nc")]
            public bool Nc2Nc { get; private set; }

            /// <summary>
            /// Nc2VM
            /// </summary>
            [JsonProperty("Nc2VM")]
            public bool Nc2VM { get; private set; }

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
            /// NonRFCVnet2VnetPeering
            /// </summary>
            [JsonProperty("NonRFCVnet2VnetPeering")]
            public bool NonRFCVnet2VnetPeering { get; private set; }

            /// <summary>
            /// NSG
            /// </summary>
            [JsonProperty("NSG")]
            public bool NSG { get; private set; }

            /// <summary>
            /// NSGwTags
            /// </summary>
            [JsonProperty("NSGwTags")]
            public bool NSGwTags { get; private set; }

            /// <summary>
            /// UDR
            /// </summary>
            [JsonProperty("UDR")]
            public bool UDR { get; private set; }

            /// <summary>
            /// PrivateLink
            /// </summary>
            [JsonProperty("PrivateLink")]
            public bool PrivateLink { get; private set; }

            /// <summary>
            /// ExternalDNS
            /// </summary>
            [JsonProperty("ExternalDNS")]
            public bool ExternalDNS { get; private set; }

            /// <summary>
            /// PrivateDNS
            /// </summary>
            [JsonProperty("PrivateDNS")]
            public bool PrivateDNS { get; private set; }

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
                Nc2Nc = false;
                Nc2VM = false;
                Vnet2VnetGW =  false;
                Vnet2VnetPeering = false;
                NonRFCVnet2VnetPeering = false;
                NSG = false;
                NSGwTags = false;
                UDR = false;
                PrivateLink = false;
                ExternalDNS = false;
                PrivateDNS = false;
                LastError = "";
            }

            /// <summary>
            /// Copy Constructor.
            /// </summary>
            /// <param name="response"></param>
            public DncDataPathResponse(DncDataPathResponse response)
            {
                ServiceTunnel = response.ServiceTunnel;
                Nc2Nc = response.Nc2Nc;
                Nc2VM = response.Nc2VM;
                Vnet2VnetGW = response.Vnet2VnetGW;
                Vnet2VnetPeering = response.Vnet2VnetPeering;
                NonRFCVnet2VnetPeering = response.NonRFCVnet2VnetPeering;
                NSG = response.NSG;
                NSGwTags = response.NSGwTags;
                UDR = response.UDR;
                PrivateLink = response.PrivateLink;
                ExternalDNS = response.ExternalDNS;
                PrivateDNS = response.PrivateDNS;
                LastError = response.LastError;
            }

            /// <summary>
            /// Constructor.
            /// </summary>
            /// <param name="serviceTunnel"></param>
            /// <param name="nc2Nc"></param>
            /// <param name="nc2VM"></param>
            /// <param name="vnet2VnetGW"></param>
            /// <param name="vnet2VnetPeering"></param>
            /// <param name="nonRfcVnet2VnetPeering"></param>
            /// <param name="nsg"></param>
            /// <param name="nsgwTags"></param>
            /// <param name="udr"></param>
            /// <param name="privateLink"></param>
            /// <param name="externalDNS"></param>
            /// <param name="privateDNS"></param>
            /// <param name="lastError"></param>
            public DncDataPathResponse(bool serviceTunnel, bool nc2Nc, bool nc2VM, bool vnet2VnetGW, bool vnet2VnetPeering, bool nonRfcVnet2VnetPeering, bool nsg, bool nsgwTags, bool udr, bool privateLink, bool externalDNS, bool privateDNS, string lastError)
            {
                ServiceTunnel = serviceTunnel;
                Nc2Nc = nc2Nc;
                Nc2VM = nc2VM;
                Vnet2VnetGW = vnet2VnetGW;
                Vnet2VnetPeering = vnet2VnetPeering;
                NonRFCVnet2VnetPeering = nonRfcVnet2VnetPeering;
                NSG = nsg;
                NSGwTags = nsgwTags;
                UDR = udr;
                PrivateLink = privateLink;
                ExternalDNS = externalDNS;
                PrivateDNS = privateDNS;
                LastError = lastError;
            }

            public bool Healthy()
            {
                return ServiceTunnel && Nc2Nc && Nc2VM && Vnet2VnetGW && Vnet2VnetPeering && NonRFCVnet2VnetPeering && NSG && NSGwTags && UDR && PrivateLink && ExternalDNS && PrivateDNS;
            }

            /// <summary>
            /// ToString implementation
            /// </summary>
            /// <returns>String representation of the object.</returns>
            public override string ToString()
            {
                return $"[serviceTunnel:{ServiceTunnel} Nc2Nc:{Nc2Nc} Nc2VM:{Nc2VM} vnet2VnetGW:{Vnet2VnetGW} vnet2VnetPeering:{Vnet2VnetPeering} nonRfcVnet2VnetPeering:{NonRFCVnet2VnetPeering} nsg:{NSG} nsgwTags:{NSGwTags} udr:{UDR} privateLink:{PrivateLink} externalDNS:{ExternalDNS} privateDNS:{PrivateDNS} lastError:{LastError}]";
            }
        }
    }
}