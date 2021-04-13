package api

import (
	"fmt"
	"strings"
)

// NodeInfoResponse - Struct to hold the node info response.
type NodeInfoResponse struct {
	NodeInfo                NodeInfoRequest
	NetworkContainers       []NetworkContainerResponse
	ConstrainedReservations []ReservationResponse
}

/* SlimNodeInfo is returned when a client calls the Get Nodes endpoint.
Since all nodes are returned, this struct is a subset of a Node's Configuration to ensure
the response does not end up too big */
type SlimNodeInfo struct {
	Name string
}

type NodesResponse struct {
	Nodes []SlimNodeInfo
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

func (s *SlimNodeInfo) String() string {
	return fmt.Sprintf("Node:[Name:[%s]]", s.Name)
}

func (n *NodesResponse) String() string {
	nodeStrings := make([]string, len(n.Nodes))
	for i, node := range n.Nodes {
		nodeStrings[i] = node.String()
	}

	nodeStringsComb := strings.Join(nodeStrings, ",")

	return fmt.Sprintf("Nodes:[%s]", nodeStringsComb)
}
