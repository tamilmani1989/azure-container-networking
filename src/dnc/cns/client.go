package cns

import (
	"bytes"
	"dnc/configuration"
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