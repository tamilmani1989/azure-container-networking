//--------------------------------------------------------------------------
// <copyright file="NodeInfoRequest.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
// Definition for NodeInfoRequest class.
// </summary>
//--------------------------------------------------------------------------

using System;
using System.Runtime.Serialization;

namespace DNCService.API
{
    /// <summary>
    /// Class that defines a node info request.
    /// </summary>
    [DataContract]
    public class NodeInfoRequest : ICloneable
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
        public NodeInfoRequest(
            string[] ipAddresses,
            string locationType,
            string orchestratorType)
        {
            if (ipAddresses == null)
            {
                throw new ArgumentNullException(nameof(ipAddresses));
            }

            if (ipAddresses.Length == 0 )
            {
                throw new ArgumentException("IPAddress list be empty list", nameof(ipAddresses));
            }

            if (string.IsNullOrWhiteSpace(orchestratorType))
            {
                throw new ArgumentException("Orchestrator type cannot be empty", nameof(orchestratorType));
            }

            IPAddresses = ipAddresses;
            LocationType = locationType;
            OrchestratorType = orchestratorType;
        }

        /// <summary>
        /// Clone implementation.
        /// </summary>
        /// <returns>Cloned node request object.</returns>
        public object Clone()
        {
            return new NodeInfoRequest(
                IPAddresses,
                LocationType,
                OrchestratorType);
        }
    }

}
