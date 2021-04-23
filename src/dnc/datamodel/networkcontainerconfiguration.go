package datamodel

import (
	"dnc/api"
	"time"

	"github.com/Azure/azure-container-networking/cns"
)

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
	LocalIP               string
	MACAddress            string
	GreKey                uint16
	CreatedDateTime       time.Time
	StateTransitionLog    map[string]api.StateTransitionStats
}

func (ncConfiguration *NCConfiguration) LogStateCompletion(state NetworkContainerState) {
	if ncConfiguration.StateTransitionLog == nil {
		ncConfiguration.StateTransitionLog = make(map[string]api.StateTransitionStats)
	}

	ncConfiguration.StateTransitionLog[state.String()] = api.StateTransitionStats{
		State:       state.String(),
		CompletedOn: time.Now().UTC(),
	}
}

// GetAssociationInfo - Retrieves association info associated with NC.
func (ncConfiguration *NCConfiguration) GetAssociationInfo() api.AssociationInfo {
	if len(ncConfiguration.RequestV1.AssociationInfo.NodeID) != 0 {
		return ncConfiguration.RequestV1.AssociationInfo
	}
	return ncConfiguration.RequestV2.AssociationInfo
}

// GetAllowHostToNCCommunicationStatus - Retrieves association info associated with NC.
func (ncConfiguration *NCConfiguration) GetAllowHostToNCCommunicationStatus() bool {
	if len(ncConfiguration.RequestV1.AssociationInfo.NodeID) != 0 {
		return ncConfiguration.RequestV1.AllowHostToNCCommunication
	}
	return ncConfiguration.RequestV2.AllowHostToNCCommunication
}

// GetAllowNCToHostCommunicationStatus - Retrieves association info associated with NC.
func (ncConfiguration *NCConfiguration) GetAllowNCToHostCommunicationStatus() bool {
	if len(ncConfiguration.RequestV1.AssociationInfo.NodeID) != 0 {
		return ncConfiguration.RequestV1.AllowNCToHostCommunication
	}
	return ncConfiguration.RequestV2.AllowNCToHostCommunication
}

// Endpoint Policies - Retrieves endpoint policies that were called as part of the network container request -
func (ncConfiguration *NCConfiguration) GetEndpointPolicies() []cns.NetworkContainerRequestPolicies {
	if ncConfiguration.RequestV1.EndpointPolicies != nil {
		return ncConfiguration.RequestV1.EndpointPolicies
	}

	if ncConfiguration.RequestV2.EndpointPolicies != nil {
		return ncConfiguration.RequestV2.EndpointPolicies
	}
	return nil
}

// InstanceID is used for baremetal scenario which uses ncRequestV2. Should I add it in RequestV1 ?
func (ncConfiguration *NCConfiguration) GetInstanceID() string {
	return ncConfiguration.RequestV2.BmContainerPhysicalInstID
}
