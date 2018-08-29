//-----------------------------------------------------------------------
// <copyright file="DncNetworkManagementException.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// 
//-----------------------------------------------------------------------
using System;
using System.Runtime.Serialization;

namespace DNCService.API
{
    /// <summary>
    /// Class defintition for DncManagementException.
    /// </summary>
    [Serializable]
    [DataContract]
    public class DncNetworkManagementException : DncManagementException, ISerializable
    {
        /// <summary>
        /// ID of the network.
        /// </summary>
        [DataMember]
        public string NetworkID { get; private set; }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="networkID">Supply the ID of the network.</param>
        /// <param name="errorInfo">Supply the error Info to be associated with th exception.</param>
        public DncNetworkManagementException(string networkID, ErrorInfo errorInfo) : base (errorInfo)
        {
            this.NetworkID = networkID;
        }

        /// <summary>
        /// ToString implementation.
        /// </summary>
        /// <returns>ToString representation of the DncNetworkManagementException object.</returns>
        public override string ToString()
        {
            return $"Network:{this.NetworkID} Error:{base.ToString()}";
        }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="info">Supply the serialization info.</param>
        /// <param name="context">Supply the streaming context.</param>
        protected DncNetworkManagementException(SerializationInfo info, StreamingContext context)
                   : base(info, context)
        {
            this.NetworkID = info.GetString(nameof(this.NetworkID));
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

            info.AddValue(nameof(this.NetworkID), this.NetworkID);

            // MUST call through to the base class to let it save its own state
            base.GetObjectData(info, context);
        }
    }
}
