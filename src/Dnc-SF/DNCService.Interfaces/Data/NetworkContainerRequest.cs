//--------------------------------------------------------------------------
// <copyright file="NetworkContainerRequest.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
// Definititon for IP NetworkContainerRequest class.
// </summary>
//--------------------------------------------------------------------------

using System;
using System.Runtime.Serialization;

namespace DNCService.API
{
    [KnownType(typeof(NetworkContainerRequestV1))]
    [KnownType(typeof(NetworkContainerRequestV2))]
    [DataContract]
    public abstract class NetworkContainerRequest : ICloneable
    {
        /// <summary>
        /// ID of the network container.
        /// </summary>
        [DataMember]
        public string ID  { get; private set; }

        /// <summary>
        /// ID of the network profile.
        /// </summary>
        [DataMember]
        public string NetworkProfileID { get; private set; }

        /// <summary>
        /// Node to network container association information.
        /// </summary>
        [DataMember]
        public AssociationInfo AssociationInfo { get; private set; }

        /// <summary>
        /// Flag to represent if Host to NC communication is allowed.
        /// </summary>
        [DataMember]
        public bool AllowHostToNCCommunication { get; private set; }

        /// <summary>
        /// Flag to represent if NC to Host communication is allowed.
        /// </summary>
        [DataMember]
        public bool AllowNCToHostCommunication { get; private set; }

        /// <summary>
        /// Default constructor.
        /// </summary>
        public NetworkContainerRequest() { }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="associationInfo">Supply the association info.</param>
        /// <param name="networkProfileID">Supply the network profile ID.</param>
        /// <param name="allowHostToNCCommunication">Supply the flag to represent
        /// if Host to NC communication is allowed.</param>
        public NetworkContainerRequest(
            AssociationInfo associationInfo,
            string networkProfileID,
            bool allowHostToNCCommunication,
            bool allowNCToHostCommunication
            )
        {
            AssociationInfo = associationInfo;
            NetworkProfileID = networkProfileID;
            AllowHostToNCCommunication = allowHostToNCCommunication;
            AllowNCToHostCommunication = allowNCToHostCommunication;
        }

        /// <summary>
        /// ToString implementation.
        /// </summary>
        /// <returns>String representation of the NetworkContainerRequest object.</returns>
        public override string ToString()
        {
            return $"ID:{ID} networkProfileID:{NetworkProfileID} AssociationInfo:{AssociationInfo}" +
                $" AllowHostToNCCommunication: {AllowHostToNCCommunication}" +
                $" AllowNCToHostCommunication: {AllowNCToHostCommunication}";
        }

        public abstract object Clone();
    }
}
