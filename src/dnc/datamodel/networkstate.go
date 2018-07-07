package datamodel

// NetworkState - Network state.
type NetworkState int

const (
	// Join - Join state.
	Join NetworkState = iota + 1

	// UnJoin - Network in unjoin state.
	UnJoin
)

func (networkState NetworkState) String() string {
	switch networkState {

	case Join:
		return "Join"

	case UnJoin:
		return "UnJoin"
	}

	return "Unknown Status"
}
