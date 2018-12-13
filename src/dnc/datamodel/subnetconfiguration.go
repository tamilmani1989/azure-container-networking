package datamodel

import (
	"crypto/sha256"
	"dnc/nmagentcontracts"
	"dnc/utils"
	"fmt"
)

// SubnetUpdateStatus  - type indicating the update status for the subnet.
type SubnetUpdateStatus int

// SubnetUpdateStatus enum indicating the update status for the subnet.
const (
	SubnetUpdateCompleted SubnetUpdateStatus = iota
	SubnetUpdateFailed
	SubnetUpdatePending
)

// SubnetConfiguration  - Struct to hold subnet configuration.
type SubnetConfiguration struct {
	NetworkID         string
	Reservations      map[string]string //Key:ReservationId: Value:Allocated IP
	SubnetName        string
	IPSubnet          string
	IsDelegatedSubnet bool
	Gateway           string
	AccessToken       string
	Joined            bool
	Tags              []nmagentcontracts.Tag
	APIVersion        string
	UpdateStatus      SubnetUpdateStatus
}

func (s *SubnetConfiguration) String() string {
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
			"IsDelegatedSubnet:%t Gateway:%s AccessToken:%s, Joined:%t, "+
			"Servicetags:%s APIVersion:%s]",
		s.NetworkID, reservations, s.SubnetName, s.IPSubnet, s.IsDelegatedSubnet,
		s.Gateway, sha256.Sum256([]byte(s.AccessToken)), s.Joined, serviceTags, s.APIVersion)
}
