package api

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/Azure/azure-container-networking/cns"
)

// NodeInfoRequest - Node ifnormation reuqest
type NodeInfoRequest struct {
	IPAddresses           []string `json:"ipaddresses"`
	LocationType          string   `json:"locationtype"` // Optional.
	OrchestratorType      string   `json:"orchestratortype"`
	InfrastructureNetwork string   `json:"infrastructurenetwork"`
}

func (n *NodeInfoRequest) String() string {
	ipAddresses := strings.Join(n.IPAddresses, ",")
	return fmt.Sprintf(
		"NodeInfoRequest:[IPAddresses:%s Locationtype:%s OrchestratorType:%s InfrastructureNetwork:%s]",
		ipAddresses, n.LocationType, n.OrchestratorType, n.InfrastructureNetwork)
}

// Validate - Validates the node info request
func (n *NodeInfoRequest) Validate() *JsonErr {
	if n.IPAddresses == nil || len(n.IPAddresses) < 1 {
		return &JsonErr{
			Code:       InvalidNodeIPAddress,
			Text:       "Node IP Addresses is invalid.",
			HTTPStatus: http.StatusBadRequest}
	}

	for _, ipAddress := range n.IPAddresses {
		ip := net.ParseIP(ipAddress)
		if ip == nil {
			return &JsonErr{
				Code:       InvalidNodeIPAddress,
				Text:       "Node IP Addresses is invalid. Parse failed",
				HTTPStatus: http.StatusBadRequest}
		}
	}

	switch n.OrchestratorType {
	case cns.WebApps:
	case cns.Kubernetes:
	case cns.Batch:
	case cns.ServiceFabric:
	default:
		return &JsonErr{
			Code:       InvalidOrchestratorType,
			Text:       "Node Orchestrator Type is invalid.",
			HTTPStatus: http.StatusBadRequest}
	}

	return nil
}

// NodeInfoRequests - Array of node info request
type NodeInfoRequests []NodeInfoRequest

func (nodeInfos NodeInfoRequests) String() string {
	nodeInfosString := "NodeInfoRequests:["
	for _, nodeInfo := range nodeInfos {
		nodeInfosString += nodeInfo.String()
	}
	nodeInfosString += "]"

	return nodeInfosString
}
