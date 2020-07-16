package datamodel

import (
	"dnc/api"
	"fmt"
)

// ReservationConfiguration - struct to hold reservation configuration.
type ReservationConfiguration struct {
	ReservationID   string
	NetworkID       string
	AllocatedSubnet string
	VniID           int
	ReservationState
	Status
	FailureDetail        api.JsonErr
	AllocationRequest    api.AllocationRequest
	CreatedDateTime      string
	LastUpdatedDateTime  string
	AssociatedNCID       string
	MarkedForDelete      bool
	APIVersion           string
	SecondaryAllocations map[string]string // allocation id to ip address
}

func (r *ReservationConfiguration) String() string {
	return fmt.Sprintf(
		"ReservationID:%s NetworkID:%s AllocatedSubnet:%s VniID:%d "+
			"ReservationState:%s Status:%s FailureDetail:%s "+
			"AllocationRequest:%s CreatedDateTime:%s LastUpdatedDateTime:%s "+
			"AssociatedNCID:%s, MarkedForDelete:%v",
		r.ReservationID, r.NetworkID, r.AllocatedSubnet, r.VniID,
		r.ReservationState.String(), r.Status.String(), r.FailureDetail.String(),
		r.AllocationRequest.String(), r.CreatedDateTime, r.LastUpdatedDateTime,
		r.AssociatedNCID, r.MarkedForDelete)

}
