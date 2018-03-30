package api

import (
	"fmt"
	"strings"
)

// IPConfiguration - Struct to hold the ip configuration.
type IPConfiguration struct {
	NetworkID  string   `json:"NetworkID"`
	SubnetName string   `json:"SubnetName"`
	IPSubnet   string   `json:"IPSubnet"`
	IPAddress  string   `json:"IPAddress"`
	GatewayIP  string   `json:"GatewayIP"`
	DNSServers []string `json:"DnsServers"`
}

func (ipConfiguration *IPConfiguration) String() string {
	return fmt.Sprintf(
		"IPConfiguration:[NetworkID:%s, SubnetName: %s IPSubnet: %s IPAsdress:%s GatewayIP:%s, DNSServers:%s]",
		ipConfiguration.NetworkID, ipConfiguration.SubnetName, ipConfiguration.IPSubnet,
		ipConfiguration.IPAddress, ipConfiguration.GatewayIP, strings.Join(ipConfiguration.DNSServers, ","))
}
