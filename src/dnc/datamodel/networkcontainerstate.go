package datamodel

import (
	"dnc/api"
)

// NetworkContainerState - NetworkContainer state.
type NetworkContainerState int

const (
	// Invalid state
	Invalid NetworkContainerState = iota

	// Prepare - Prepare NC for allocation
	Prepare

	// Reserve - NC reserve processing pending
	Reserve

	// Associate - NC is pending association.
	Associate

	// Publish - NC is is pending publishing.
	Publish

	// Program - NC is is programmed on the host.
	Program

	// UnPublish - NC is pending unpublishing.
	UnPublish

	// Revert - NC goal state is reverted
	Revert

	// DisAssociate - NC is pending disassociation.
	DisAssociate

	// UnReserve - NC is pending unreserve
	UnReserve

	// Purge - NC is pending cleanup of the NC
	Purge

	// RefreshAssociation - Refresh the association info due to changes in network configuration
	RefreshAssociation

	// AllocateLocalIP - Allocate Local IP
	AllocateLocalIP

	// AllocateMAC - Allocate MAC address
	AllocateMAC

	// DeallocateMAC - Deallocate MAC address
	DeallocateMAC

	// PrepareSecondaryIpCleanup
	PrepareSecondaryIpCleanup
)

// IsNCCreateInProgress - True if nc is in creation states
func IsNCCreateInProgress(ncState NetworkContainerState) bool {
	return ncState == Prepare ||
		ncState == Reserve ||
		ncState == AllocateLocalIP ||
		ncState == AllocateMAC ||
		ncState == Associate ||
		ncState == RefreshAssociation ||
		ncState == Publish ||
		ncState == Program ||
		ncState == PrepareSecondaryIpCleanup
}

// IsDeleteInProgress - True if nc is in termination states
func IsDeleteInProgress(ncState NetworkContainerState) bool {
	return ncState == UnPublish ||
		ncState == DisAssociate ||
		ncState == UnReserve ||
		ncState == DeallocateMAC ||
		ncState == Purge
}

// PromoteContainerState - Promotes container to the next state
func PromoteContainerState(nc NetworkContainerEntity) *api.JsonErr {
	if nc.Config.FailureDetail.Code != 0 {
		switch nc.Config.State {
		case Reserve:
			fallthrough
		case UnReserve:
			nc.Config.State = Purge
		case AllocateLocalIP:
			fallthrough
		case AllocateMAC:
			fallthrough
		case Associate:
			fallthrough
		case DisAssociate:
			nc.Config.State = UnReserve
		case Publish:
			fallthrough
		case Program:
			nc.Config.State = UnPublish
		case UnPublish:
			nc.Config.State = DisAssociate
		case DeallocateMAC:
			nc.Config.State = UnReserve
		}

		nc.Config.FailureDetail = api.JsonErr{}
		nc.Config.Status = Pending

		if err := NcRepo.PersistNetworkContainerData(&nc); err != nil {
			return err
		}

		// Add request to the pending queue
		requestContext := RequestContext{
			Type:      NetworkContainerRequest,
			NetworkID: nc.Config.NetworkID,
			NodeID:    nc.Config.GetAssociationInfo().NodeID,
		}
		ncKey := GetNCKey(nc.Config.NetworkID, nc.Config.NetworkContainerID)
		return AddRequestToPendingQueue(ncKey, &requestContext)
	}

	return nil
}

func (networkContainerState NetworkContainerState) String() string {
	switch networkContainerState {
	case Invalid:
		return "Invalid"

	case Prepare:
		return "Prepare"

	case Reserve:
		return "Reserve"

	case Associate:
		return "Associate"

	case Publish:
		return "Publish"

	case Program:
		return "Program"

	case UnPublish:
		return "UnPublish"

	case Revert:
		return "Revert"

	case DisAssociate:
		return "DisAssociate"

	case UnReserve:
		return "UnReserve"

	case Purge:
		return "Purge"

	case RefreshAssociation:
		return "RefreshAssociation"

	case AllocateLocalIP:
		return "AllocateLocalIP"

	case AllocateMAC:
		return "AllocateMAC"

	case DeallocateMAC:
		return "DeallocateMAC"

	case PrepareSecondaryIpCleanup:
		return "PrepareSecondaryIpCleanup"
	}

	return "Unknown Status"
}
