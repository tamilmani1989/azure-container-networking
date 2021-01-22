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

func CompareSubnetConfigurationResponse(a, b SubnetConfigurationResponse) bool {
	if a.NetworkID == b.NetworkID &&
		a.SubnetName == b.SubnetName &&
		a.IPSubnet == b.IPSubnet &&
		a.GatewayIP == b.GatewayIP &&
		a.AccessToken == b.AccessToken &&
		len(a.DNSServers) == len(b.DNSServers) &&
		len(a.IPAllocationsData) == len(b.IPAllocationsData) {
		// Check if the slices have the same elements
		dnsServersA := make(map[string]struct{})
		for _, s := range a.DNSServers {
			dnsServersA[s] = struct{}{}
		}

		for _, s := range b.DNSServers {
			if _, ok := dnsServersA[s]; !ok {
				return false
			}
		}

		ipAllocDataA := make(map[string]IPAllocationData)
		for _, d := range a.IPAllocationsData {
			ipAllocDataA[d.ReservationID] = d
		}

		for _, allocDataB := range b.IPAllocationsData {
			allocDataA, ok := ipAllocDataA[allocDataB.ReservationID]
			if !ok {
				return false
			}

			if allocDataB.IPAddress != allocDataA.IPAddress ||
				allocDataB.AssociatedNetworkContainerID != allocDataA.AssociatedNetworkContainerID {
				return false
			}
		}

		return true
	}

	return false
}
