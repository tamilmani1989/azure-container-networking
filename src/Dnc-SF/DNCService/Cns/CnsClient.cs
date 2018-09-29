using Cns.Data;
using DNCService;
using DNCService.API;
using NetworkManager;
using Newtonsoft.Json;
using Newtonsoft.Json.Linq;
using System;
using System.Linq;
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
        public DncConfiguration DncConfiguration { get; private set; }

        public CnsClient(DncConfiguration dncConfiguration)
        {
            this.DncConfiguration = dncConfiguration;
        }

        public async Task CreateNetworkContainer(
            NetworkContainerInfo networkContainerInfo)
        {
            string createNCUrl = string.Format(
                    DncConfiguration.CNSSettings.ConnectionStrings.CreateOrUpdateNetworkContainerURLFmt,
                    networkContainerInfo.NodeInterfaceID);
            CreateNetworkContainerRequest createNetworkContainerRequest =
                networkContainerInfo.GetCreateCnsNetworkContainerRequest();

            createNetworkContainerRequest.OrchestratorContext =
                new JRaw(networkContainerInfo.OrchestratorContext);
            ServiceEventSource.Current.Message("CreateNCUrl:{0}, body:{1}",
                createNCUrl,
                createNetworkContainerRequest);
            if (DncConfiguration.CnsSimulatorMode)
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
            CreateNetworkContainerResponse createNCResponse = null;

            try
            {
                createNCResponse = JsonConvert.DeserializeObject<CreateNetworkContainerResponse>(
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
            if (DncConfiguration.CnsSimulatorMode)
            {
                return;
            }
            string deleteNCUrl = string.Format(
                    DncConfiguration.CNSSettings.ConnectionStrings.DeleteNetworkContainerURLFmt,
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
            DeleteNetworkContainerResponse deleteNCResponse = null;
            try
            {
                deleteNCResponse = JsonConvert.DeserializeObject<DeleteNetworkContainerResponse>(
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
            if (DncConfiguration.CnsSimulatorMode)
            {
                return;
            }

            SetOrchestratorTypeRequest setOrchestratorTypeRequest = new SetOrchestratorTypeRequest(nodeInfoRequest.OrchestratorType);
            string setOrchestratorTypeUrl = string.Format(
                    DncConfiguration.CNSSettings.ConnectionStrings.SetOrchestratorTypeURLFmt,
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
    }
}