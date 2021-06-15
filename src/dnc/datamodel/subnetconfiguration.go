package datamodel

import (
	"dnc/api"
	"dnc/nmagentcontracts"
)

// SubnetUpdateStatus  - type indicating the update status for the subnet.
type SubnetUpdateStatus int

// SubnetUpdateStatus enum indicating the update status for the subnet.
const (
	SubnetUpdateCompleted SubnetUpdateStatus = iota
	SubnetUpdateFailed
	SubnetUpdatePending
	SubnetCleanupPending
	SubnetCleanupFailed
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
	MarkedForDelete   bool
	Tags              []nmagentcontracts.Tag
	APIVersion        string
	UpdateStatus      SubnetUpdateStatus
	Version           uint64
	FailureDetail     api.JsonErr
}