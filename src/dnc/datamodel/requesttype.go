package datamodel

// RequestType - Type of the request
type RequestType int

const (
	// InvalidRequest - Not Used
	InvalidRequest RequestType = iota

	// ReservationRequest - Reservation request
	ReservationRequest

	// NetworkContainerRequest - Network container request
	NetworkContainerRequest

	// VirtualNetworkRequest - Request to retrieve virtual network.
	VirtualNetworkRequest

	// ReservationSetRequest - Reservation set request
	ReservationSetRequest

	// SubnetCleanupRequest - Request to cleanup subnet. It attempts to delete all allocations in the subnet
	// and then deletes the subnet
	SubnetCleanupRequest

	// NodeSetCleanupRequest - Request to cleanup nodeset.
	NodeSetCleanupRequest
)

func (requestType RequestType) String() string {
	switch requestType {
	case ReservationRequest:
		return "ReservationRequest"

	case NetworkContainerRequest:
		return "NetworkContainerRequest"

	case VirtualNetworkRequest:
		return "VirtualNetworkRequest"

	case ReservationSetRequest:
		return "ReservationSetRequest"

	case SubnetCleanupRequest:
		return "SubnetCleanupRequest"

	case NodeSetCleanupRequest:
		return "NodeSetCleanupRequest"

	case InvalidRequest:
		return "InvalidRequest"
	}

	return "Unknown Status"
}
