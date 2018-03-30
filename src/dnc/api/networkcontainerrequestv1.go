package api

import (
	"fmt"
)

// NetworkContainerRequestV1 - struct to hold network container request.
type NetworkContainerRequestV1 struct {
	ReservationID   string          `json:"ReservationID"`
	AssociationInfo AssociationInfo `json:"AssociationInfo"`
}

func (networkContainerRequestV1 *NetworkContainerRequestV1) String() string {
	return fmt.Sprintf(
		"NetworkContainerRequestV1:[ReservationID:%s, AssociationInfo:%s]",
		networkContainerRequestV1.ReservationID, networkContainerRequestV1.AssociationInfo.String())
}

// Validate - Validates the NetworkContainerRequestV1
func (networkContainerRequestV1 *NetworkContainerRequestV1) Validate() *JsonErr {
	if len(networkContainerRequestV1.ReservationID) == 0 {
		return &JsonErr{Code: InvalidReservationID, Text: "Reservation ID cannot be empty"}
	}

	return networkContainerRequestV1.Validate()
}
