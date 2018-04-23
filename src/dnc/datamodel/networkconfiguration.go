package datamodel

import (
	"dnc/api"
	"dnc/utils"
	"fmt"
	"strings"
)

// NetworkConfiguration - struct to hold info related to network configuration
type NetworkConfiguration struct {
	NetworkID  string
	Version    uint64
	CNetSpace  string
	VNetSpace  string
	Subnets    map[string]string // Key:SubnetName Value:SubnetKey
	DNSServers []string
	NetworkState
	Status
	FailureDetail api.JsonErr
	APIVersion    string
}

func (v *NetworkConfiguration) String() string {
	var dnsServers string
	if len(v.DNSServers) != 0 {
		dnsServers = strings.Join(v.DNSServers, ",")
	} else {
		dnsServers = ""
	}
	return fmt.Sprintf(
		"[NetworkID:%s	Version:{%d} CNetSpace:%s VNetSpace:%s "+
			"Subnets:%s DNSServers:%s, NetworkState:%s Status:%s FailureDetail:%s, APIVersion:%s]",
		v.NetworkID, v.Version, v.CNetSpace, v.VNetSpace,
		utils.PrintMapKeys(v.Subnets), dnsServers, v.NetworkState.String(), v.Status.String(),
		v.FailureDetail.String(), v.APIVersion)

}
