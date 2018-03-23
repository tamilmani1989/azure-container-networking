package datamodel

// ReservationState - Request state.
type ReservationState int

const (
	// Process - Reservation has not been processed.
	Process ReservationState = iota

	// Allocate - Reservation pending allocation.
	Allocate

	// Release - Reservation pending release.
	Release
)

func (reservationState ReservationState) String() string {
	switch reservationState {

	case Process:
		return "Process"

	case Allocate:
		return "Allocate"

	case Release:
		return "Release"
	}

	return "Unknown Status"
}
