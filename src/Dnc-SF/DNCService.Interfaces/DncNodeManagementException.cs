//-----------------------------------------------------------------------
// <copyright file="DncNodeManagementException.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// 
//-----------------------------------------------------------------------
using System;
using System.Runtime.Serialization;

namespace DNCService.API
{
    /// <summary>
    /// Class definition for DncNodeManagementException.
    /// </summary>
    [DataContract]
    [Serializable]
    public class DncNodeManagementException : DncManagementException, ISerializable
    {
        /// <summary>
        /// ID of the node.
        /// </summary>
        [DataMember]
        public string NodeID { get; private set; }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="nodeID">Supply the ID of the node.</param>
        /// <param name="errorInfo">Supply the error Info to be associated with th exception.</param>
        public DncNodeManagementException(string nodeID, ErrorInfo errorInfo) : base(errorInfo)
        {
            this.NodeID = nodeID;
        }

        /// <summary>
        /// ToString implementation.
        /// </summary>
        /// <returns>ToString representation of the DncNodeManagementException object.</returns>
        public override string ToString()
        {
            return $"Node:{this.NodeID} Error:{base.ToString()}";
        }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="info">Supply the serialization info.</param>
        /// <param name="context">Supply the streaming context.</param>
        protected DncNodeManagementException(SerializationInfo info, StreamingContext context)
                   : base(info, context)
        {
            this.NodeID = info.GetString(nameof(this.NodeID));
        }

        /// Implements the get object method used for serialization.
        /// </summary>
        /// <param name="info">Supply the serialization info.</param>
        /// <param name="context">Supply the streaming context.</param>
        public override void GetObjectData(SerializationInfo info, StreamingContext context)
        {
            if (info == null)
            {
                throw new ArgumentNullException("info");
            }

            info.AddValue(nameof(this.NodeID), this.NodeID);

            // MUST call through to the base class to let it save its own state
            base.GetObjectData(info, context);
        }
    }
}
