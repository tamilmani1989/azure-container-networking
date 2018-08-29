//-----------------------------------------------------------------------
// <copyright file="IPSubnet.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// 
//-----------------------------------------------------------------------

using Newtonsoft.Json;

namespace Cns.Data
{
    /// <summary>
    /// Defines the IP subnet class.
    /// </summary>
    public class IPSubnet
    {
        /// <summary>
        /// IPAddress associated with the subnet.
        /// </summary>
        [JsonProperty("IPAddress")]
        public string IPAddress { get; private set; }

        /// <summary>
        /// Prefix associated with the subnet.
        /// </summary>
        [JsonProperty("PrefixLength")]
        public byte PrefixLength { get; private set; }

        /// <summary>
        /// Cosntructor.
        /// </summary>
        /// <param name="ipAddress"></param>
        /// <param name="prefix"></param>
        public IPSubnet(string ipAddress, byte prefix)
        {
            this.IPAddress = ipAddress;
            this.PrefixLength = prefix;
        }

        /// <summary>
        /// ToString implementation
        /// </summary>
        /// <returns>String representation of the object.</returns>
        public override string ToString()
        {
            return $"{this.IPAddress}/{this.PrefixLength}";
        }
    }
}
