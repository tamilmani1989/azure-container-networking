package api

import (
	"fmt"
)

// NetworkConfigurationResponse - Struct to hold the network configuration response.
type NetworkConfigurationResponse struct {
	DelegatedSubnets []string
}

func (n *NetworkConfigurationResponse) String() string {
	delegatedSubnets := "DelegatedSubnets:["
	if n.DelegatedSubnets != nil {
		for _, subnet := range n.DelegatedSubnets {
			delegatedSubnets += subnet
		}
	}
	delegatedSubnets += "]"

	return fmt.Sprintf(
		"NetworkConfigurationResponse:[DelegatedSubnets:%s]", delegatedSubnets)
}
