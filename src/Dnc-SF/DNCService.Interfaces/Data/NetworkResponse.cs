//--------------------------------------------------------------------------
// <copyright file="NetworkResponse.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
// Class that defines a NetworkResponse 
// </summary>
//--------------------------------------------------------------------------

using System;
using System.Collections.Generic;
using System.Linq;
using System.Runtime.Serialization;

namespace DNCService.API
{
    /// <summary>
    /// NetworkResponse definition.
    /// </summary>
    [DataContract]
    public class NetworkResponse : ICloneable
    {
        /// <summary>
        /// Type of network.
        /// </summary>
        [DataMember]
        public string NetworkType { get; private set; }

        /// <summary>
        /// Network space associated with the network.
        /// </summary>
        [DataMember]
        public string NetworkSpace { get; private set; }


        /// <summary>
        /// Network space associated with the network.
        /// </summary>
        [DataMember]
        public SubnetResponse[] SubnetConfigurations { get; private set; }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="networkType">Supply the network type.</param>
        /// <param name="networkSpace">Supply the network space.</param>
        /// <param name="networkSpace"
        public NetworkResponse(string networkType, string networkSpace, IEnumerable<SubnetResponse> subnetConfigurations)
        {
            if (string.IsNullOrWhiteSpace(networkType))
            {
                throw new ArgumentException("Cannot be null or empty", nameof(networkType));
            }

            if (string.IsNullOrWhiteSpace(networkSpace))
            {
                throw new ArgumentException("Cannot be null or empty", nameof(networkSpace));
            }

            if (subnetConfigurations == null)
            {
                SubnetConfigurations = new SubnetResponse[] { };
            }
            else
            {
                this.SubnetConfigurations = subnetConfigurations.ToArray();
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
            return $"NetworkType:{this.NetworkType} NetworkSpace:{this.NetworkSpace} SubnetConfigurations:{string.Join(",", (object[])SubnetConfigurations)}";
        }

        /// <summary>
        /// Clone implementation.
        /// </summary>
        /// <returns>Cloned network request object.</returns>
        public object Clone()
        {
            return new NetworkResponse(this.NetworkType, this.NetworkSpace, this.SubnetConfigurations);
        }
    }
}
