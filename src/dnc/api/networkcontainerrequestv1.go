package api

import (
	"fmt"
	"net/http"

	"github.com/Azure/azure-container-networking/cns"
)

// NetworkContainerRequestV1 - struct to hold network container request.
type NetworkContainerRequestV1 struct {
	ReservationID              string                                `json:"ReservationID"`
	ReservationSetID           string                                `json:"ReservationSetID"`
	AssociationInfo            AssociationInfo                       `json:"AssociationInfo"`
	AllowHostToNCCommunication bool                                  `json:"AllowHostToNCCommunication"`
	AllowNCToHostCommunication bool                                  `json:"AllowNCToHostCommunication"`
	EndpointPolicies           []cns.NetworkContainerRequestPolicies `json:"EndpointPolicies,omitempty"`
}

func (networkContainerRequestV1 *NetworkContainerRequestV1) String() string {
	return fmt.Sprintf(
		"NetworkContainerRequestV1:[ReservationID:%s, ReservationSetID:%s, AssociationInfo:%s, AllowHostToNCCommunication:%t AllowNCToHostCommunication:%t, EndpointPolicies:%+v]",
		networkContainerRequestV1.ReservationID, networkContainerRequestV1.ReservationSetID, networkContainerRequestV1.AssociationInfo.String(), networkContainerRequestV1.AllowHostToNCCommunication,
		networkContainerRequestV1.AllowNCToHostCommunication,
		networkContainerRequestV1.EndpointPolicies)
}

// Validate - Validates the NetworkContainerRequestV1
func (networkContainerRequestV1 *NetworkContainerRequestV1) Validate() *JsonErr {
	if networkContainerRequestV1.ReservationID == "" && networkContainerRequestV1.ReservationSetID == "" {
			return &JsonErr{Code: InvalidReservationID, Text: "Either Reservation ID or Reservation Set ID is required"}
	}

	if networkContainerRequestV1.ReservationID != "" && networkContainerRequestV1.ReservationSetID != "" {
		return &JsonErr{Code: InvalidReservationID, Text: "Cannot specify both Reservation ID and Reservation Set ID. Only one is required."}
	}

	if err := networkContainerRequestV1.AssociationInfo.Validate(); err != nil {
		return err
	}

	if networkContainerRequestV1.EndpointPolicies != nil {
		for _, requestedPolicy := range networkContainerRequestV1.EndpointPolicies {
			if err := requestedPolicy.Validate(); err != nil {
				return &JsonErr{Code: InvalidEndpointPolicy,
					Text:       fmt.Sprintf("Invalid endpoint policy %s ", err),
					HTTPStatus: http.StatusBadRequest}
			}
		}
	}

	return nil
}
