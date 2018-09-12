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
        /// Constructor.
        /// </summary>
        /// <param name="id">Supply the ID of the network container.</param>
        /// <param name="vlanID">Supply the Vlan ID.</param>
        /// <param name="ipConfiguration">Supply the IP configuration.</param>
        /// <param name="associationInfo">Supply the association info,</param>
        /// <param name="requestStatus">Supply the status of the network container.</param>
        public NetworkContainerResponse(
            string id,
            int vlanID,
            IPConfiguration ipConfiguration, 
            AssociationInfo associationInfo, 
            string networkProfileID,
            RequestStatus requestStatus)
        {
            this.ID = id;
            this.VlanID = vlanID;
            this.IPConfiguration = ipConfiguration;
            this.AssociationInfo = associationInfo;
            this.NetworkProfileID = networkProfileID;
            this.RequestStatus = requestStatus;
        }

        /// <summary>
        /// ToString implementation.
        /// </summary>
        /// <returns>String representation of the NetworkContainerResponse object.</returns>
        public override string ToString()
        {
            return $"ID:{this.ID} VlanID:{this.VlanID} NetworkProfileID:{this.NetworkProfileID} " +
                $"IPConfiguration:{this.IPConfiguration} AssociationInfo:{this.AssociationInfo} " +
                $"RequestStatus:{this.RequestStatus}";
        }
    }
}
