package api

import (
	"fmt"
)

// ReservationResponse - Struct to hold reservation response.
type ReservationResponse struct {
	NetworkID         string            `json:"networkid"`
	ReservationID     string            `json:"reservationid"`
	AllocationRequest AllocationRequest `json:"AllocationRequest"`
	AllocatedIPSubnet string            `json:"AllocatedIPSubnet"`
	RequestStatus     RequestStatus     `json:"RequestStatus"`
}

func (r *ReservationResponse) String() string {
	return fmt.Sprintf(
		"ReservationResponse:[NetworkID:%s, ReservationID:%s, AllocationRequest:%s, AllocatedIPSubnet:%s, RequestStatus:%s]",
		r.NetworkID, r.ReservationID, r.AllocationRequest.String(), r.AllocatedIPSubnet, r.RequestStatus.String())
}
