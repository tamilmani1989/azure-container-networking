//--------------------------------------------------------------------------
// <copyright file="NodeRequest.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
// Definititon for NodeRequest class.
// </summary>
//--------------------------------------------------------------------------

using System;
using System.Runtime.Serialization;

namespace DNCService.API
{
    /// <summary>
    /// Class that defines a node request.
    /// </summary>
    [DataContract]
    public class NodeRequest : ICloneable
    {
        /// <summary>
        /// List of IPAddresses to connect to the node.
        /// </summary>
        [DataMember]
        public string[] IPAddresses { get; private set; }

        /// <summary>
        /// Optional. Identifies if node is onprem.
        /// </summary>
        [DataMember]
        public string LocationType { get; private set; }

        /// <summary>
        /// Type of the orchestrator that manages the node.
        /// </summary>
        [DataMember]
        public string OrchestratorType { get; private set; }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="ipAddresses">Supply the IPAddresses of the node.</param>
        /// <param name="locationType">Supply the location type.</param>
        /// <param name="orchestratorType">Supply the type of orchestrator.</param>
        public NodeRequest(string[] ipAddresses, string locationType, string orchestratorType)
        {
            if (ipAddresses == null)
            {
                throw new ArgumentNullException(nameof(ipAddresses));
            }

            if (ipAddresses.Length == 0 )
            {
                throw new ArgumentException("Cannot be empty list", nameof(ipAddresses));
            }

            if (string.IsNullOrWhiteSpace(locationType))
            {
                throw new System.ArgumentException("message", nameof(locationType));
            }

            if (string.IsNullOrWhiteSpace(orchestratorType))
            {
                throw new System.ArgumentException("message", nameof(orchestratorType));
            }

            this.IPAddresses = ipAddresses;
            this.LocationType = locationType;
            this.OrchestratorType = orchestratorType;
        }

        /// <summary>
        /// ToString implementation.
        /// </summary>
        /// <returns>String representation of the NodeRequest object.</returns>
        public override string ToString()
        {
            string ipAddresses = string.Join(",", (object[])this.IPAddresses);
            return $"Node IPAddresses:{ipAddresses} LocationType:{this.LocationType} OrchestratorType:{this.OrchestratorType}";
        }

        /// <summary>
        /// Clone implementation.
        /// </summary>
        /// <returns>Cloned node request object.</returns>
        public object Clone()
        {
            return new NodeRequest(
                this.IPAddresses, 
                this.LocationType, 
                this.OrchestratorType);
        }
    }

}
