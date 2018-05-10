package nmagentcontracts

import (
	"fmt"
	"strings"
)

// CreateOrUpdateNetworkContainerRequest - struct to hold info for create or update newtork container
type CreateOrUpdateNetworkContainerRequest struct {
	IPV4Addresses    []string `json:"ipV4Addresses"`
	SubnetName       string   `json:"subnetName"`
	Version          string   `json:"version"`
	Policies         []string `json:"policies"`
	VirtualNetworkID string   `json:"virtualNetworkId"`
}

func (ncRequest *CreateOrUpdateNetworkContainerRequest) String() string {
	ipAddressesString := ""
	if ncRequest.IPV4Addresses != nil {
		ipAddressesString = strings.Join(ncRequest.IPV4Addresses, ",")
	} else {
		ipAddressesString = "nil"
	}

	return fmt.Sprintf(
		"CreateOrUpdateNetworkContainerRequest:[IPV4Addresses:%s, SubnetName:%s, Version:%s, VirtualNetworkID:%s]",		
		ipAddressesString,
		ncRequest.SubnetName,
		ncRequest.Version,
		ncRequest.VirtualNetworkID)
}
