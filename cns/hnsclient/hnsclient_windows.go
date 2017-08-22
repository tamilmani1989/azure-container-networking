// Copyright 2017 Microsoft. All rights reserved.
// MIT License

package hnsclient

import (
	"github.com/Azure/azure-container-networking/log"
	"github.com/Microsoft/hcsshim"
	"fmt"
	"net"
)

// executeHNSRequest calls HNS to execute the method passed in the parameter.
func executeHNSRequest(method string, path string, request string) error {
	log.Printf("[Azure CNS] executeHNSRequest")
	return nil
}

// attachHNSEndPoint attaches an endpoint.
func (hnsClient *HnsClient) attachHNSEndPoint(hnsEndpoint hcsshim.HNSEndpoint) (error) {
	log.Printf("[Azure CNS] AttachHNSEndPoint")	
	return nil
}

// detachHNSEndPoint detaches an endpoint.
func (hnsClient *HnsClient) detachHNSEndPoint(id string) (error) {
	log.Printf("[Azure CNS] detachHNSEndPoint")	
	return nil
}

// GetContainerNetworkID retrieves the network id for the networkType (e.g., "Overlay").
func (hnsClient *HnsClient) GetContainerNetworkID(networkType string) (string, error) {
	log.Printf("[Azure CNS] GetContainerNetworkID")

	hnsNetworks, err := hcsshim.HNSListNetworkRequest("GET", "", "")	
	if err != nil {
		return "", err
	}

	for _, network := range hnsNetworks {
		if(network.Type == "Overlay") {
			return network.Id, nil
		}
	}

	return "", fmt.Errorf("[Azure CNS] HNS does not know about any overlay network")
}

// AddHNSEndPoint adds an endpoint.
func (hnsClient *HnsClient) AddHNSEndPoint(networkType string, vnetID string, ipaddress net.IP) (error) {
	log.Printf("[Azure CNS] AddHNSEndPoint")

	hnsEndpoint := &hcsshim.HNSEndpoint{
		VirtualNetwork: vnetID,
		IPAddress: ipaddress,
	}
	
	_, err := hnsEndpoint.Create()
	
	return err
}

// DeleteHNSEndPoint deletes an endpoint. Detach first if its host local.
func (hnsClient *HnsClient) DeleteHNSEndPoint(endpointID string) (error) {
	log.Printf("[Azure CNS] DeleteHNSEndPoint")	
	
	hnsEndpoint := &hcsshim.HNSEndpoint{
		Id:endpointID,
	}
	
	_, err := hnsEndpoint.Delete()
	return err
}