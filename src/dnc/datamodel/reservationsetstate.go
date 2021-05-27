package datamodel

// ReservationSetState - Request state.
type ReservationSetState int

const (
	// Invalid - ReservationSet Invalid state.
	Unknown ReservationSetState = iota

	// Create - ReservationSet is in create state.
	Create

	// ScaleUp - ReservationSet is in ScaleUp state where set is adding new reservations to the set.
	ScaleUp

	// ScaleDown - ReservationSet is in ScaleDown state where set is removing existing reservations from the set.
	ScaleDown

	// Delete - ReservationSet is in deletion state.
	Delete
)

func (reservationSetState ReservationSetState) String() string {
	switch reservationSetState {
	case Create:
		return "Create"

	case ScaleUp:
		return "ScaleUp"

	case ScaleDown:
		return "ScaleDown"

	case Delete:
		return "Delete"
	}

	return "Unknown State"
}

func StringToReservationSetState(state string) ReservationSetState {
	switch state {
	case "Create":
		return Create

	case "ScaleUp":
		return ScaleUp

	case "ScaleDown":
		return ScaleDown

	case "Delete":
		return Delete
	}

	return Unknown
}
