package api

import "fmt"

// FailedToDisassociateNCResponse - Response is set to true to signify that the target NC should skip its disassociation state
type FailedToDisassociateNCResponse struct {
	FailedToDisassociateNCResponse bool `json:"SkipNodeCleanupForNetworkContainer"`
}

func (r *FailedToDisassociateNCResponse) String() string {
	return fmt.Sprintf("FailedToDisassociateNCResponse:[FailedToDisassociateNCResponse:%t]", r.FailedToDisassociateNCResponse)
}
