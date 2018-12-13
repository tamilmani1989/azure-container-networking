//-----------------------------------------------------------------------
// <copyright file="HostAgentClient.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// 
//-----------------------------------------------------------------------
using DNCService;
using DNCService.API;
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
            if (DncConfiguration.NMAgentSimulatorMode)
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

            HttpResponseMessage joinResponse;
            try
            {
                joinResponse = await httpClient.PostAsync(joinNetworkUrl, httpContent);
            }
            catch (Exception exception)
            {
                string errorMsg = (exception.InnerException == null) ?
                    exception.Message : exception.InnerException.Message;
                throw new DncHostAgentException(
                    new ErrorInfo(ErrorCode.StatusInternalServerError,
                        $"Failed to post request to host agent for joinNetwork due to exception: {errorMsg}",
                        ErrorType.Retriable));
            }

            switch (joinResponse.StatusCode)
            {
                // TODO: Missing StatusProcessing (102) in System.Net.HttpStatusCode
                case (HttpStatusCode)102:/*StatusProcessing*/
                    throw new DncHostAgentException(
                        new ErrorInfo(ErrorCode.ProcessingPending,
                            "Join for network pending",
                            (int)joinResponse.StatusCode,
                            ErrorType.Retriable));

                case HttpStatusCode.Unauthorized:
                    throw new DncHostAgentException(
                        new ErrorInfo(ErrorCode.NetworkJoinFailed,
                            "Join for network failed due to authentication failure",
                            (int)joinResponse.StatusCode,
                            ErrorType.NonRetriable));

                case HttpStatusCode.OK:
                    break;

                default:
                    throw new DncHostAgentException(
                        new ErrorInfo(ErrorCode.NetworkJoinFailed,
                            $"Join for network failed. Unable to retrive network response" +
                            $" due to error {joinResponse.StatusCode}",
                            (int)joinResponse.StatusCode,
                            ErrorType.Retriable));
            }

            HttpResponseMessage getResponse;
            try
            {
                getResponse = await httpClient.GetAsync(getNetworkUrl);
            }
            catch (Exception exception)
            {
                string errorMsg = (exception.InnerException == null) ?
                    exception.Message : exception.InnerException.Message;
                throw new DncHostAgentException(
                    new ErrorInfo(ErrorCode.StatusInternalServerError,
                        $"Failed to post request to host agent for getNetworkConfiguration due to exception: {exception}",
                        ErrorType.Retriable));
            }

            string responseBody = await getResponse.Content.ReadAsStringAsync();
            VirtualNetworkResponse virtualNetworkResponse;
            HttpStatusCode getNetworkResponseCode;
            try
            {
                virtualNetworkResponse = JsonConvert.DeserializeObject<VirtualNetworkResponse>(responseBody);
                getNetworkResponseCode = (HttpStatusCode)Enum.Parse(typeof(HttpStatusCode),
                    virtualNetworkResponse.ResponseCode);
            }
            catch (Exception exception)
            {
                string errorMsg = (exception.InnerException == null) ?
                    exception.Message : exception.InnerException.Message;
                throw new DncHostAgentException(
                    new ErrorInfo(ErrorCode.DecodeResponseFailed,
                        $"Failed to decode response from host agent for getNetworkConfiguration " +
                        $"due to exception: {errorMsg}",
                        ErrorType.Retriable));
            }

            switch (getNetworkResponseCode)
            {
                // TODO: Missing StatusProcessing (102) in System.Net.HttpStatusCode
                case (HttpStatusCode)102:/*StatusProcessing*/
                    throw new DncHostAgentException(
                        new ErrorInfo(ErrorCode.ProcessingPending,
                            "Join for network pending",
                            (int)getNetworkResponseCode,
                            ErrorType.Retriable));

                case HttpStatusCode.Unauthorized:
                    throw new DncHostAgentException(
                        new ErrorInfo(ErrorCode.NetworkJoinFailed,
                            "Join for network failed due to authentication failure.",
                            (int)getNetworkResponseCode,
                            ErrorType.NonRetriable));

                case HttpStatusCode.OK:
                    ServiceEventSource.Current.Message($"Retrieved virtual network response:" +
                        $" {virtualNetworkResponse}");
                    return virtualNetworkResponse;

                default:
                    throw new DncHostAgentException(
                        new ErrorInfo(ErrorCode.NetworkJoinFailed,
                            $"Join for network failed. Unable to retrive network response " +
                            $"due to error {getNetworkResponseCode}",
                            (int)getNetworkResponseCode,
                            ErrorType.NonRetriable));
            }
        }

        /// <summary>
        /// Get system tag definition from host agent.
        /// </summary>
        /// <param name="tagName">Supply the name of the tag.</param>
        /// <returns>Tag Response.</returns>
        public async Task<TagResponse> GetSystemTagDefinition(
            string tagName,
            bool ipv4 = true)
        {
            if (DncConfiguration.NMAgentSimulatorMode)
            {
                return null;
            }

            string getSystemTagDefinitionUrl = string.Format(
                DncConfiguration.HostAgentSettings.ConnectionStrings.GetSystemTagDefinitionURLFmt,
                DncConfiguration.HostAgentSettings.WireServerIP,
                tagName,
                (ipv4) ? "IPv4" : "IPv6");

            ServiceEventSource.Current.Message($"GetSystemTagDefinitionUrl:{getSystemTagDefinitionUrl}");

            HttpClient httpClient = new HttpClient();
            HttpResponseMessage getResponse;
            try
            {
                getResponse = await httpClient.GetAsync(getSystemTagDefinitionUrl);
            }
            catch (Exception exception)
            {
                string errorMsg = (exception.InnerException == null) ?
                    exception.Message : exception.InnerException.Message;
                throw new DncHostAgentException(
                    new ErrorInfo(ErrorCode.StatusInternalServerError,
                        $"Failed to post request to host agent for GetSystemTagDefinition " +
                        $"due to exception: {errorMsg}",
                        ErrorType.Retriable));
            }

            string responseBody = await getResponse.Content.ReadAsStringAsync();
            TagResponse tagResponse;
            HttpStatusCode getTagResponseCode;
            try
            {
                tagResponse = JsonConvert.DeserializeObject<TagResponse>(responseBody);
                getTagResponseCode = (HttpStatusCode)Enum.Parse(
                    typeof(HttpStatusCode),
                    tagResponse.ResponseCode);
            }
            catch (Exception exception)
            {
                string errorMsg = (exception.InnerException == null) ?
                    exception.Message : exception.InnerException.Message;
                throw new DncHostAgentException(
                    new ErrorInfo(ErrorCode.DecodeResponseFailed,
                        $"Failed to decode response from host agent for GetSystemTagDefinition " +
                        $"due to exception: {errorMsg}",
                        ErrorType.Retriable));
            }

            // TODO: Missing StatusProcessing (102) in System.Net.HttpStatusCode
            if (getTagResponseCode == (HttpStatusCode)102/*StatusProcessing*/)
            {
                throw new DncHostAgentException(
                    new ErrorInfo(ErrorCode.ProcessingPending,
                        $"GetSystemTagDefinition for {tagName} is in pending state",
                        (int)getTagResponseCode,
                        ErrorType.Retriable));
            }

            if (getTagResponseCode == HttpStatusCode.Unauthorized)
            {
                throw new DncHostAgentException(
                    new ErrorInfo(ErrorCode.AgentTagDefinitionRetrievalFailed,
                        $"GetSystemTagDefinition for {tagName} is unauthorized",
                        (int)getTagResponseCode,
                        ErrorType.NonRetriable));
            }

            if (getTagResponseCode != HttpStatusCode.OK)
            {
                throw new DncHostAgentException(
                    new ErrorInfo(ErrorCode.AgentTagDefinitionRetrievalFailed,
                        $"Tag definition retrival failed from host agent " +
                        $"due to error {getTagResponseCode}",
                        (int)getTagResponseCode,
                        ErrorType.NonRetriable));
            }

            return tagResponse;
        }

        /// <summary>
        /// Create network container request.
        /// </summary>
        /// <param name="networkContainerInfo">Supply the network container info.</param>
        /// <returns></returns>
        public async Task CreateNetworkContainer(NetworkContainerInfo networkContainerInfo)
        {
            if (DncConfiguration.NMAgentSimulatorMode)
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

            HttpResponseMessage response;
            try
            {
                response = await httpClient.PostAsync(createNCUrl, httpContent);
            }
            catch (Exception exception)
            {
                string errorMsg = (exception.InnerException == null) ?
                    exception.Message : exception.InnerException.Message;
                throw new DncHostAgentException(
                    new ErrorInfo(ErrorCode.StatusInternalServerError,
                        $"Failed to post request to host agent for Create NC due to exception: {errorMsg}",
                        ErrorType.Retriable));
            }

            if (response.StatusCode != HttpStatusCode.OK)
            {
                throw new DncHostAgentException(
                    new ErrorInfo(ErrorCode.PublishingFailed,
                        $"Create NC failed with response code {response.StatusCode}",
                        (int)response.StatusCode,
                        ErrorType.Retriable));
            }

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
                string errorMsg = (exception.InnerException == null) ?
                    exception.Message : exception.InnerException.Message;
                throw new DncHostAgentException(
                    new ErrorInfo(ErrorCode.DecodeResponseFailed,
                        $"Failed to decode response from host agent for Create NC due to exception: {errorMsg}",
                        ErrorType.Retriable));
            }

            if (createNCReponseCode != HttpStatusCode.OK)
            {
                throw new DncHostAgentException(
                    new ErrorInfo(ErrorCode.PublishingFailed,
                        $"Create NC failed with response code {createNCReponseCode}",
                        (int)response.StatusCode,
                        ErrorType.NonRetriable));
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
        public async Task DeleteNetworkContainer(
            string networkContainerID,
            string nodeInterfaceID,
            string accessToken)
        {
            if (DncConfiguration.NMAgentSimulatorMode)
            {
                return;
            }

            string deleteNCUrl = string.Format(
                DncConfiguration.HostAgentSettings.ConnectionStrings.DeleteNetworkContainerURLFmt,
                DncConfiguration.HostAgentSettings.WireServerIP,
                nodeInterfaceID,
                networkContainerID,
                accessToken);

            ServiceEventSource.Current.Message("DeleteNCUrl:{0}", deleteNCUrl);

            var httpContent = new StringContent(" ", Encoding.UTF8, "application/json");
            HttpClient httpClient = new HttpClient();

            HttpResponseMessage response;
            try
            {
                response = await httpClient.PostAsync(deleteNCUrl, httpContent);
            }
            catch (Exception exception)
            {
                string errorMsg = (exception.InnerException == null) ?
                    exception.Message : exception.InnerException.Message;
                throw new DncHostAgentException(
                    new ErrorInfo(ErrorCode.StatusInternalServerError,
                        $"Failed to post request to host agent for Delete NC due to exception: {errorMsg}",
                        ErrorType.Retriable));
            }

            if (response.StatusCode != HttpStatusCode.OK)
            {
                throw new DncHostAgentException(
                    new ErrorInfo(ErrorCode.UnpublishingFailed,
                        $"Delete NC failed with response code {response.StatusCode}",
                        (int)response.StatusCode,
                        ErrorType.Retriable));
            }

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
                string errorMsg = (exception.InnerException == null) ?
                    exception.Message : exception.InnerException.Message;
                throw new DncHostAgentException(
                    new ErrorInfo(ErrorCode.DecodeResponseFailed,
                        $"Failed to decode response from host agent for Delete NC due to exception: {errorMsg}",
                        ErrorType.Retriable));
            }

            if (deleteNCReponseCode != HttpStatusCode.OK)
            {
                throw new DncHostAgentException(
                    new ErrorInfo(ErrorCode.UnpublishingFailed,
                        $"Delete NC failed with response code {deleteNCReponseCode}",
                        (int)deleteNCReponseCode,
                        ErrorType.NonRetriable));
            }

            ServiceEventSource.Current.Message(
                $"Successfully unpublished NC:{networkContainerID} " +
                $"Received response:{deleteNCResponse}");
        }
    }
}
