package api

import (
	"fmt"
)

const (
	// AzureNet - Constant to identify azure network
	AzureNet = "AzureNet"
)

// NetworkInfoRequest - Network info.
type NetworkInfoRequest struct {
	NetworkType  string `json:"NetworkType"`
	NetworkSpace string `json:"NetworkSpace"` // Optional, not required for azure vnets
}

func (networkInfoRequest *NetworkInfoRequest) String() string {
	return fmt.Sprintf(
		"NetworkInfoRequest:[NetworkType:%s NetworkSpace:%s]",
		networkInfoRequest.NetworkType, networkInfoRequest.NetworkSpace)
}

// Validate - Validate network info request.
func (networkInfoRequest *NetworkInfoRequest) Validate() *JsonErr {
	if networkInfoRequest.NetworkType != AzureNet {
		return &JsonErr{
			Code: UnSupprotedNetworkType,
			Text: fmt.Sprintf("Network type is not supported. Supported network types are %s", AzureNet)}
	}

	return nil
}
