//--------------------------------------------------------------------------
// <copyright file="AssociationInfo.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
// Class that defines AssociationInfo 
// </summary>
//--------------------------------------------------------------------------

using API;
using Newtonsoft.Json;
using Newtonsoft.Json.Linq;
using System;
using System.Linq;
using System.Runtime.Serialization;
using System.Text;

namespace DNCService.API
{
    /// <summary>
    /// Definition for association info.
    /// </summary>
    [DataContract]
    public class AssociationInfo : ICloneable
    {
        /// <summary>
        /// Id of the associated node.
        /// </summary>
        [DataMember]
        public string NodeID { get; private set; }

        /// <summary>
        /// Id of the interface to which the network container will be associated.
        /// </summary>
        [DataMember]
        public string InterfaceID { get; private set; }

        /// <summary>
        /// Type of the container to be associated.
        /// </summary>
        [DataMember]
        public string ContainerType { get; private set; }

        /// <summary>
        /// Context associated with the network container. 
        /// </summary>
        [DataMember]
        [JsonConverter(typeof(JSonRawConverter))]

        public string OrchestratorContext { get;  set; }
        
        /// <summary>
        /// Default constructor.
        /// </summary>
        public AssociationInfo() { }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="nodeID">Supply the Id of the node.</param>
        /// <param name="interfaceID">Supply the interface Id.</param>
        /// <param name="containerType">Supply the container type.</param>
        /// <param name="orchestratorContext">Supply the orchestrator context.</param>
        public AssociationInfo(
            string nodeID, 
            string interfaceID, 
            string containerType, 
            string orchestratorContext)
        {
            this.NodeID = nodeID;
            this.InterfaceID = interfaceID;
            this.ContainerType = containerType;
            this.OrchestratorContext = orchestratorContext;
        }

        /// <summary>
        /// ToString implementation.
        /// </summary>
        /// <returns>String representation of the association info.</returns>
        public override string ToString()
        {
            string orchestratorContext = this.OrchestratorContext.ToString();
            return $"NodeID:{this.NodeID} InterfaceID:{this.InterfaceID} ContainerType:{this.ContainerType} " +
                $"OrchestratorContext:{orchestratorContext}";
        }

        /// <summary>
        /// Clone implementation.
        /// </summary>
        /// <returns>Cloned object.</returns>
        public object Clone()
        {
            return new AssociationInfo(this.NodeID, this.InterfaceID, this.ContainerType, this.OrchestratorContext);
        }
    }
}
