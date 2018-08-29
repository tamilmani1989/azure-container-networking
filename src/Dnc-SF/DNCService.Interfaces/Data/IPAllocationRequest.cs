//--------------------------------------------------------------------------
// <copyright file="ErrorInfo.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
// Definititon for error info class.
// </summary>
//--------------------------------------------------------------------------

using System;
using System.Runtime.Serialization;

namespace DNCService.API
{
    [DataContract]
    public class IPAllocationRequest : ICloneable
    {
        /// <summary>
        /// Name of th subnet.
        /// </summary>
        [DataMember]
        public string SubnetName { get; private set; }

        /// <summary>
        /// Optional. Constriant to request for speicific IP.
        /// </summary>
        [DataMember]
        public string IPConstraint { get; private set; }

        /// <summary>
        /// Optional. Indicates that the allocated IP will be used on a specific node.
        /// </summary>
        [DataMember]
        public string NodeConstraint { get; private set; }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="subnetName">Supply the subnet name.</param>
        /// <param name="ipConstraint">Supply the IP allocation constraint.</param>
        /// <param name="nodeConstraint">Supply the node constraint.</param>
        public IPAllocationRequest(string subnetName, string ipConstraint, string nodeConstraint)
        {
            if (string.IsNullOrWhiteSpace(subnetName))
            {
                throw new ArgumentException("Subnet name cannot be null or empty", nameof(subnetName));
            }

            this.SubnetName = subnetName;
            this.IPConstraint = ipConstraint;
            this.NodeConstraint = nodeConstraint;
        }

        /// <summary>
        /// ToString implementation.
        /// </summary>
        /// <returns>String representation of the IPAllocationRequest object.</returns>
        public override string ToString()
        {
            return $"[SubnetName:{this.SubnetName} IPConstraint:{this.IPConstraint} NodeConstraint{this.NodeConstraint}]";
        }

        /// <summary>
        /// Clone implementation.
        /// </summary>
        /// <returns>Cloned object.</returns>
        public object Clone()
        {
            return new IPAllocationRequest(this.SubnetName, this.IPConstraint, this.NodeConstraint);
        }
    }
}
