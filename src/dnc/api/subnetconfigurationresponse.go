package api

import (
	"fmt"
	"strings"
)

// SubnetConfigurationResponse - Struct to hold the subnet configuration.
type SubnetConfigurationResponse struct {
	NetworkID         string            `json:"NetworkID"`
	SubnetName        string            `json:"SubnetName"`
	IPSubnet          string            `json:"IPSubnet"`
	GatewayIP         string            `json:"GatewayIP"`
	DNSServers        []string          `json:"DnsServers"`
	AccessToken       string            `json:"AccessToken"`
	IPAllocationsData IPAllocationsData `json:"IPAllocationsData"`
}

func (subnetConfigurationResponse *SubnetConfigurationResponse) String() string {
	return fmt.Sprintf(
		"SubnetConfiguration:[NetworkID:%s, SubnetName: %s IPSubnet: %s GatewayIP:%s, DNSServers:%s, %s]",
		subnetConfigurationResponse.NetworkID, subnetConfigurationResponse.SubnetName, subnetConfigurationResponse.IPSubnet,
		subnetConfigurationResponse.GatewayIP, strings.Join(subnetConfigurationResponse.DNSServers, ","),
		subnetConfigurationResponse.IPAllocationsData.String())
}
