package api

import (
	"fmt"
)

// NodeInfoResponse - Struct to hold the node info response.
type NodeInfoResponse struct {
	NodeInfo                NodeInfoRequest
	NetworkContainers       []NetworkContainerResponse
	ConstrainedReservations []ReservationResponse
}

func (n *NodeInfoResponse) String() string {
	ncString := "NetworkContainers:["
	if n.NetworkContainers != nil {
		for _, networkContainer := range n.NetworkContainers {
			ncString += networkContainer.String()
		}
	}

	ncString += "]"
	constrainedReservationsString := "ConstrainedReservations:["
	if n.ConstrainedReservations != nil {
		for _, reservation := range n.ConstrainedReservations {
			constrainedReservationsString += reservation.String()
		}
	}

	constrainedReservationsString += "]"
	return fmt.Sprintf(
		"NodeInfoResponse:[NodeInfo:%s, NetworkContainers:%s, ConstrainedReservations:%s]",
		n.NodeInfo.String(), ncString, constrainedReservationsString)
}
