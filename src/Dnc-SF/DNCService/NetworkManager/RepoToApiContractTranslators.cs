//-----------------------------------------------------------------------
// <copyright file="RequestResetMonitor.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
//-----------------------------------------------------------------------

using DNCService;
using DNCService.API;
using NetworkManager.DataModel;
using System.Collections.Generic;
using System.Linq;
using System.Security.Cryptography;
using System.Text;

namespace NetworkManager
{
    /// <summary>
    /// Helpers to translate repo data structures to Api contract structures.
    /// </summary>
    public static class RepoToApiContractTranslators
    {
        /// <summary>
        /// Generate API allocation data from repo reservation configuration.
        /// </summary>
        /// <param name="reservationConfiguration">Supply the reservation configuration.</param>
        /// <returns>Generated IPAllocation response object.</returns>
        public static IPAllocationResponse GetIPAllocationData(
            ReservationConfiguration reservationConfiguration)
        {
            return new IPAllocationResponse(
                reservationConfiguration.AllocatedSubnet,
                reservationConfiguration.AssociatedNCID,
                reservationConfiguration.ReservationID);
        }

        /// <summary>
        /// Generate API subnet response.
        /// </summary>
        /// <param name="networkConfiguration">Supply the repo network configuration</param>
        /// <param name="subnetConfiguration">Supply the repo subnet configuration.</param>
        /// <param name="reservationConfigurations">Supply the reservation configurations.</param>
        /// <returns>Generated subnet response.</returns>
        public static SubnetResponse GetSubnetConfigurationResponse(
            NetworkConfiguration networkConfiguration,
            SubnetConfiguration subnetConfiguration,
            ICollection<ReservationConfiguration> reservationConfigurations)
        {
            IReadOnlyCollection<string> dnsServers;
            ServiceEventSource.Current.Message($"Network Config:{networkConfiguration}");
            if(networkConfiguration.DnsServers.Count > 0)
            {
                dnsServers = networkConfiguration.DnsServers;
            }
            else
            {
                dnsServers = AllocationHelper.GetDNSServerAddressesForSubnet(IPSubnet.Parse(subnetConfiguration.IPSubnet)).Select(item => item.ToString()).ToList();
            }

            List<IPAllocationResponse> ipAllocationDataSet = new List<IPAllocationResponse>();
            foreach(ReservationConfiguration reservationConfiguration in reservationConfigurations)
            {
                ipAllocationDataSet.Add(GetIPAllocationData(reservationConfiguration));
            }

            // Encrypt the access token
            string tokenHashString;
            using (var algorithm = SHA256.Create())
            {
                byte[] tokenHash = algorithm.ComputeHash(Encoding.ASCII.GetBytes(subnetConfiguration.AccessToken));
                tokenHashString = System.BitConverter.ToString(tokenHash).Replace("-", "").ToLower();
            }

            return new SubnetResponse(
                networkConfiguration.NetworkID,
                subnetConfiguration.SubnetName,
                subnetConfiguration.IPSubnet.ToString(),
                subnetConfiguration.Gateway,
                dnsServers.Select(item => item.ToString()).ToArray(),
                tokenHashString,
                ipAllocationDataSet.ToArray());
        }

        /// <summary>
        /// Translates repo network container to api network container response.
        /// </summary>
        /// <param name="networkConfiguration">Supply the repo network configuration.</param>
        /// <param name="ncConfiguration">Supply the repo network container configuration.</param>
        /// <param name="subnetConfiguration">Supply the repo subnet configuration.</param>
        /// <param name="reservationConfiguration">Supply the repo reservation configuration.</param>
        /// <param name="nodeConfiguration"Supply the repo node configuration.></param>
        /// <returns>Generated nerwork container response.</returns>
        public static NetworkContainerResponse GetNetworkContainerResponse(
            NetworkConfiguration networkConfiguration,
            NCConfiguration ncConfiguration,
            SubnetConfiguration subnetConfiguration,
            ReservationConfiguration reservationConfiguration,
            NodeConfiguration nodeConfiguration)
        {
            IReadOnlyCollection<string> dnsServers;
            if (networkConfiguration.DnsServers.Count > 0)
            {
                dnsServers = networkConfiguration.DnsServers;
            }
            else
            {
                dnsServers = AllocationHelper.GetDNSServerAddressesForSubnet(
                    IPSubnet.Parse(subnetConfiguration.IPSubnet)).Select(item => item.ToString()).ToList();
            }

            ushort vlanID = 0;
            if (nodeConfiguration != null)
            {
                nodeConfiguration.NetworkVlanIDMap.TryGetValue(ncConfiguration.NetworkID, out vlanID);
            }
            IPConfiguration ipconfiguration = new IPConfiguration(
                ncConfiguration.NetworkID,
                reservationConfiguration.AllocationRequest.SubnetName,
                subnetConfiguration.IPSubnet,
                reservationConfiguration.AllocatedSubnet,
                subnetConfiguration.Gateway,
                dnsServers.ToArray());

            RequestStatus requestStatus = new RequestStatus(
                ncConfiguration.State.ToString(),
                ncConfiguration.Status.ToString(),
                ncConfiguration.FailureDetail);

            return new NetworkContainerResponse(
                ncConfiguration.NetworkContainerID,
                vlanID,
                ipconfiguration,
                ncConfiguration.Request.AssociationInfo,
                ncConfiguration.Request.NetworkProfileID,
                requestStatus,
                ncConfiguration.LocalIP,
                ncConfiguration.Request.AllowHostToNCCommunication,
                ncConfiguration.Request.AllowNCToHostCommunication);
        }
    }
}
