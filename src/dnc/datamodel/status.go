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
	}

	return "Unknown Status"
}
