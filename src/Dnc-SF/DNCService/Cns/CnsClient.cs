using Cns.Data;
using DNCService;
using DNCService.API;
using NetworkManager;
using NetworkManager.DataModel;
using Newtonsoft.Json;
using Newtonsoft.Json.Linq;
using NMAgent.Data;
using System;
using System.Collections.Generic;
using System.Linq;
using System.Net;
using System.Net.Http;
using System.Text;
using System.Threading.Tasks;

namespace Cns
{
    public class CnsClient
    {
        /// <summary>
        /// Configuration for DNC
        /// </summary>
        private DncConfiguration dncConfiguration;

        /// <summary>
        /// Manager to handle tags cache
        /// </summary>
        private TagsCacheManager tagsCacheManager;

        public CnsClient(
            DncConfiguration dncConfiguration,
            TagsCacheManager tagsCacheManager)
        {
            this.dncConfiguration = dncConfiguration;
            this.tagsCacheManager = tagsCacheManager;
        }

        public async Task CreateNetworkContainer(
            NetworkContainerInfo networkContainerInfo)
        {
            string createNCUrl = string.Format(
                    dncConfiguration.CNSSettings.ConnectionStrings.CreateOrUpdateNetworkContainerURLFmt,
                    networkContainerInfo.NodeInterfaceID);
            CreateNetworkContainerRequest createNetworkContainerRequest =
                networkContainerInfo.GetCreateCnsNetworkContainerRequest();

            createNetworkContainerRequest.OrchestratorContext =
                new JRaw(networkContainerInfo.OrchestratorContext);
            ServiceEventSource.Current.Message("CreateNCUrl:{0}, body:{1}",
                createNCUrl,
                createNetworkContainerRequest);
            if (dncConfiguration.CnsSimulatorMode)
            {
                return;
            }
            string createOrUpdateNCJson =
                JsonConvert.SerializeObject(createNetworkContainerRequest);
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
                throw new DncCnsException(
                    new ErrorInfo(ErrorCode.StatusInternalServerError,
                        $"Failed to post request Create NC to CNS due to exception {errorMsg}",
                        ErrorType.Retriable));
            }

            string responseBody = await response.Content.ReadAsStringAsync();
            Cns.Data.CreateNetworkContainerResponse createNCResponse = null;

            try
            {
                createNCResponse = JsonConvert.DeserializeObject<Cns.Data.CreateNetworkContainerResponse>(
                    responseBody);
            }
            catch (Exception exception)
            {
                throw new DncCnsException(
                    new ErrorInfo(ErrorCode.DecodeResponseFailed,
                        $"Failed to decode response {createNCResponse} due to exception {exception}",
                        ErrorType.Retriable));
            }

            if (createNCResponse.Response.ReturnCode != 0)
            {
                throw new DncCnsException(
                    new ErrorInfo(ErrorCode.AssociationFailed,
                        $"Failed association due to CNS returning return error code " +
                        $"{createNCResponse.Response.ReturnCode}",
                        ErrorType.NonRetriable));
            }

            ServiceEventSource.Current.Message($"NC created successfully:{createNCResponse}");
            return;

        }

        public async Task DeleteNetworkContainer(
            NetworkContainerInfo networkContainerInfo)
        {
            if (dncConfiguration.CnsSimulatorMode)
            {
                return;
            }
            string deleteNCUrl = string.Format(
                    dncConfiguration.CNSSettings.ConnectionStrings.DeleteNetworkContainerURLFmt,
                    networkContainerInfo.NodeInterfaceID);
            DeleteNetworkContainerRequest deleteNetworkContainerRequest = 
                new DeleteNetworkContainerRequest(networkContainerInfo.NetworkContainerID);
            ServiceEventSource.Current.Message("DeleteNCUrl:{0}, body:{1}",
                deleteNCUrl,
                deleteNetworkContainerRequest);
            string deleteNCJson =
                JsonConvert.SerializeObject(deleteNetworkContainerRequest);
            var httpContent = new StringContent(deleteNCJson, Encoding.UTF8, "application/json");
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
                throw new DncCnsException(
                    new ErrorInfo(ErrorCode.StatusInternalServerError,
                        $"Failed to post request Delete NC to CNS due to exception {errorMsg}",
                        ErrorType.Retriable));
            }

            string responseBody = await response.Content.ReadAsStringAsync();
            Cns.Data.DeleteNetworkContainerResponse deleteNCResponse = null;
            try
            {
                deleteNCResponse = JsonConvert.DeserializeObject<Cns.Data.DeleteNetworkContainerResponse>(
                    responseBody);
            }
            catch (Exception exception)
            {
                throw new DncCnsException(
                    new ErrorInfo(ErrorCode.DecodeResponseFailed,
                        $"Failed to decode response {deleteNCResponse} due to exception {exception}",
                        ErrorType.Retriable));
            }

            if (deleteNCResponse.Response.ReturnCode != 0)
            {
                throw new DncCnsException(
                    new ErrorInfo(ErrorCode.DisAssociationFailed,
                        $"Failed disassociation due to CNS returning return error code " +
                        $"{deleteNCResponse.Response.ReturnCode}",
                        ErrorType.NonRetriable));
            }

            ServiceEventSource.Current.Message($"NC deleted successfully:{deleteNCResponse}");
            return;

        }

        public async Task SetOrchestratorType(NodeRequest nodeInfoRequest)
        {
            if (dncConfiguration.CnsSimulatorMode)
            {
                return;
            }

            SetOrchestratorTypeRequest setOrchestratorTypeRequest = new SetOrchestratorTypeRequest(nodeInfoRequest.OrchestratorType);
            string setOrchestratorTypeUrl = string.Format(
                    dncConfiguration.CNSSettings.ConnectionStrings.SetOrchestratorTypeURLFmt,
                    nodeInfoRequest.IPAddresses.First());
            ServiceEventSource.Current.Message("setOrchestratorTypeUrl:{0}, body:{1}",
                setOrchestratorTypeUrl,
                setOrchestratorTypeRequest);
            string setOrchestratorJsonRequest =
                JsonConvert.SerializeObject(setOrchestratorTypeRequest);
            var httpContent = new StringContent(setOrchestratorJsonRequest, Encoding.UTF8, "application/json");
            HttpClient httpClient = new HttpClient();
            try
            {
                HttpResponseMessage response = await httpClient.PostAsync(setOrchestratorTypeUrl, httpContent);
            }
            catch (Exception exception)
            {
                string errorMsg = (exception.InnerException == null) ?
                    exception.Message : exception.InnerException.Message;
                throw new DncCnsException(
                    new ErrorInfo(ErrorCode.StatusInternalServerError,
                        $"Failed to post request for orchestrator type to CNS due to exception {errorMsg}",
                        ErrorType.Retriable));
            }
        }

        public void ParseTagsIntoCnetAddresses(
            ref NetworkContainerInfo networkContainerInfo,
            ref NCConfiguration ncConfig)
        {
            Dictionary<string, UInt64> latestTagInfo = new Dictionary<string, UInt64>();
            List<TagDefinition> resolvedTags = new List<TagDefinition>();
            bool tagResolveError = false;
            foreach (Tag tag in networkContainerInfo.Tags)
            {
                TagDefinition tagDefinition = tagsCacheManager.RegisterAndGetTagDefinition(tag);
                if (tagDefinition != null)
                {
                    resolvedTags.Add(tagDefinition);
                    latestTagInfo.Add(Tag.GetTagKey(tag), tagDefinition.Version);
                }
                else
                {
                    tagResolveError = true;
                }
            }

            if (tagResolveError)
            {
                ServiceEventSource.Current.Message($"Error resolving tag");
                    throw new DncManagementException(
                        new ErrorInfo(ErrorCode.GetTagDefinitionFailed,
                        $"Encountered error resolving tag",
                        (int)HttpStatusCode.InternalServerError,
                        ErrorType.Retriable));
            }

            if ((ncConfig.ProgrammedVnetVersion != networkContainerInfo.VnetVersion) ||
                TagsCacheManager.AreTagsUpdated(latestTagInfo, ncConfig.ProgrammedTagsInfo))
            {
                ++ncConfig.CNSVersion;
                networkContainerInfo.UpdateCnsVersion();
                ServiceEventSource.Current.Message($"Updating CNSVersion to {ncConfig.CNSVersion}");
            }

            // TODO: Optimization - Store the resolved CnetAddresses in NCConfig. It can be reused if the
            // tags are not updated. That avoids performing below operation every time.

            List<Cns.Data.IPSubnet> cnsCnetAddresses = new List<Cns.Data.IPSubnet>();
            List<string> cnetAddresses = new List<string>();

            foreach (string cnetAddress in networkContainerInfo.CnetAddressSpace)
            {
                cnetAddresses.Add(cnetAddress);
            }

            foreach (TagDefinition tagDefinition in resolvedTags)
            {
                foreach (string tagAddress in tagDefinition.Values)
                {
                    cnetAddresses.Add(tagAddress);
                }
            }

            foreach (string cnetAddress in cnetAddresses)
            {
                try
                {
                    NetworkManager.IPSubnet cnsAddress =
                        NetworkManager.IPSubnet.Parse(cnetAddress);
                    cnsCnetAddresses.Add(
                        new Data.IPSubnet(cnsAddress.IPAddress.ToString(),
                            cnsAddress.Prefix));
                }
                catch (Exception)
                {
                    ServiceEventSource.Current.Message($"ParseTagsIntoCnetAddresses " +
                        $"parsing failed for cnet address: {cnetAddress}");

                    throw new DncManagementException(
                        new ErrorInfo(ErrorCode.InvalidIPSubnet,
                            $"Failed to parse invalid CnetAddress {cnetAddress}",
                            (int)HttpStatusCode.InternalServerError,
                            ErrorType.NonRetriable));
                }
            }

            // Update cnsCnetAddress in networkContainerInfo
            networkContainerInfo.SetCnsCnetAddresses(cnsCnetAddresses);

            // Update tag versions and vnet version in NC configuration
            ncConfig.UpdateProgrammedTagsInfo(latestTagInfo);
            ncConfig.ProgrammedVnetVersion = networkContainerInfo.VnetVersion;
        }
    }
}