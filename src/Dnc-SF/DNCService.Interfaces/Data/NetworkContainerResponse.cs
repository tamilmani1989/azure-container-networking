//--------------------------------------------------------------------------
// <copyright file="IPConfiguration.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
// Definititon for NetworkContainerResponse class.
// </summary>
//--------------------------------------------------------------------------

using System.Runtime.Serialization;

namespace DNCService.API
{
    /// <summary>
    /// Class definition for the network container response.
    /// </summary>
    [DataContract]
    public class NetworkContainerResponse
    {
        /// <summary>
        /// ID of the network container.
        /// </summary>
        [DataMember]
        public string ID { get; private set; }

        /// <summary>
        /// Vlan ID associated with the network container.
        /// </summary>
        [DataMember]
        public int VlanID { get; private set; }

        /// <summary>
        /// Network profile ID associated with the network container.
        /// </summary>
        [DataMember]
        public string NetworkProfileID { get; private set; }

        /// <summary>
        /// IPConfiguration associated with the network container.
        /// </summary>
        [DataMember]
        public IPConfiguration IPConfiguration { get; private set; }

        /// <summary>
        /// Association info.
        /// </summary>
        [DataMember]
        public AssociationInfo AssociationInfo { get; private set; }

        /// <summary>
        /// Status of the network container.
        /// </summary>
        [DataMember]
        public RequestStatus RequestStatus { get; private set; }

        /// <summary>
        /// Local IP used for Host to NC Communication
        /// </summary>
        [DataMember]
        public string LocalIP { get; private set; }

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
        /// Constructor.
        /// </summary>
        /// <param name="id">Supply the ID of the network container.</param>
        /// <param name="vlanID">Supply the Vlan ID.</param>
        /// <param name="ipConfiguration">Supply the IP configuration.</param>
        /// <param name="associationInfo">Supply the association info,</param>
        /// <param name="requestStatus">Supply the status of the network container.</param>
        /// <param name="localIP">Local IP used for Host to NC Communication.</param>
        /// <param name="allowHostToNCCommunication">Flag to represent if Host to NC communication is allowed.</param>
        /// <param name="allowNCToHostCommunication">Flag to represent if NC to Host communication is allowed.</param>
        public NetworkContainerResponse(
            string id,
            int vlanID,
            IPConfiguration ipConfiguration,
            AssociationInfo associationInfo,
            string networkProfileID,
            RequestStatus requestStatus,
            string localIP,
            bool allowHostToNCCommunication,
            bool allowNCToHostCommunication)
        {
            ID = id;
            VlanID = vlanID;
            IPConfiguration = ipConfiguration;
            AssociationInfo = associationInfo;
            NetworkProfileID = networkProfileID;
            RequestStatus = requestStatus;
            LocalIP = localIP;
            AllowHostToNCCommunication = allowHostToNCCommunication;
            AllowNCToHostCommunication = allowNCToHostCommunication;
        }

        /// <summary>
        /// ToString implementation.
        /// </summary>
        /// <returns>String representation of the NetworkContainerResponse object.</returns>
        public override string ToString()
        {
            return $"ID:{ID} VlanID:{VlanID} NetworkProfileID:{NetworkProfileID} IPConfiguration:{IPConfiguration}" +
                $" AssociationInfo:{this.AssociationInfo} RequestStatus:{this.RequestStatus} LocalIP: {LocalIP} " +
                $" AllowHostToNCCommunication: {AllowHostToNCCommunication}" +
                $" AllowNCToHostCommunication: {AllowNCToHostCommunication}";
        }
    }
}
