//-----------------------------------------------------------------------
// <copyright file="Route.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// 
//-----------------------------------------------------------------------

using Newtonsoft.Json;

namespace Cns.Data
{
    /// <summary>
    /// Defines the route class.
    /// </summary>
    public class Route
    {
        /// <summary>
        /// Destination IPAddress subnet
        /// </summary>
        [JsonProperty("IPAddress")]
        public string IPAddress { get; private set; }

        /// <summary>
        /// Gateway to use for routing.
        /// </summary>
        [JsonProperty("GatewayIPAddress")]
        public string GatewayIPAddress { get; private set; }

        /// <summary>
        /// Id of the interface to use.
        /// </summary>
        [JsonProperty("InterfaceToUse")]
        public string InterfaceToUse { get; private set; }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="ipAddress"></param>
        /// <param name="gatewayIPAddress"></param>
        /// <param name="interfaceID"></param>
        public Route(string ipAddress, string gatewayIPAddress, string interfaceID)
        {
            this.IPAddress = ipAddress;
            this.GatewayIPAddress = gatewayIPAddress;
            this.InterfaceToUse = interfaceID;
        }

        /// <summary>
        /// ToString implementation
        /// </summary>
        /// <returns>String representation of the object.</returns>
        public override string ToString()
        {
            return $"[IPAddress:{this.IPAddress} GatewayIPAddress:{this.GatewayIPAddress} " +
                $"InterfaceToUse:{this.InterfaceToUse}]";
        }
    }
}
