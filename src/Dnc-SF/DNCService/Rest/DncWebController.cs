//--------------------------------------------------------------------------
// <copyright file="DncWebController.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
// Implements the DNC web controller.
// </summary>
//--------------------------------------------------------------------------

using DNCService.API;
using DNCService.Interfaces;
using Microsoft.AspNetCore.Http;
using Microsoft.AspNetCore.Routing;
using Microsoft.ServiceFabric.Services.Client;
using Microsoft.ServiceFabric.Services.Remoting.Client;
using Newtonsoft.Json;
using System;
using System.Net;
using System.Text;
using System.Threading.Tasks;

namespace DNCService.Rest
{
    /// <summary>
    /// Implements routing for DNC.
    /// </summary>
    public class DncWebController
    {
        private const string dncServiceUri = "fabric:/DNCServiceApplication/DNCService";

        /// <summary>
        /// Helper to set the reponse body and status.
        /// </summary>
        /// <param name="httpContext">Supply the http context.</param>
        /// <param name="responseBytes">Supply the response buffer.</param>
        /// <param name="httpStatusCode">Supply the status code.</param>
        /// <returns></returns>
        private async Task WriteHttpResponse(
            HttpContext httpContext,
            byte[] responseBytes,
            int httpStatusCode)
        {
            httpContext.Response.StatusCode = httpStatusCode;
            if (responseBytes != null)
            {
                httpContext.Response.ContentType = "text/json";
                httpContext.Response.ContentLength = responseBytes.Length;
                await httpContext.Response.Body.WriteAsync(responseBytes, 0, responseBytes.Length);
            }
        }

        /// <summary>
        /// Wrapper to invoke DNC service and setting the response.
        /// </summary>
        /// <param name="func">Delegate to be invoked.</param>
        /// <param name="httpContext">Supply the httpcontext.</param>
        /// <returns></returns>
        public async Task Invoke(Func<Task<byte[]>> func, HttpContext httpContext)
        {
            try
            {
                byte[] responseBytes = await func();
                await WriteHttpResponse(httpContext, responseBytes, (int)HttpStatusCode.OK);
            }
            catch (Exception exception)
            {
                ErrorInfo errorInfo;
                ServiceEventSource.Current.Message("Received exception:{0}.", exception);
                if (exception is DncManagementException dncManagementException)
                {
                    errorInfo = dncManagementException.ErrorInfo;
                }
                else if (exception.InnerException != null &&
                    exception.InnerException is DncManagementException dncManagementInnerException)
                {
                    errorInfo = dncManagementInnerException.ErrorInfo;
                }
                else
                {
                    errorInfo = new ErrorInfo(
                        ErrorCode.StatusInternalServerError, 
                        $"Failed with exception:{exception}",
                        (int)HttpStatusCode.InternalServerError,
                        ErrorType.NonRetriable);
                }
                byte[] responseBytes = Encoding.UTF8.GetBytes(JsonConvert.SerializeObject(errorInfo));
                await WriteHttpResponse(httpContext, responseBytes, errorInfo.HTTPStatus);
            }
        }

        /// <summary>
        /// Create/Update node.
        /// </summary>
        /// <param name="httpContext">Supply the context.</param>
        /// <returns></returns>
        public async Task CreateOrUpdateNodeAsync(HttpContext httpContext)
        {
            async Task<byte[]> createNodeAsync()
            {
                string nodeID = (string)httpContext.GetRouteValue("NodeID");
                if(string.IsNullOrEmpty(nodeID))
                {
                    throw new DncNodeManagementException(
                        "",
                        new ErrorInfo(
                            ErrorCode.InvalidNodeID,
                            "NodeID cannot be null or empty",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }

                NodeRequest nodeInfoRequest = null;
                try
                {
                    nodeInfoRequest = await ReadAsAsync<NodeRequest>(httpContext);
                }
                catch(Exception exception)
                {
                    throw new DncNodeManagementException(
                        "",
                        new ErrorInfo(
                            ErrorCode.InvalidNodeRequest,
                            $"Failed to parse request body. Exception:{exception.Message}",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }

                if(nodeInfoRequest == null)
                {
                    throw new DncNodeManagementException(
                        "",
                        new ErrorInfo(
                            ErrorCode.InvalidNodeRequest,
                            $"Failed to parse request body. Request post parsing request body is null",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }

                if(nodeInfoRequest.IPAddresses == null || nodeInfoRequest.IPAddresses.Length == 0)
                {
                    throw new DncNodeManagementException(
                        "",
                        new ErrorInfo(
                            ErrorCode.InvalidNodeRequest,
                            $"IPAddress for node cannot be null or empty",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }

                if (string.IsNullOrEmpty(nodeInfoRequest.OrchestratorType))
                {
                    throw new DncNodeManagementException(
                        "",
                        new ErrorInfo(
                            ErrorCode.InvalidNodeRequest,
                            $"Orchestrator context cannot be null or empty",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }

                if (!nodeInfoRequest.OrchestratorType.Equals(
                    Cns.Data.Constants.ServiceFabric,
                    StringComparison.CurrentCultureIgnoreCase) &&
                    !nodeInfoRequest.OrchestratorType.Equals(
                    Cns.Data.Constants.Kubernetes,
                    StringComparison.CurrentCultureIgnoreCase))
                {
                    throw new DncNodeManagementException(
                        "",
                        new ErrorInfo(
                            ErrorCode.InvalidNodeRequest,
                            $"Orchestrator type {nodeInfoRequest.OrchestratorType} is invalid. " +
                            $"Supported orchestrators {Cns.Data.Constants.ServiceFabric}, {Cns.Data.Constants.Kubernetes}.",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }

                var dncServiceProxy = ServiceProxy.Create<IDNCService>(new Uri(dncServiceUri),
                new ServicePartitionKey(1),
                Microsoft.ServiceFabric.Services.Communication.Client.TargetReplicaSelector.PrimaryReplica);

                await dncServiceProxy.CreateOrUpdateNodeAsync(
                    nodeID,
                    nodeInfoRequest,
                    "");
                return null;
            }

            await Invoke(createNodeAsync, httpContext);
        }

        /// <summary>
        /// Get node information.
        /// </summary>
        /// <param name="httpContext">Supply the context.</param>
        /// <returns></returns>
        public async Task GetNodeAsync(HttpContext httpContext)
        {
            async Task<byte[]> getNodeAsync()
            {
                string nodeID = (string)httpContext.GetRouteValue("NodeID");
                if (string.IsNullOrEmpty(nodeID))
                {
                    throw new DncNodeManagementException(
                        "",
                        new ErrorInfo(
                            ErrorCode.InvalidNodeID,
                            "NodeID cannot be null or empty",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }
                var dncServiceProxy = ServiceProxy.Create<IDNCService>(new Uri(dncServiceUri),
                    new ServicePartitionKey(1),
                    Microsoft.ServiceFabric.Services.Communication.Client.TargetReplicaSelector.PrimaryReplica);
                NodeResponse nodeResponse = await dncServiceProxy.GetNodeInfoResponseAsync(
                    nodeID);
                return Encoding.UTF8.GetBytes(JsonConvert.SerializeObject(nodeResponse));
            }

            await Invoke(getNodeAsync, httpContext);
        }

        /// <summary>
        /// Delete node.
        /// </summary>
        /// <param name="httpContext">Supply the context.</param>
        /// <returns></returns>
        public async Task DeleteNodeAsync(HttpContext httpContext)
        {
            async Task<byte[]> getNodeAsync()
            {
                string nodeID = (string)httpContext.GetRouteValue("NodeID");
                if (string.IsNullOrEmpty(nodeID))
                {
                    throw new DncNodeManagementException(
                        "",
                        new ErrorInfo(
                            ErrorCode.InvalidNodeID,
                            "NodeID cannot be null or empty",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }

                var dncServiceProxy = ServiceProxy.Create<IDNCService>(new Uri(dncServiceUri),
                new ServicePartitionKey(1),
                Microsoft.ServiceFabric.Services.Communication.Client.TargetReplicaSelector.PrimaryReplica);
                await dncServiceProxy.DeleteNodeAsync(
                                 nodeID);
                return null;
            }

            await Invoke(getNodeAsync, httpContext);
        }

        /// <summary>
        /// Create/Update network.
        /// </summary>
        /// <param name="httpContext">Supply the context.</param>
        /// <returns></returns>
        public async Task CreateOrUpdateNetworkAsync(HttpContext httpContext)
        {
            async Task<byte[]> createNetworkAsync()
            {
                string networkID = (string)httpContext.GetRouteValue("NetworkID");
                if (string.IsNullOrEmpty(networkID))
                {
                    throw new DncNodeManagementException(
                        "",
                        new ErrorInfo(
                            ErrorCode.InvalidNetworkID,
                            "NetworkID cannot be null or empty",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }

                var dncServiceProxy = ServiceProxy.Create<IDNCService>(new Uri(dncServiceUri),
                    new ServicePartitionKey(1),
                    Microsoft.ServiceFabric.Services.Communication.Client.TargetReplicaSelector.PrimaryReplica);
                NetworkRequest networkInfoRequest = await ReadAsAsync<NetworkRequest>(httpContext);
                await dncServiceProxy.CreateOrUpdateNetworkAsync(
                    networkID,
                    networkInfoRequest,
                    "");
                return null;
            }

            await Invoke(createNetworkAsync, httpContext);
        }

        /// <summary>
        /// Delete network.
        /// </summary>
        /// <param name="httpContext">Supply the context.</param>
        /// <returns></returns>
        public async Task DeleteNetworkAsync(HttpContext httpContext)
        {
            async Task<byte[]> deleteNetworkAsync()
            {
                string networkID = (string)httpContext.GetRouteValue("NetworkID");
                if (string.IsNullOrEmpty(networkID))
                {
                    throw new DncNodeManagementException(
                        "",
                        new ErrorInfo(
                            ErrorCode.InvalidNetworkID,
                            "NetworkID cannot be null or empty",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }

                var dncServiceProxy = ServiceProxy.Create<IDNCService>(new Uri(dncServiceUri),
                new ServicePartitionKey(1),
                Microsoft.ServiceFabric.Services.Communication.Client.TargetReplicaSelector.PrimaryReplica);
                await dncServiceProxy.DeleteNetworkAsync(
                    networkID);
                return null;
            }

            await Invoke(deleteNetworkAsync, httpContext);
        }

        /// <summary>
        /// Create/Update network contianer.
        /// </summary>
        /// <param name="httpContext">Supply the context.</param>
        /// <returns></returns>
        public async Task CreateOrUpdateNetworkContainerAsync(HttpContext httpContext)
        {
            async Task<byte[]> createNetworkContainerAsync()
            {
                string networkID = (string)httpContext.GetRouteValue("NetworkID");
                if (string.IsNullOrEmpty(networkID))
                {
                    throw new DncNetworkContainerException(
                        "",
                        "",
                        new ErrorInfo(
                            ErrorCode.InvalidNetworkID,
                            "NetworkID cannot be null or empty",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }

                string networkContainerID = (string)httpContext.GetRouteValue("NetworkContainerID");
                if (string.IsNullOrEmpty(networkContainerID))
                {
                    throw new DncNetworkContainerException(
                        networkID,
                        "",
                        new ErrorInfo(
                            ErrorCode.InvalidNetworkContainerID,
                            "NetworkContainerID cannot be null or empty",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }

                byte[] body = await ReadBodyAsAsync(httpContext);
                NetworkContainerRequest networkContainerRequest;
                NetworkContainerRequestV1 networkContainerRequestV1 = null;
                try
                {
                    networkContainerRequestV1 = Deserialize<NetworkContainerRequestV1>(body);
                }
                catch (Exception exception)
                {
                    throw new DncNetworkContainerException(
                        networkID,
                        networkContainerID,
                        new ErrorInfo(
                            ErrorCode.InvalidNetworkContainerRequest,
                            $"Received exception {exception.Message} while serializing Json body",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }

                if (networkContainerRequestV1 != null && !string.IsNullOrWhiteSpace(networkContainerRequestV1.ReservationID))
                {
                    networkContainerRequest = networkContainerRequestV1;
                }
                else
                {
                    NetworkContainerRequestV2 networkContainerRequestV2 = null;
                    try
                    {
                        networkContainerRequestV2 = Deserialize<NetworkContainerRequestV2>(body);
                    }
                    catch (Exception exception)
                    {
                        throw new DncNetworkContainerException(
                            networkID,
                            networkContainerID,
                            new ErrorInfo(
                                ErrorCode.InvalidNetworkContainerRequest,
                                $"Received exception {exception.Message} while serializing Json body",
                                (int)HttpStatusCode.BadRequest,
                                ErrorType.NonRetriable));
                    }

                    if (networkContainerRequestV2 == null)
                    {
                        throw new DncNetworkContainerException(
                            networkID,
                            networkContainerID,
                            new ErrorInfo(
                                ErrorCode.InvalidNetworkContainerRequest,
                                "Failed to serialize Json body to network container request",
                                (int)HttpStatusCode.BadRequest,
                                ErrorType.NonRetriable));
                    }

                    if (networkContainerRequestV2.AllocationRequest == null)
                    {
                        throw new DncNetworkContainerException(
                            networkID,
                            networkContainerID,
                            new ErrorInfo(
                                ErrorCode.InvalidNetworkContainerRequest,
                                "Allocation request cannot be null or empty",
                                (int)HttpStatusCode.BadRequest,
                                ErrorType.NonRetriable));
                    }

                    if (string.IsNullOrEmpty(networkContainerRequestV2.AllocationRequest.SubnetName))
                    {
                        throw new DncNetworkContainerException(
                            networkID,
                            networkContainerID,
                            new ErrorInfo(
                                ErrorCode.InvalidNetworkContainerRequest,
                                "Subnet name in allocation request cannot be null or empty",
                                (int)HttpStatusCode.BadRequest,
                                ErrorType.NonRetriable));
                    }

                    networkContainerRequest = networkContainerRequestV2;
                }
                if (networkContainerRequest.AssociationInfo == null)
                {
                    throw new DncNetworkContainerException(
                        networkID,
                        networkContainerID,
                        new ErrorInfo(
                            ErrorCode.InvalidNetworkContainerRequest,
                            "AssociationInfo cannot be null or empty",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }

                if (string.IsNullOrEmpty(networkContainerRequest.AssociationInfo.ContainerType))
                {
                    throw new DncNetworkContainerException(
                        networkID,
                        networkContainerID,
                        new ErrorInfo(
                            ErrorCode.InvalidNetworkContainerRequest,
                            "ContainerType in association info cannot be null or empty",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }

                if (string.IsNullOrEmpty(networkContainerRequest.AssociationInfo.NodeID))
                {
                    throw new DncNetworkContainerException(
                        networkID,
                        networkContainerID,
                        new ErrorInfo(
                            ErrorCode.InvalidNetworkContainerRequest,
                            "NodeID in association info cannot be null or empty",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }

                if (networkContainerRequest.AssociationInfo.OrchestratorContext == null)
                {
                    throw new DncNetworkContainerException(
                        networkID,
                        networkContainerID,
                        new ErrorInfo(
                            ErrorCode.InvalidNetworkContainerRequest,
                            "OrchestratorContext in association info cannot be null or empty",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }

                if (!networkContainerRequest.AssociationInfo.ContainerType.Equals(
                    Cns.Data.Constants.ClearContainer,
                    StringComparison.CurrentCultureIgnoreCase) &&
                    !networkContainerRequest.AssociationInfo.ContainerType.Equals(
                    Cns.Data.Constants.AzureContainerInstance,
                    StringComparison.CurrentCultureIgnoreCase))
                {
                    throw new DncNetworkContainerException(
                        networkID,
                        networkContainerID,
                        new ErrorInfo(
                            ErrorCode.InvalidNodeRequest,
                            $"Container type {networkContainerRequest.AssociationInfo.ContainerType} is invalid. " +
                            $"Supported container types {Cns.Data.Constants.ClearContainer}, {Cns.Data.Constants.AzureContainerInstance}.",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }

                var dncServiceProxy = ServiceProxy.Create<IDNCService>(new Uri(dncServiceUri),
                    new ServicePartitionKey(1),
                    Microsoft.ServiceFabric.Services.Communication.Client.TargetReplicaSelector.PrimaryReplica);


                await dncServiceProxy.CreateOrUpdateNetworkContainerAsync(
                    networkID,
                    networkContainerID,
                    networkContainerRequest,
                    "");
                return null;
            }

            await Invoke(createNetworkContainerAsync, httpContext);
        }

        /// <summary>
        /// Get network container information.
        /// </summary>
        /// <param name="httpContext">Supply the context.</param>
        /// <returns></returns>
        public async Task GetNetworkContainerAsync(HttpContext httpContext)
        {
            async Task<byte[]> getNetworkContainerAsync()
            {
                string networkID = (string)httpContext.GetRouteValue("NetworkID");
                if (string.IsNullOrEmpty(networkID))
                {
                    throw new DncNetworkContainerException(
                        "",
                        "",
                        new ErrorInfo(
                            ErrorCode.InvalidNetworkID,
                            "NetworkID cannot be null or empty",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }

                string networkContainerID = (string)httpContext.GetRouteValue("NetworkContainerID");
                if (string.IsNullOrEmpty(networkContainerID))
                {
                    throw new DncNetworkContainerException(
                        networkID,
                        "",
                        new ErrorInfo(
                            ErrorCode.InvalidNetworkContainerID,
                            "NetworkContainerID cannot be null or empty",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }
                var dncServiceProxy = ServiceProxy.Create<IDNCService>(new Uri(dncServiceUri),
                    new ServicePartitionKey(1),
                    Microsoft.ServiceFabric.Services.Communication.Client.TargetReplicaSelector.PrimaryReplica);

                NetworkContainerResponse networkContainerResponse = await dncServiceProxy.GetNetworkContainerResponseAsync(
                    networkID,
                    networkContainerID);
                byte[] responseBytes = Encoding.UTF8.GetBytes(JsonConvert.SerializeObject(networkContainerResponse));
                return responseBytes;
            }

            await Invoke(getNetworkContainerAsync, httpContext);
        }

        /// <summary>
        /// Delete network container.
        /// </summary>
        /// <param name="httpContext">Supply the context.</param>
        /// <returns></returns>
        public async Task DeleteNetworkContainerAsync(HttpContext httpContext)
        {
            async Task<byte[]> deleteNetworkContainerAsync()
            {
                string networkID = (string)httpContext.GetRouteValue("NetworkID");
                if (string.IsNullOrEmpty(networkID))
                {
                    throw new DncNetworkContainerException(
                        "",
                        "",
                        new ErrorInfo(
                            ErrorCode.InvalidNetworkID,
                            "NetworkID cannot be null or empty",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }

                string networkContainerID = (string)httpContext.GetRouteValue("NetworkContainerID");
                if (string.IsNullOrEmpty(networkContainerID))
                {
                    throw new DncNetworkContainerException(
                        networkID,
                        "",
                        new ErrorInfo(
                            ErrorCode.InvalidNetworkContainerID,
                            "NetworkContainerID cannot be null or empty",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }

                var dncServiceProxy = ServiceProxy.Create<IDNCService>(new Uri(dncServiceUri),
                    new ServicePartitionKey(1),
                    Microsoft.ServiceFabric.Services.Communication.Client.TargetReplicaSelector.PrimaryReplica);

                await dncServiceProxy.DeleteNetworkContainerAsync(
                    networkID,
                    networkContainerID);
                return null;
            }

            await Invoke(deleteNetworkContainerAsync, httpContext);
        }

        /// <summary>
        /// Join subnet.
        /// </summary>
        /// <param name="httpContext">Supply the context.</param>
        /// <returns></returns>
        public async Task JoinSubnetAsync(HttpContext httpContext)
        {
            async Task<byte[]> joinNetworkContainerAsync()
            {
                string networkID = (string)httpContext.GetRouteValue("NetworkID");
                if (string.IsNullOrEmpty(networkID))
                {
                    throw new DncNetworkManagementException(
                        "",
                        new ErrorInfo(
                            ErrorCode.InvalidNetworkID,
                            "NetworkID cannot be null or empty",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }

                string subnetName = (string)httpContext.GetRouteValue("SubnetName");
                if (string.IsNullOrEmpty(subnetName))
                {
                    throw new DncNetworkManagementException(
                        networkID,
                        new ErrorInfo(
                            ErrorCode.InvalidSubnetName,
                            "SubnetName cannot be null or empty",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }

                SubnetRequest subnetRequest = null;
                try
                {
                    subnetRequest = await ReadAsAsync<SubnetRequest>(httpContext);
                }
                catch (Exception exception)
                {
                    throw new DncNetworkManagementException(
                        networkID,
                        new ErrorInfo(
                            ErrorCode.InvalidSubnetRequest,
                            $"Failed to parse request body. Exception:{exception.Message}",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }

                if (subnetRequest == null)
                {
                    throw new DncNetworkManagementException(
                        networkID,
                        new ErrorInfo(
                            ErrorCode.InvalidSubnetRequest,
                            $"Failed to parse request body. Request post parsing request body is null",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }

                if(string.IsNullOrEmpty(subnetRequest.AuthenticationToken))
                {
                    throw new DncNetworkManagementException(
                        networkID,
                        new ErrorInfo(
                            ErrorCode.InvalidSubnetRequest,
                            $"AuthenticationToken cannot be null or empty.",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }

                var dncServiceProxy = ServiceProxy.Create<IDNCService>(new Uri(dncServiceUri),
                            new ServicePartitionKey(1),
                            Microsoft.ServiceFabric.Services.Communication.Client.TargetReplicaSelector.PrimaryReplica);

                await dncServiceProxy.JoinSubnet(networkID, subnetName, subnetRequest, "");
                return null;
            }
            await Invoke(joinNetworkContainerAsync, httpContext);
        }

        /// <summary>
        /// Add network profile.
        /// </summary>
        /// <param name="httpContext">Supply the context.</param>
        /// <returns></returns>
        public async Task AddNetworkProfileAsync(HttpContext httpContext)
        {
            async Task<byte[]> addNetworkProfileAsync()
            {
                var dncServiceProxy = ServiceProxy.Create<IDNCService>(new Uri(dncServiceUri),
                new ServicePartitionKey(1),
                Microsoft.ServiceFabric.Services.Communication.Client.TargetReplicaSelector.PrimaryReplica);

                string networkProfileID = (string)httpContext.GetRouteValue("NetworkProfileID");

                await dncServiceProxy.AddNetworkProfileAsync(networkProfileID);
                return null;
            }

            await Invoke(addNetworkProfileAsync, httpContext);
        }

        /// <summary>
        /// Delete network profile.
        /// </summary>
        /// <param name="httpContext">Supply the context.</param>
        /// <returns></returns>
        public async Task DeleteNetworkProfileAsync(HttpContext httpContext)
        {
            async Task<byte[]> deleteNetworkProfileAsync()
            {
                var dncServiceProxy = ServiceProxy.Create<IDNCService>(new Uri(dncServiceUri),
                new ServicePartitionKey(1),
                Microsoft.ServiceFabric.Services.Communication.Client.TargetReplicaSelector.PrimaryReplica);

                string networkProfileID = (string)httpContext.GetRouteValue("NetworkProfileID");

                await dncServiceProxy.DeleteNetworkProfileAsync(networkProfileID);
                return null;
            }

            await Invoke(deleteNetworkProfileAsync, httpContext);
        }

        /// <summary>
        /// Get subnet information.
        /// </summary>
        /// <param name="httpContext">Supply the context.</param>
        /// <returns></returns>
        public async Task GetSubnetAsync(HttpContext httpContext)
        {
            async Task<byte[]> getSubnetAsync()
            {
                string networkID = (string)httpContext.GetRouteValue("NetworkID");
                if (string.IsNullOrEmpty(networkID))
                {
                    throw new DncNetworkManagementException(
                        "",
                        new ErrorInfo(
                            ErrorCode.InvalidNetworkID,
                            "NetworkID cannot be null or empty",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }

                string subnetName = (string)httpContext.GetRouteValue("SubnetName");
                if (string.IsNullOrEmpty(subnetName))
                {
                    throw new DncNetworkManagementException(
                        networkID,
                        new ErrorInfo(
                            ErrorCode.InvalidSubnetName,
                            "SubnetName cannot be null or empty",
                            (int)HttpStatusCode.BadRequest,
                            ErrorType.NonRetriable));
                }
                var dncServiceProxy = ServiceProxy.Create<IDNCService>(new Uri(dncServiceUri),
                    new ServicePartitionKey(1),
                    Microsoft.ServiceFabric.Services.Communication.Client.TargetReplicaSelector.PrimaryReplica);

                SubnetResponse subnetResponse = await dncServiceProxy.GetSubnetConfigurationResponseAsync(
                    networkID,
                    subnetName);
                byte[] responseBytes = Encoding.UTF8.GetBytes(JsonConvert.SerializeObject(subnetResponse));
                return responseBytes;
            }
            await Invoke(getSubnetAsync, httpContext);
        }

        /// <summary>
        /// Get network status information.
        /// </summary>
        /// <param name="httpContext">Supply the context.</param>
        /// <returns></returns>
        public async Task GetNetworkStatusAsync(HttpContext httpContext)
        {
            async Task<byte[]> getNetworkStatusAsync()
            {
                RequestStatus requestStatus;
                string networkID = (string)httpContext.GetRouteValue("NetworkID");
                try
                {
                    if (string.IsNullOrEmpty(networkID))
                    {
                        throw new DncNodeManagementException(
                            "",
                            new ErrorInfo(
                                ErrorCode.InvalidNetworkID,
                                "NetworkID cannot be null or empty",
                                (int)HttpStatusCode.BadRequest,
                                ErrorType.NonRetriable));
                    }
                }
                catch (Exception exception)
                {
                    DncManagementException dncManagementException =
                        (DncManagementException)exception;
                    requestStatus = new RequestStatus("", "", dncManagementException.ErrorInfo);
                    return Encoding.UTF8.GetBytes(JsonConvert.SerializeObject(requestStatus));
                }

                var dncServiceProxy = ServiceProxy.Create<IDNCService>(new Uri(dncServiceUri),
                    new ServicePartitionKey(1),
                    Microsoft.ServiceFabric.Services.Communication.Client.TargetReplicaSelector.PrimaryReplica);

                requestStatus = await dncServiceProxy.GetNetworkStatusResponseAsync(networkID);

                return Encoding.UTF8.GetBytes(JsonConvert.SerializeObject(requestStatus));
            }
            await Invoke(getNetworkStatusAsync, httpContext);
        }

        /// <summary>
        /// Get subnet status.
        /// </summary>
        /// <param name="httpContext">Supply the context.</param>
        /// <returns></returns>
        public async Task GetSubnetStatusAsync(HttpContext httpContext)
        {
            async Task<byte[]> getSubnetStatusAsync()
            {
                string networkID = (string)httpContext.GetRouteValue("NetworkID");
                string subnetName = (string)httpContext.GetRouteValue("SubnetName");
                RequestStatus requestStatus;
                try
                {
                    if (string.IsNullOrEmpty(networkID))
                    {
                        throw new DncNetworkManagementException(
                            "",
                            new ErrorInfo(
                                ErrorCode.InvalidNetworkID,
                                "NetworkID cannot be null or empty",
                                (int)HttpStatusCode.BadRequest,
                                ErrorType.NonRetriable));
                    }

                    if (string.IsNullOrEmpty(subnetName))
                    {
                        throw new DncNetworkManagementException(
                            networkID,
                            new ErrorInfo(
                                ErrorCode.InvalidSubnetName,
                                "SubnetName cannot be null or empty",
                                (int)HttpStatusCode.BadRequest,
                                ErrorType.NonRetriable));
                    }
                }
                catch (Exception exception)
                {
                    DncManagementException dncManagementException =
                        (DncManagementException)exception;
                    requestStatus = new RequestStatus("", "", dncManagementException.ErrorInfo);
                    return Encoding.UTF8.GetBytes(JsonConvert.SerializeObject(requestStatus));
                }

                var dncServiceProxy = ServiceProxy.Create<IDNCService>(new Uri(dncServiceUri),
                    new ServicePartitionKey(1),
                    Microsoft.ServiceFabric.Services.Communication.Client.TargetReplicaSelector.PrimaryReplica);

                requestStatus = await dncServiceProxy.GetSubnetStatusResponseAsync(networkID, subnetName);

                return Encoding.UTF8.GetBytes(JsonConvert.SerializeObject(requestStatus));
            }
            await Invoke(getSubnetStatusAsync, httpContext);
        }

        /// <summary>
        /// Get network container status.
        /// </summary>
        /// <param name="httpContext">Supply the context.</param>
        /// <returns></returns>
        public async Task GetNetworkContainerStatusAsync(HttpContext httpContext)
        {
            async Task<byte[]> getNetworkContainerStatusAsync()
            {
                string networkID = (string)httpContext.GetRouteValue("NetworkID");
                string networkContainerID = (string)httpContext.GetRouteValue("NetworkContainerID");
                RequestStatus requestStatus;
                try
                {
                    if (string.IsNullOrEmpty(networkID))
                    {
                        throw new DncNetworkContainerException(
                            "",
                            "",
                            new ErrorInfo(
                                ErrorCode.InvalidNetworkID,
                                "NetworkID cannot be null or empty",
                                (int)HttpStatusCode.BadRequest,
                                ErrorType.NonRetriable));
                    }

                    if (string.IsNullOrEmpty(networkContainerID))
                    {
                        throw new DncNetworkContainerException(
                            networkID,
                            "",
                            new ErrorInfo(
                                ErrorCode.InvalidNetworkContainerID,
                                "NetworkContainerID cannot be null or empty",
                                (int)HttpStatusCode.BadRequest,
                                ErrorType.NonRetriable));
                    }
                }
                catch (Exception exception)
                {
                    DncManagementException dncManagementException =
                        (DncManagementException)exception;
                    requestStatus = new RequestStatus("", "", dncManagementException.ErrorInfo);
                    return Encoding.UTF8.GetBytes(JsonConvert.SerializeObject(requestStatus));
                }

                var dncServiceProxy = ServiceProxy.Create<IDNCService>(new Uri(dncServiceUri),
                    new ServicePartitionKey(1),
                    Microsoft.ServiceFabric.Services.Communication.Client.TargetReplicaSelector.PrimaryReplica);

                requestStatus = await dncServiceProxy.GetNetworkContainerStatusResponseAsync(networkID, networkContainerID);

                return Encoding.UTF8.GetBytes(JsonConvert.SerializeObject(requestStatus));
            }
            await Invoke(getNetworkContainerStatusAsync, httpContext);
        }

        /// <summary>
        /// Helper to deserialize request body.
        /// </summary>
        /// <typeparam name="T">Supply the type of the object.</typeparam>
        /// <param name="context">Supply the context.</param>
        /// <returns>Deserialized object from the request body.</returns>
        public async static Task<T> ReadAsAsync<T>(HttpContext context)
        {
            var buffer = await ReadBodyAsAsync(context);
            return Deserialize<T>(buffer);
        }


        /// <summary>
        /// Helper to deserialize request buffer.
        /// </summary>
        /// <typeparam name="T">Supply the type of the object.</typeparam>
        /// <param name="context">Supply the buffer to be deserialized.</param>
        /// <returns>Deserialized object from the request buffer.</returns>
        public static T Deserialize<T>(byte[] buffer)
        {
            var json = Encoding.UTF8.GetString(buffer);
            return JsonConvert.DeserializeObject<T>(json);
        }

        /// <summary>
        /// Helper to read the request body from http context.
        /// </summary>
        /// <param name="context">Supply the context.</param>
        /// <returns>The extracted buffer.</returns>
        public async static Task<byte[]> ReadBodyAsAsync(HttpContext context)
        {
            var buffer = new byte[Convert.ToInt32(context.Request.ContentLength)];
            await context.Request.Body.ReadAsync(buffer, 0, buffer.Length);
            return buffer;
        }
    }
}
