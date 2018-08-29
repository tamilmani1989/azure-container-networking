//-----------------------------------------------------------------------
// <copyright file="HostAgentClient.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// 
//-----------------------------------------------------------------------
using DNCService;
using NetworkManager;
using Newtonsoft.Json;
using NMAgent.Data;
using System;
using System.Collections.Generic;
using System.Net;
using System.Net.Http;
using System.Text;
using System.Threading.Tasks;

namespace NMAgent
{
    /// <summary>
    /// Implements the host agent client.
    /// </summary>
    public class HostAgentClient
    {
        /// <summary>
        /// Handle to the DNC configuration.
        /// </summary>
        public DncConfiguration DncConfiguration { get; private set; }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="dncConfiguration">Supply the Dnc configuration.</param>
        public HostAgentClient(DncConfiguration dncConfiguration)
        {
            this.DncConfiguration = dncConfiguration;
        }

        /// <summary>
        /// Joins and get network configuration for host agent.
        /// </summary>
        /// <param name="networkID">Supply the ID of the network.</param>
        /// <returns>Retrieved virtual network response.</returns>
        public async Task<VirtualNetworkResponse> JoinAndGetNetwork(
            string networkID)
        {
            if (DncConfiguration.SimulatorMode)
            {
                List<Subnet> subnets = new List<Subnet>();
                Subnet subnet = new Subnet("FrontEndSubnet", "10.0.1.0/24", null, "10.0.1.1");
                subnets.Add(subnet);
                VirtualNetworkResponse response = new VirtualNetworkResponse(
                    "200", 
                    "1",
                    "10.0.0.0/8",
                    "10.0.0.0/16, 10.1.0.0/16, 10.2.0.0/16",
                    subnets.ToArray(),
                    "8.8.8.8,9.9.9.9");
                return response;
            }
            string joinNetworkUrl = string.Format(
                    DncConfiguration.HostAgentSettings.ConnectionStrings.JoinNetworkURLFmt,
                    DncConfiguration.HostAgentSettings.WireServerIP,
                    networkID);
            string getNetworkUrl = string.Format(
                DncConfiguration.HostAgentSettings.ConnectionStrings.GetNetworkConfigurationURLFmt,
                DncConfiguration.HostAgentSettings.WireServerIP,
                networkID);

            ServiceEventSource.Current.Message("JoinNetworkUrl:{0}", joinNetworkUrl);

            // Empty body is required as wireserver cannot handle a post without the body.
            var httpContent = new StringContent("  ", Encoding.UTF8, "application/json");
            HttpClient httpClient = new HttpClient();
            HttpResponseMessage joinResponse = await httpClient.PostAsync(joinNetworkUrl, httpContent);
            joinResponse.EnsureSuccessStatusCode();

            HttpResponseMessage getResponse = await httpClient.GetAsync(getNetworkUrl);
            getResponse.EnsureSuccessStatusCode();

            string responseBody = await getResponse.Content.ReadAsStringAsync();
            VirtualNetworkResponse virtualNetworkResponse =
                JsonConvert.DeserializeObject<VirtualNetworkResponse>(responseBody);
            HttpStatusCode getNetworkResponseCode = (HttpStatusCode)Enum.Parse(
                typeof(HttpStatusCode),
                virtualNetworkResponse.ResponseCode);
            if (getNetworkResponseCode != HttpStatusCode.OK)
            {
                throw new Exception(
                    $"Invalid response code . Received response:{virtualNetworkResponse}");
            }

            ServiceEventSource.Current.Message($"Retrieved virtual network response:{virtualNetworkResponse}");
            return virtualNetworkResponse;
        }

        /// <summary>
        /// Create network container request.
        /// </summary>
        /// <param name="networkContainerInfo">Supply the network container info.</param>
        /// <returns></returns>
        public async Task CreateNetworkContainer(NetworkContainerInfo networkContainerInfo)
        {
            if (DncConfiguration.SimulatorMode)
            {
                return;
            }
            string createNCUrl = string.Format(
                    DncConfiguration.HostAgentSettings.ConnectionStrings.PutNetworkContainerURLFmt,
                    DncConfiguration.HostAgentSettings.WireServerIP,
                    networkContainerInfo.NodeInterfaceID,
                    networkContainerInfo.NetworkContainerID,
                    networkContainerInfo.AccessToken);

            ServiceEventSource.Current.Message("createNCUrl:{0}", createNCUrl);

            CreateOrUpdateNetworkContainerRequest createOrUpdateNetworkContainerRequest =
                networkContainerInfo.GetCreateNMAgentNetworkContainerRequest();
            string createOrUpdateNCJson =
                JsonConvert.SerializeObject(createOrUpdateNetworkContainerRequest);
            var httpContent = new StringContent(createOrUpdateNCJson, Encoding.UTF8, "application/json");
            HttpClient httpClient = new HttpClient();
            HttpResponseMessage response = await httpClient.PostAsync(createNCUrl, httpContent);
            response.EnsureSuccessStatusCode();

            string responseBody = await response.Content.ReadAsStringAsync();
            ServiceEventSource.Current.Message($"responseBody:{responseBody}");
            CreateNetworkContainerResponse createNCReponse = null;
            HttpStatusCode createNCReponseCode;
            try
            {
                createNCReponse = JsonConvert.DeserializeObject<CreateNetworkContainerResponse>(
                    responseBody);
                createNCReponseCode = (HttpStatusCode)Enum.Parse(
                    typeof(HttpStatusCode),
                    createNCReponse.ResponseCode);
            }
            catch(Exception exception)
            {
                throw new Exception(
                    $"Failed to decode response {createNCReponse} due to exception {exception}");
            }

            if (createNCReponseCode != HttpStatusCode.OK)
            {
                throw new Exception(
                    $"Create NC failed with response code {createNCReponseCode}. " +
                    $"Received response:{createNCReponse}");
            }

            ServiceEventSource.Current.Message(
                $"Successfully published NC:{networkContainerInfo.NetworkContainerID} " +
                $"Received response:{createNCReponse}");
        }

        /// <summary>
        /// Deletes network container.
        /// </summary>
        /// <param name="networkContainerInfo">Supply thenetwork container to be deleted.</param>
        /// <returns></returns>
        public async Task DeleteNetworkContainer(NetworkContainerInfo networkContainerInfo)
        {
            string deleteNCUrl = string.Format(
                     DncConfiguration.HostAgentSettings.ConnectionStrings.DeleteNetworkContainerURLFmt,
                     DncConfiguration.HostAgentSettings.WireServerIP,
                     networkContainerInfo.NodeInterfaceID,
                     networkContainerInfo.NetworkContainerID,
                     networkContainerInfo.AccessToken);

            ServiceEventSource.Current.Message("DeleteNCUrl:{0}", deleteNCUrl);

            var httpContent = new StringContent(" ", Encoding.UTF8, "application/json");
            HttpClient httpClient = new HttpClient();
            HttpResponseMessage response = await httpClient.PostAsync(deleteNCUrl, httpContent);
            response.EnsureSuccessStatusCode();

            string responseBody = await response.Content.ReadAsStringAsync();
            DeleteNetworkContainerResponse deleteNCResponse = null;
            HttpStatusCode deleteNCReponseCode;
            try
            {
                deleteNCResponse = JsonConvert.DeserializeObject<DeleteNetworkContainerResponse>(
                    responseBody);
                deleteNCReponseCode = (HttpStatusCode)Enum.Parse(
                    typeof(HttpStatusCode),
                    deleteNCResponse.ResponseCode);
            }
            catch (Exception exception)
            {
                throw new Exception(
                    $"Failed to decode response {deleteNCResponse} due to exception {exception}");
            }

            if (deleteNCReponseCode != HttpStatusCode.OK)
            {
                throw new Exception(
                    $"Delete NC failed with response code {deleteNCReponseCode}. " +
                    $"Received response:{deleteNCResponse}");
            }

            ServiceEventSource.Current.Message(
                $"Successfully unpublished NC:{networkContainerInfo.NetworkContainerID} " +
                $"Received response:{deleteNCResponse}");
        }
    }
}
