package datamodel

// NetworkContainerState - NetworkContainer state.
type NetworkContainerState int

const (
	// Prepare - Prepare NC for allocation
	Prepare NetworkContainerState = iota + 1

	// Reserve - NC resevrve processing pending
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
)

func (networkContainerState NetworkContainerState) String() string {
	switch networkContainerState {

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
	}

	return "Unknown Status"
}
