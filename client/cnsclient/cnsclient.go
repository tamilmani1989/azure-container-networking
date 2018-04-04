package cnsclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Azure/azure-container-networking/cns"
	"github.com/Azure/azure-container-networking/log"
)

// IpamClient specifies a client to connect to Ipam Plugin.
type CNSClient struct {
	connectionURL string
}

const (
	defaultCnsURL = "http://localhost:10090"
)

// NewCnsClient create a new cns client.
func NewCnsClient(url string) (*CNSClient, error) {
	if url == "" {
		url = defaultCnsURL
	}
	return &CNSClient{
		connectionURL: url,
	}, nil
}

// GetNetworkConfiguration Request to get network config.
func (cnsClient *CNSClient) GetNetworkConfiguration(podName, podNamespace string) (*cns.GetNetworkConfigResponse, error) {
	var body bytes.Buffer
	httpc := &http.Client{}

	url := cnsClient.connectionURL + cns.GetNetworkConfigByOrchestratorInfo
	log.Printf("GetNetworkConfiguration url %v", url)

	azContainerInfo := cns.AzureContainerInstanceInfo{PodName: podName, PodNamespace: podNamespace}
	azContainerInfoBytes, err := json.Marshal(azContainerInfo)
	if err != nil {
		log.Printf("Marshalling azure container instance info failed with %v", err)
		return nil, err
	}

	ncOrchestratorInfo := cns.OrchestratorInfo{OrchestratorType: cns.AzureContainerInstance, OrchestratorContext: azContainerInfoBytes}

	payload := &cns.GetNetworkConfigRequest{
		OrchestratorInfo: ncOrchestratorInfo,
	}

	err = json.NewEncoder(&body).Encode(payload)
	if err != nil {
		log.Printf("encoding json failed with %v", err)
		return nil, err
	}

	res, err := httpc.Post(url, "application/json", &body)
	if err != nil {
		log.Printf("[Azure CNSClient] HTTP Post returned error %v", err.Error())
		return nil, err
	}

	if res.StatusCode == 200 {
		var resp cns.GetNetworkConfigResponse
		err := json.NewDecoder(res.Body).Decode(&resp)
		if err != nil {
			log.Printf("[Azure CNSClient] Error received while parsing GetNetworkConfiguration response resp:%v err:%v", res.Body, err.Error())
			return nil, err
		}

		if resp.Response.ReturnCode != 0 {
			log.Printf("[Azure CNSClient] GetNetworkConfiguration received error response :%v", resp.Response.Message)
			return nil, fmt.Errorf(resp.Response.Message)
		}

		return &resp, nil
	}

	var errMsg string
	errMsg = fmt.Sprintf(errMsg, "[Azure CNSClient] GetNetworkConfiguration invalid http status code: %v", res.StatusCode)
	log.Printf(errMsg)
	return nil, fmt.Errorf(errMsg)
}
