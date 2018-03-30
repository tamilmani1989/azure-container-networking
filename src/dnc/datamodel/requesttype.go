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
)

func (requestType RequestType) String() string {
	switch requestType {

	case ReservationRequest:
		return "ReservationRequest"

	case NetworkContainerRequest:
		return "NetworkContainerRequest"

	case InvalidRequest:
		return "InvalidRequest"
	}

	return "Unknown Status"
}
