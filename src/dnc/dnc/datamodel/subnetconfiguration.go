package datamodel

import (
	"dnc/nmagentcontracts"
	"dnc/utils"
	"fmt"
	"strings"
)

// SubnetConfiguration  - Struct to hold subnet configuration.
type SubnetConfiguration struct {
	NetworkID         string
	Reservations      map[string]string //Key:ReservationId: Value:Allocated IP
	SubnetName        string
	IPSubnet          string
	IsDelegatedSubnet bool
	DNSServers        []string
	Gateway           string
	AccessToken       string
	Joined            bool
	Tags              []nmagentcontracts.Tag
	APIVersion        string
}

func (s *SubnetConfiguration) String() string {
	dnsServers := strings.Join(s.DNSServers, ",")
	reservations := ""
	if s.Reservations != nil {
		reservations = utils.PrintMapKeys(s.Reservations)
	}
	serviceTags := ""
	if s.Tags != nil {
		for index, tag := range s.Tags {
			serviceTags += tag.String()
			if index != 0 && index < (len(s.Tags)-1) {
				serviceTags += ","
			}
		}
	} else {
		serviceTags = "nil"
	}

	return fmt.Sprintf(
		"[NetworkID:%s, Reservations:%s SubnetName:%s IPSubnet:%s "+
			"IsDelegatedSubnet:%t DNSServers:%s "+
			"Gateway:%s AccessToken:%s, Joined:%t, Servicetags:%s APIVersion:%s]",
		s.NetworkID, reservations, s.SubnetName, s.IPSubnet,
		s.IsDelegatedSubnet, dnsServers,
		s.Gateway, s.AccessToken, s.Joined, serviceTags, s.APIVersion)
}
