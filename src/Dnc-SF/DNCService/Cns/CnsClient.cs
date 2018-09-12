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
            HttpResponseMessage response = await httpClient.PostAsync(createNCUrl, httpContent);
            response.EnsureSuccessStatusCode();

            string responseBody = await response.Content.ReadAsStringAsync();
            CreateNetworkContainerResponse createNCReponse = null;
            try
            {
                createNCReponse = JsonConvert.DeserializeObject<CreateNetworkContainerResponse>(
                    responseBody);
            }
            catch (Exception exception)
            {
                throw new Exception(
                    $"Failed to decode response {createNCReponse} due to exception {exception}");
            }

            if (createNCReponse.ReturnCode != 0)
            {
                throw new Exception(
                    $"CreateNC failed. Received response:{createNCReponse}");
            }

            ServiceEventSource.Current.Message($"NC created successfully:{createNCReponse}");
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
            HttpResponseMessage response = await httpClient.PostAsync(deleteNCUrl, httpContent);
            response.EnsureSuccessStatusCode();

            string responseBody = await response.Content.ReadAsStringAsync();
            DeleteNetworkContainerResponse deleteNCResponse = null;
            try
            {
                deleteNCResponse = JsonConvert.DeserializeObject<DeleteNetworkContainerResponse>(
                    responseBody);
            }
            catch (Exception exception)
            {
                throw new Exception(
                    $"Failed to decode response {deleteNCResponse} due to exception {exception}");
            }

            if (deleteNCResponse.ReturnCode != 0)
            {
                throw new Exception(
                    $"DeleteNC failed. Received response:{deleteNCResponse}");
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
            HttpResponseMessage response = await httpClient.PostAsync(setOrchestratorTypeUrl, httpContent);
            response.EnsureSuccessStatusCode();
        }
    }
}