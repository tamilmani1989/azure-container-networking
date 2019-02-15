package api

import "fmt"

// IsWaitingForNodeCleanupResponse - Response is set to true to signify that the target NC should skip its disassociation state
type IsWaitingForNodeCleanupResponse struct {
	IsWaitingForNodeCleanup bool `json:"IsWaitingForNodeCleanup"`
}

func (r *IsWaitingForNodeCleanupResponse) String() string {
	return fmt.Sprintf("IsWaitingForNodeCleanupResponse:[IsWaitingForNodeCleanup:%t]", r.IsWaitingForNodeCleanup)
}
