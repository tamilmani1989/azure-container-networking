//--------------------------------------------------------------------------
// <copyright file="DNCService.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
// Defines the contracts exposed by Dnc service.
// </summary>
//--------------------------------------------------------------------------

namespace DNCService.Interfaces
{
    using System.Threading.Tasks;
    using DNCService.API;
    using Microsoft.ServiceFabric.Services.Remoting;

    /// <summary>
    /// This interface defines the methods exposed by DNC service.
    /// Clients use this interface to interact with the service that implements it.
    /// </summary>
    public interface IDNCService : IService
    {
            /// <summary>
        /// Creates or updates an existing network.
        /// </summary>
        /// <param name="networkID">Supply the ID of the network.</param>
        /// <param name="networkRequest">Supply the network request.</param>
        /// <returns></returns>
        Task CreateOrUpdateNetworkAsync(string networkID, NetworkRequest networkRequest, string apiVersion);

        /// <summary>
        /// Deletes a network.
        /// </summary>
        /// <param name="networkID">Supply the ID of the network.</param>
        /// <returns></returns>
        Task DeleteNetworkAsync(string networkID);


        /// <summary>
        /// Creates or updates an existing node.
        /// </summary>
        /// <param name="networkID">Supply the ID of the node.</param>
        /// <param name="nodeRequest">Supply the node request.</param>
        /// <returns></returns>
        Task CreateOrUpdateNodeAsync(string nodeID, NodeRequest nodeRequest, string apiVersion);

        /// <summary>
        /// Delete node.
        /// </summary>
        /// <param name="nodeID">Supply the id of the node to be deleted.</param>
        /// <returns></returns>
        Task DeleteNodeAsync(string nodeID);

        Task CreateOrUpdateNetworkContainerAsync(
            string networkID, 
            string networkContainerID,
            NetworkContainerRequest ncRequest,
            string apiVersion);

        /// <summary>
        /// Delete network container.
        /// </summary>
        /// <param name="networkID">Supply the ID of the network.</param>
        /// <param name="networkContainerID">Supply the network container ID.</param>
        /// <returns></returns>
        Task DeleteNetworkContainerAsync(
            string networkID,
            string networkContainerID);

        /// <summary>
        /// Retrieve the subnet configuration.
        /// </summary>
        /// <param name="networkID">Supply the ID of the network.</param>
        /// <param name="subnetName">Supply the name of the subnet.</param>
        /// <returns></returns>
        Task<SubnetResponse> GetSubnetConfigurationResponseAsync(
            string networkID, 
            string subnetName);

        /// <summary>
        /// Retrieve the network container.
        /// </summary>
        /// <param name="networkID">Supply the ID of the network.</param>
        /// <param name="subnetName">Supply the subnet name.</param>
        /// <returns>Retrieved subnet configuration.</returns>
        Task<NetworkContainerResponse> GetNetworkContainerResponseAsync(
            string networkID,
            string subnetName);

        /// <summary>
        /// Retrieve the node information.
        /// </summary>
        /// <param name="nodeID">Supply the ID of the node.</param>
        /// <returns>Retrieved node configuration.</returns>
        Task<NodeResponse> GetNodeInfoResponseAsync(
            string nodeID);

        /// <summary>
        /// Retrieve the network status.
        /// </summary>
        /// <param name="networkID">Supply the ID of the network.</param>
        /// <returns>Network status.</returns>
        Task<RequestStatus> GetNetworkStatusResponseAsync(
            string networkID);

        /// <summary>
        /// Retrieve the subnet status.
        /// </summary>
        /// <param name="networkID">Supply the ID of the network.</param>
        /// <param name="subnetName">Supply the subnet name.</param>
        /// <returns>Subnet status.</returns>
        Task<RequestStatus> GetSubnetStatusResponseAsync(
            string networkID,
            string subnetName);

        /// <summary>
        /// Retrieve the network container status.
        /// </summary>
        /// <param name="networkID">Supply the ID of the network.</param>
        /// <param name="networkContainerID">Supply the network container ID.</param>
        /// <returns>Network container status.</returns>
        Task<RequestStatus> GetNetworkContainerStatusResponseAsync(
            string networkID,
            string networkContainerID);

        /// <summary>
        /// Join subnet.
        /// </summary>
        /// <param name="networkID">Supply the ID of the network.</param>
        /// <param name="subnetName">Supply the subnet name.</param>
        /// <param name="subnetRequest">Supply the subnet request.</param>
        /// <param name="apiVersion">Supply the API version.</param>
        /// <returns></returns>
        Task JoinSubnet(
            string networkID, 
            string subnetName,
            SubnetRequest subnetRequest, 
            string apiVersion);

        /// <summary>
        /// Add a network profile ID.
        /// </summary>
        /// <param name="networkProfileID">Supply the ID of the network profile.</param>
        /// <returns></returns>
        Task AddNetworkProfileAsync(
            string networkProfileID);

        /// <summary>
        /// Delete a network profile ID.
        /// </summary>
        /// <param name="networkProfileID">Supply the ID of the network profile.</param>
        /// <returns></returns>
        Task DeleteNetworkProfileAsync(
            string networkProfileID);
    }
}
