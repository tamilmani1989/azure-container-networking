package api

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
