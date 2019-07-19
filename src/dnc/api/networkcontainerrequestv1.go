package api

import (
	"fmt"
)

// NetworkContainerRequestV1 - struct to hold network container request.
type NetworkContainerRequestV1 struct {
	ReservationID              string          `json:"ReservationID"`
	AssociationInfo            AssociationInfo `json:"AssociationInfo"`
	AllowHostToNCCommunication bool            `json:"AllowHostToNCCommunication"`
	AllowNCToHostCommunication bool            `json:"AllowNCToHostCommunication"`
}

func (networkContainerRequestV1 *NetworkContainerRequestV1) String() string {
	return fmt.Sprintf(
		"NetworkContainerRequestV1:[ReservationID:%s, AssociationInfo:%s, AllowHostToNCCommunication:%t AllowNCToHostCommunication:%t]",
		networkContainerRequestV1.ReservationID, networkContainerRequestV1.AssociationInfo.String(), networkContainerRequestV1.AllowHostToNCCommunication,
		networkContainerRequestV1.AllowNCToHostCommunication)
}

// Validate - Validates the NetworkContainerRequestV1
func (networkContainerRequestV1 *NetworkContainerRequestV1) Validate() *JsonErr {
	if len(networkContainerRequestV1.ReservationID) == 0 {
		return &JsonErr{Code: InvalidReservationID, Text: "Reservation ID cannot be empty"}
	}

	return networkContainerRequestV1.AssociationInfo.Validate()
}
