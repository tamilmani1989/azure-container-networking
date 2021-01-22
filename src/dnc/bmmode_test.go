// +build bmMode

package main

import (
	"bytes"
	"dnc/api"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/Azure/azure-container-networking/cns"
)

var (
	bmnodeReq = api.NodeInfoRequest{
		IPAddresses:      []string{"127.0.0.1"},
		NodeType:         "BareMetal",
		OrchestratorType: cns.ServiceFabric,
		AZID:             "eastus2euap-az01",
	}

	bmnodeReqEmptyAZID = api.NodeInfoRequest{
		IPAddresses:      []string{"127.0.0.1"},
		NodeType:         "BareMetal",
		OrchestratorType: cns.ServiceFabric,
		AZID:             "",
	}

	bmnodeID = "dadd9e8c-4e5c-4576-a8ae-0a8861af4bea"

	nodeID  = "testnode"
	nodeReq = api.NodeInfoRequest{
		IPAddresses:      []string{"127.0.0.1"},
		LocationType:     "",
		OrchestratorType: cns.Kubernetes,
	}
)

// Setup :- Set up the environment (if necessary).
func Setup() error {
	go StartDnc(ReadConfig())
	time.Sleep(1 * time.Second)
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 1024
	http.DefaultTransport.(*http.Transport).TLSHandshakeTimeout = 0 * time.Second

	httpClient = http.DefaultClient
	return nil
}

func TestBMMode_CreateNodeWithGUIDValue(t *testing.T) {
	addOrUpdateNodeOp(t, dncEP, bmnodeID, apiVersion, &bmnodeReq, http.StatusOK)
}

func TestBMMode_CreateNodeWithNonGUIDValue(t *testing.T) {
	addOrUpdateNodeOp(t, dncEP, nodeID, apiVersion, &bmnodeReq, http.StatusBadRequest)
}

func TestBMMode_CreateNodeWithAZID(t *testing.T) {
	addOrUpdateNodeOp(t, dncEP, bmnodeID, apiVersion, &bmnodeReq, http.StatusOK)
}

func TestBMMode_CreateNodeWithEmptyAZID(t *testing.T) {
	addOrUpdateNodeOp(t, dncEP, bmnodeID, apiVersion, &bmnodeReqEmptyAZID, http.StatusBadRequest)
}

func addOrUpdateNodeOp(
	t *testing.T,
	dncEP string,
	nodeID string,
	apiVersion string,
	nodeInfoReq *api.NodeInfoRequest,
	expectedHttpReturnCode int) {
	var body bytes.Buffer
	var err error
	var response *http.Response

	json.NewEncoder(&body).Encode(nodeInfoReq)
	fmt.Printf("NodeID: %s, NodeInfo: %s", nodeID, nodeInfoReq.String())
	response, err = http.Post(dncEP+"/nodes/"+nodeID+apiVersion, "application/json", &body)
	if err != nil {
		t.Fatalf("CreateNode failed with error: %+v\n", err)
	}

	defer response.Body.Close()

	if response.StatusCode != expectedHttpReturnCode {
		t.Fatalf("Return code CreateNode request doesn't match Expected: %d, Observed: %d",
			expectedHttpReturnCode,
			response.StatusCode)
	}
}
