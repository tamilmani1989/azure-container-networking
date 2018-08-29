//-----------------------------------------------------------------------
// <copyright file="DncNetworkContainerException.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// 
//-----------------------------------------------------------------------

using System;
using System.Runtime.Serialization;

namespace DNCService.API
{
    /// <summary>
    /// Class defintition for Network container exception.
    /// </summary>
    [DataContract]
    [Serializable]
    public class DncNetworkContainerException : DncManagementException, ISerializable
    {
        /// <summary>
        /// Id of the network.
        /// </summary>
        [DataMember]
        public string NetworkID { get; private set; }

        /// <summary>
        /// Id of the network container.
        /// </summary>
        [DataMember]
        public string NetworkContainerID { get; private set; }

        /// <summary>
        /// Id of the node.
        /// </summary>
        [DataMember]
        public string NodeID { get; private set; }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="networkID">Supply the ID of the network.</param>
        /// <param name="networkContainerID">Supply the network container ID.</param>
        /// <param name="errorInfo">Supply the error Info to be associated with th exception.</param>
        public DncNetworkContainerException(string networkID, string networkContainerID, ErrorInfo errorInfo) : base(errorInfo)
        {
            this.NetworkID = networkID;
            this.NetworkContainerID = networkContainerID;
        }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="info">Supply the serialization info.</param>
        /// <param name="context">Supply the streaming context.</param>
        protected DncNetworkContainerException(SerializationInfo info, StreamingContext context)
                           : base(info, context)
        {
            this.NetworkID = info.GetString(nameof(this.NetworkID));
            this.NetworkContainerID = info.GetString(nameof(this.NetworkContainerID));
        }

        /// <summary>
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

            info.AddValue(nameof(this.NetworkID), this.NetworkID);
            info.AddValue(nameof(this.NetworkContainerID), this.NetworkContainerID);

            // MUST call through to the base class to let it save its own state
            base.GetObjectData(info, context);
        }

        /// <summary>
        /// ToString implementation.
        /// </summary>
        /// <returns>ToString representation of the DncNetworkContainerException object.</returns>
        public override string ToString()
        {
            return $"Network:{this.NetworkID} NetworkcontainerID:{this.NetworkContainerID} " +
                $"Error:{base.ToString()}";
        }
    }
}
