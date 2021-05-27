package datamodel

// Status - Enum for status of the operation.
type Status int

const (
	// NotStarted - Prcessing not started.
	NotStarted Status = iota

	// Pending - Processing pending.
	Pending

	// Failed - Request processing failed.
	Failed

	// Denied - Request processing denied.
	Denied

	// Completed - Request proecssing completed.
	Completed

	// Refreshing - Request proecssing in refresh state.
	Refreshing

	// Undefined - Undefined / Invalid state.
	Undefined
)

func (status Status) String() string {
	switch status {
	case NotStarted:
		return "NotStarted"

	case Pending:
		return "Pending"

	case Failed:
		return "Failed"

	case Denied:
		return "Denied"

	case Completed:
		return "Completed"

	case Refreshing:
		return "Refreshing"
	}

	return "Undefined / Invalid"
}

func StringToStatus(status string) Status {
	switch status {
	case "NotStarted":
		return NotStarted

	case "Pending":
		return Pending

	case "Failed":
		return Failed

	case "Denied":
		return Denied

	case "Completed":
		return Completed

	case "Refreshing":
		return Refreshing
	}

	return Undefined
}
