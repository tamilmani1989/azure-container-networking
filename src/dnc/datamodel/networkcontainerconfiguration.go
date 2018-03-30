package datamodel

import "dnc/api"

// NCConfiguration - struct to hold the network container configuration.
type NCConfiguration struct {
	NetworkContainerID  string
	Version             uint64
	NetworkID           string
	ReservationID       string
	ImplicitReservation bool
	RequestV1           api.NetworkContainerRequestV1
	RequestV2           api.NetworkContainerRequestV2
	State               NetworkContainerState
	Status              Status
	FailureDetail       api.JsonErr
	MarkedForDelete     bool
	APIVersion          string
}

// GetAssociationInfo - Retrieves association info assocuated with NC.
func (ncConfiguration *NCConfiguration) GetAssociationInfo() api.AssociationInfo {
	if len(ncConfiguration.RequestV1.AssociationInfo.NodeID) != 0 {
		return ncConfiguration.RequestV1.AssociationInfo
	}
	return ncConfiguration.RequestV2.AssociationInfo
}
