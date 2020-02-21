package nmagentcontracts

import (
	"encoding/json"
	"errors"
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
	VlanID           int      `json:"vlanId"`
}

func (ncRequest *CreateOrUpdateNetworkContainerRequest) String() string {
	ipAddressesString := ""
	if ncRequest.IPV4Addresses != nil {
		ipAddressesString = strings.Join(ncRequest.IPV4Addresses, ",")
	} else {
		ipAddressesString = "nil"
	}

	return fmt.Sprintf(
		"CreateOrUpdateNetworkContainerRequest:[IPV4Addresses:%s, SubnetName:%s, Version:%s, VirtualNetworkID:%s, VlanID:%d]",
		ipAddressesString,
		ncRequest.SubnetName,
		ncRequest.Version,
		ncRequest.VirtualNetworkID,
		ncRequest.VlanID)
}

// MarshalJSON - Custom Marshal handler to enforce validation
func (ncRequest CreateOrUpdateNetworkContainerRequest) MarshalJSON() ([]byte, error) {
	objectMap := make(map[string]interface{})

	if ncRequest.IPV4Addresses == nil || len(ncRequest.IPV4Addresses) == 0 {
		return nil, errors.New("IPV4Addresses field cannot be empty")
	}

	for _, ipAddress := range ncRequest.IPV4Addresses {
		if ipAddress == "" {
			return nil, errors.New("Empty IPAddress specified in IPV4Addresses")
		}
	}

	if ncRequest.SubnetName == "" {
		return nil, errors.New("Subnet field cannot be empty")
	}

	if ncRequest.Version == "" {
		return nil, errors.New("Version field cannot be empty")
	}

	if ncRequest.VirtualNetworkID == "" {
		return nil, errors.New("Network ID field cannot be empty")
	}

	objectMap["ipV4Addresses"] = ncRequest.IPV4Addresses
	objectMap["subnetName"] = ncRequest.SubnetName
	objectMap["version"] = ncRequest.Version
	objectMap["policies"] = ncRequest.Policies
	objectMap["virtualNetworkId"] = ncRequest.VirtualNetworkID
	objectMap["vlanId"] = ncRequest.VlanID

	return json.Marshal(objectMap)
}
