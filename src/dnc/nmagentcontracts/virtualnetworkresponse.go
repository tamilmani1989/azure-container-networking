package nmagentcontracts

import (
	"fmt"
)

// VirtualNetworkResponse - struct to hold info related to network configuration
type VirtualNetworkResponse struct {
	ResponseCode string   `json:"httpStatusCode"`
	CNetSpace    string   `json:"cnetSpace"`
	DNSServers   string   `json:"dnsServers"`
	Subnets      []Subnet `json:"subnets"`
	VNetSpace    string   `json:"vnetSpace"`
	VnetVersion  string   `json:"vnetVersion"`
}

func (virtualNetworkResponse *VirtualNetworkResponse) String() string {
	subnetsString := "null"
	if virtualNetworkResponse.Subnets != nil {
		for index, subnet := range virtualNetworkResponse.Subnets {
			if index > 0 && index < len(virtualNetworkResponse.Subnets)-1 {
				subnetsString += ","
			}

			subnetsString += subnet.String()
		}
	}

	return fmt.Sprintf(
		"VirtualNetworkResponse:[ResponseCode:%s, CNetSpace:%s, DNSServers:%s, Subnets:%s, VNetSpace:%s, VnetVersion:%s]",
		virtualNetworkResponse.ResponseCode,
		virtualNetworkResponse.CNetSpace,
		virtualNetworkResponse.DNSServers,
		subnetsString,
		virtualNetworkResponse.VNetSpace,
		virtualNetworkResponse.VnetVersion)
}

// NMAgentResponse - NMAgent response.
type NMAgentResponse struct {
	ResponseCode string `json:"httpResponseCode"`
}

//  DeleteNetworkContainerResponse - NMAgent delete network container response.
type DeleteNetworkContainerResponse struct {
	ResponseCode string `json:"httpStatusCode"`
	ErrorMessage string `json:"errorMessage"`
}
