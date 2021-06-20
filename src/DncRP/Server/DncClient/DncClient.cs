// --------------------------------------------------------------------------------------------------------------------
// <copyright file="DncClient.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------



namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Azure.ContainerNetworking.DncRP;
    using Microsoft.Extensions.Logging;
    using Microsoft.Extensions.Options;
    using System;
    using System.Collections.Generic;
    using System.Linq;
    using System.Net;
    using System.Net.Http;
    using System.Net.Http.Headers;
    using System.Threading.Tasks;
    using System.Text;

    public class DncClient : IDncClient
    {
        private IHttpUtility httpUtility;
        private ILogger logger;
        private string dncEndpoint;
        private const string dncPutSubnetFormat = "{0}/networks/{1}/subnets/{2}?api-version=2021-02-24";
        private const string dncGetSubnetFormat = "{0}/networks/{1}/subnets/{2}/status?api-version=2021-02-24";
        private const string dncPutNodesetFormat = "{0}/nodesets/{1}?api-version=2021-02-24";
        private const string dncGetNodesetFormat = "{0}/nodesets/{1}/status?api-version=2021-02-24";

        public DncClient(
            IHttpUtility httpUtility,
            ILogger logger,
            string dncEndpoint)
        {
            this.httpUtility = httpUtility;
            this.logger = logger;
            this.dncEndpoint = dncEndpoint;
        }

        /// <summary>
        /// PutSubnetAsync - Calls dnc join subnet by http post.
        /// </summary>
        /// <param name="virtualNetworkId">vnet id</param>
        /// <param name="subnetName">subnet name</param>
        /// <param name="authToken">sal token</param>
        /// <returns></returns>
        public async Task PutSubnetAsync(
            string virtualNetworkId,
            string subnetName,
            string authToken)
        {
            string accessToken = await GetDncAccessTokenAsync();

            // send dnc post request for subnet join
            string requestUri = string.Format(dncPutSubnetFormat, dncEndpoint, virtualNetworkId, subnetName);
            using (HttpRequestMessage requestMessage = new HttpRequestMessage(HttpMethod.Post, requestUri))
            {
                requestMessage.Headers.Authorization = new AuthenticationHeaderValue(scheme: "Bearer", parameter: accessToken);
                SubnetJoinRequest request = new SubnetJoinRequest(authToken);
                string json = JsonUtility.Serialize(request);
                requestMessage.Content = new StringContent(json, Encoding.UTF8, "application/json");

                logger.CLogInformation($"Dnc Post subnet RequestMessage uri: { requestMessage.RequestUri.ToString() }");

                HttpResponseMessage response = await this.httpUtility.SendAsync(requestMessage);
                if (response.StatusCode != HttpStatusCode.Accepted)
                {
                    DncFailureMessage errMsg = await ParseErrorResponse(response);
                    throw new ErrorResponseMessageException(
                                           httpStatus: response.StatusCode,
                                           errorCode: ErrorResponseCode.JoinSubnetFailed,
                                           errorMessage: $"DNC Putsubnet failed to this endpoint: " +
                                           $"{dncEndpoint}, , response content: {errMsg.ToString()}");
                }
            }
        }

        /// <summary>
        /// DeleteSubnetAsync - Calls dnc delete subnet by http delete.
        /// </summary>
        /// <param name="virtualNetworkId">vnet id</param>
        /// <param name="subnetName">subnet name</param>
        /// <returns></returns>
        public async Task DeleteSubnetAsync(
            string virtualNetworkId,
            string subnetName,
            bool forceDelete)
        {
            string accessToken = await GetDncAccessTokenAsync();

            // send http delete for subnet
            string requestUri = string.Format(dncPutSubnetFormat, dncEndpoint, virtualNetworkId, subnetName);
            if (forceDelete)
            {
                requestUri = requestUri + "&force-delete=true";
            }

            using (HttpRequestMessage requestMessage = new HttpRequestMessage(HttpMethod.Delete, requestUri))
            {
                requestMessage.Headers.Authorization = new AuthenticationHeaderValue(scheme: "Bearer", parameter: accessToken);
                logger.CLogInformation($"Dnc Delete subnet RequestMessage: {  requestMessage.RequestUri.ToString() }");

                HttpResponseMessage response = await this.httpUtility.SendAsync(requestMessage);
                if (response.StatusCode != HttpStatusCode.Accepted)
                {
                    DncFailureMessage errMsg = await ParseErrorResponse(response);
                    throw new ErrorResponseMessageException(
                                           httpStatus: response.StatusCode,
                                           errorCode: ErrorResponseCode.DeleteSubnetFailed,
                                           errorMessage: $"DNC Deletesubnet failed to this endpoint: " +
                                           $"{dncEndpoint}, response content: {errMsg.ToString()}");
                }
            }
        }

        /// <summary>
        /// GetSubnetAsync - Calls dnc get subnet
        /// </summary>
        /// <param name="virtualNetworkId">vnet id</param>
        /// <param name="subnetName">subnet name</param>
        /// <returns></returns>
        public async Task<HttpResponseMessage> GetSubnetAsync(
           string virtualNetworkId,
           string subnetName)
        {
            string accessToken = await GetDncAccessTokenAsync();

            // http get call to get subnet from dnc state
            string requestUri = string.Format(dncGetSubnetFormat, dncEndpoint, virtualNetworkId, subnetName);
            using (HttpRequestMessage requestMessage = new HttpRequestMessage(HttpMethod.Get, requestUri))
            {
                requestMessage.Headers.Authorization = new AuthenticationHeaderValue(scheme: "Bearer", parameter: accessToken);
                logger.CLogInformation($"DNC Get Subnet RequestMessage: {  requestMessage.RequestUri.ToString() }");
                return await this.httpUtility.SendAsync(requestMessage);
            }
        }

        public async Task PutNodesetAsync(
            string orchestratorId)
        {
            string accessToken = await GetDncAccessTokenAsync();

            // send dnc post request for subnet join
            string requestUri = string.Format(dncPutNodesetFormat, dncEndpoint, orchestratorId);
            using (HttpRequestMessage requestMessage = new HttpRequestMessage(HttpMethod.Put, requestUri))
            {
                requestMessage.Headers.Authorization = new AuthenticationHeaderValue(scheme: "Bearer", parameter: accessToken);
                logger.CLogInformation($"Dnc post nodeset RequestMessage uri: { requestMessage.RequestUri.ToString() }");

                HttpResponseMessage response = await this.httpUtility.SendAsync(requestMessage);
                if (response.StatusCode != HttpStatusCode.OK)
                {
                    DncFailureMessage errMsg = await this.ParseErrorResponse(response);
                    throw new ErrorResponseMessageException(
                                           httpStatus: response.StatusCode,
                                           errorCode: ErrorResponseCode.NodesetCreateFailed,
                                           errorMessage: $"DNC Put Nodeset failed to this endpoint: " +
                                           $"{dncEndpoint}, , response content: {errMsg.ToString()}");
                }
            }
        }

        public async Task DeleteNodesetAsync(
            string orchestratorId,
            bool forceDelete)
        {
            string accessToken = await GetDncAccessTokenAsync();
            // send dnc post request for subnet join
            string requestUri = string.Format(dncPutNodesetFormat, dncEndpoint, orchestratorId);

            if (forceDelete)
            {
                requestUri = requestUri + "&force-delete=true";
            }

            using (HttpRequestMessage requestMessage = new HttpRequestMessage(HttpMethod.Delete, requestUri))
            {
                requestMessage.Headers.Authorization = new AuthenticationHeaderValue(scheme: "Bearer", parameter: accessToken);
                logger.CLogInformation($"Dnc Post nodeset RequestMessage uri: { requestMessage.RequestUri.ToString() }");

                HttpResponseMessage response = await this.httpUtility.SendAsync(requestMessage);
                if (response.StatusCode != HttpStatusCode.Accepted)
                {
                    DncFailureMessage errMsg = await this.ParseErrorResponse(response);
                    throw new ErrorResponseMessageException(
                                           httpStatus: response.StatusCode,
                                           errorCode: ErrorResponseCode.NodesetDeleteFailed,
                                           errorMessage: $"DNC Delete Nodeset failed to this endpoint: " +
                                           $"{dncEndpoint}, , response content: {errMsg.ToString()}");
                }
            }
        }

        /// <summary>
        /// GetSubnetAsync - Calls dnc get subnet
        /// </summary>
        /// <param name="virtualNetworkId">vnet id</param>
        /// <param name="subnetName">subnet name</param>
        /// <returns></returns>
        public async Task<HttpResponseMessage> GetNodesetAsync(
           string orchestratorId)
        {
            string accessToken = await GetDncAccessTokenAsync();

            // http get call to get subnet from dnc state
            string requestUri = string.Format(dncGetNodesetFormat, dncEndpoint, orchestratorId);
            using (HttpRequestMessage requestMessage = new HttpRequestMessage(HttpMethod.Get, requestUri))
            {
                requestMessage.Headers.Authorization = new AuthenticationHeaderValue(scheme: "Bearer", parameter: accessToken);
                logger.CLogInformation($"DNC Get Nodeset RequestMessage: {  requestMessage.RequestUri.ToString() }");
                return await this.httpUtility.SendAsync(requestMessage);
            }
        }


        private async Task<string> GetDncAccessTokenAsync()
        {
            IAzureAD azureAD = new AzureAD();
            // get access token from dnc resource id to talk to gateway
            string accessToken = await azureAD.GetAccessToken(CurrentContext.ManagedIdentity,
                CurrentContext.ArmAccessIdentity.AppId,
                CurrentContext.AzureEnvironment);
            if (String.IsNullOrEmpty(accessToken))
            {
                throw new Exception($"Invalid accesstoken for resourceid: {CurrentContext.ArmAccessIdentity.AppId}");
            }

            return accessToken;
        }

        private async Task<DncFailureMessage> ParseErrorResponse(HttpResponseMessage response)
        {
            if (response.Content != null && response.StatusCode != System.Net.HttpStatusCode.NoContent)
            {
                // This API is only expecting response Content to be in Json format.
                if (response.Content is object &&
                    response.Content.Headers != null &&
                    response.Content.Headers.ContentType != null &&
                    response.Content.Headers.ContentType.MediaType == "application/json")
                {
                    var contentStream = await response.Content.ReadAsStringAsync();
                    // use the default settings
                    return JsonUtility.Deserialize<DncFailureMessage>(contentStream);
                }
            }

            return new DncFailureMessage { };
        }
    }
}
