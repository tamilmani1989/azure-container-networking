//--------------------------------------------------------------------------
// <copyright file="IPConfiguration.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
// Definititon for IP configuration.
// </summary>
//--------------------------------------------------------------------------

using System;
using System.Runtime.Serialization;

namespace DNCService.API
{
    /// <summary>
    /// NetworkRequest definition.
    /// </summary>
    [DataContract]
    public class NetworkRequest : ICloneable
    {
        /// <summary>
        /// AzureNet const string.
        /// </summary>
        public const string AzureNet = "AzureNet";

        /// <summary>
        /// Type of network.
        /// </summary>
        [DataMember]
        public string NetworkType { get; private set; }

        /// <summary>
        /// Network space to be associated with on prem networks.
        /// </summary>
        [DataMember(IsRequired =false)]        
        public string NetworkSpace { get; private set; }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="networkType">Supply the network type.</param>
        /// <param name="networkSpace">Supply the network space.</param>
        public NetworkRequest(string networkType, string networkSpace)
        {
            if (string.IsNullOrWhiteSpace(networkType))
            {
                throw new ArgumentException("Cannot be null or empty", nameof(networkType));
            }

            if(string.Compare(networkType, AzureNet, StringComparison.CurrentCultureIgnoreCase) !=0)
            {
                throw new ArgumentException($"Only {AzureNet} network type is supported");
            }

            this.NetworkType = networkType;
            this.NetworkSpace = networkSpace;
        }

        /// <summary>
        /// ToString implementation.
        /// </summary>
        /// <returns>String representation of the network request.</returns>
        public override string ToString()
        {
            return $"NetworkType:{this.NetworkType} NetworkSpace:{this.NetworkSpace}";
        }

        /// <summary>
        /// Clone implementation.
        /// </summary>
        /// <returns>Cloned network request object.</returns>
        public object Clone()
        {
            return new NetworkRequest(this.NetworkType, this.NetworkSpace);
        }
    }
}
