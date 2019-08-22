//--------------------------------------------------------------------------
// <copyright file="NetworkContainerRequestV2.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
// Definititon for NetworkContainerRequestV2 class.
// </summary>
//--------------------------------------------------------------------------

using System.Runtime.Serialization;

namespace DNCService.API
{
    /// <summary>
    /// Class defintition for network container request.
    /// </summary>
    [DataContract]
    public class NetworkContainerRequestV2 : NetworkContainerRequest
    {
        /// <summary>
        /// The allocation constraint associated with the request.
        /// </summary>
        [DataMember]
        public IPAllocationRequest AllocationRequest { get; private set; }

        /// <summary>
        /// Default constructor.
        /// </summary>
        public NetworkContainerRequestV2() { }

        /// <summary>
        /// Construcotr.
        /// </summary>
        /// <param name="allocationRequest">Supply the allocation constraint.</param>
        /// <param name="associationInfo">Supply the association info.</param>
        /// <param name="allowHostToNCCommunication"> Supply the flag to represent
        /// if Host to NC communication is allowed.</param>
        /// <param name="allowNCToHostCommunication"> Supply the flag to represent
        /// if NC to Host communication is allowed.</param>
        public NetworkContainerRequestV2(
            string networkProfileID,
            IPAllocationRequest allocationRequest,
            AssociationInfo associationInfo,
            bool allowHostToNCCommunication,
            bool allowNCToHostCommunication) : base(associationInfo, networkProfileID, allowHostToNCCommunication, allowNCToHostCommunication)
        {
            this.AllocationRequest = allocationRequest;
        }

        /// <summary>
        /// Clone implementation.
        /// </summary>
        /// <returns>Cloned network request object.</returns>
        public override object Clone()
        {
            return new NetworkContainerRequestV2(
                NetworkProfileID, AllocationRequest, AssociationInfo, AllowHostToNCCommunication, AllowNCToHostCommunication);
        }

        /// <summary>
        /// ToString implementation.
        /// </summary>
        /// <returns>String representation of the NetworkContainerRequestV2 object.</returns>
        public override string ToString()
        {
            return $"[{base.ToString()} IPAllocationRequest:{AllocationRequest}]";
        }
    }
}
