package cns

import (
	"bytes"
	"dnc/api"
	"dnc/configuration"
	"dnc/utils"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Azure/azure-container-networking/cns"
)

// Client is a client for the CNS HTTP API
type Client struct {
	Paths configuration.CNSConnectionStrings
	HTTP  *http.Client
}

// todo:
//  - do CNS routes ever change? can we hardcode them?
//  - add other CNS methods

const jsonContentType = "application/json"

func (c *Client) DeleteNetworkContainer(nodeAddress, ncID string) (cns.DeleteNetworkContainerResponse, error) {
	deleteReq := cns.DeleteNetworkContainerRequest{NetworkContainerid: cns.SwiftPrefix + ncID}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(deleteReq); err != nil {
		return cns.DeleteNetworkContainerResponse{}, err
	}

	deleteURL := fmt.Sprintf(c.Paths.DeleteNetworkContainerURLFmt, nodeAddress)
	res, err := c.HTTP.Post(deleteURL, jsonContentType, &body)
	if err != nil {
		return cns.DeleteNetworkContainerResponse{}, err
	}
	defer res.Body.Close()

	var delResp cns.DeleteNetworkContainerResponse
	if err := json.NewDecoder(res.Body).Decode(&delResp); err != nil {
		return cns.DeleteNetworkContainerResponse{}, err
	}
	return delResp, nil
}

func (c *Client) SetOrchestratorType(nodeID, nodeAddress, orchestratorType string) *api.JsonErr {
	var (
		info = &cns.SetOrchestratorTypeRequest{
			OrchestratorType: orchestratorType,
			NodeID:           nodeID}
		body bytes.Buffer
		resp cns.Response
	)

	json.NewEncoder(&body).Encode(info)
	setOrchestratorTypeURL := fmt.Sprintf(c.Paths.SetOrchestratorTypeURLFmt, nodeAddress)
	w, respErr := c.HTTP.Post(setOrchestratorTypeURL, jsonContentType, &body)
	if respErr != nil {
		return api.InternalServerErrorf(api.NodeProcessingFailedDueToInternalError,
			"Post to URL %v for node %s failed. Error: %v", setOrchestratorTypeURL, nodeID, respErr)
	}

	defer w.Body.Close()

	if respErr = utils.DecodeResponse(w, &resp); respErr != nil {
		return api.InternalServerErrorf(api.NodeProcessingFailedDueToInternalError,
			"SetOrchestratorType for node %s failed. Error: %v", nodeID, respErr)
	}

	if resp.ReturnCode != 0 {
		return api.BadRequestf(api.NodeProcessingFailedDueToInternalError,
			"SetOrchestratorType for node %s failed with code: %d. Error: %v", nodeID, resp.ReturnCode, respErr)
	}

	return nil
}
