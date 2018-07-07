package datamodel

import "dnc/api"

// NCConfiguration - struct to hold the network container configuration.
type NCConfiguration struct {
	NetworkContainerID    string
	NetworkID             string
	ReservationID         string
	ImplicitReservation   bool
	RequestV1             api.NetworkContainerRequestV1
	RequestV2             api.NetworkContainerRequestV2
	State                 NetworkContainerState
	Status                Status
	CNSVersion            uint64
	AgentVersion          uint64
	ProgrammedVnetVersion uint64
	ProgrammedTagsInfo    map[string]uint64 // Tag version info
	FailureDetail         api.JsonErr
	MarkedForDelete       bool
	APIVersion            string
	LocalIP				  string
}

// GetAssociationInfo - Retrieves association info assocuated with NC.
func (ncConfiguration *NCConfiguration) GetAssociationInfo() api.AssociationInfo {
	if len(ncConfiguration.RequestV1.AssociationInfo.NodeID) != 0 {
		return ncConfiguration.RequestV1.AssociationInfo
	}
	return ncConfiguration.RequestV2.AssociationInfo
}
