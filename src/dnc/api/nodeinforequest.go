package api

import (
	"dnc/configuration"
	"dnc/utils"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// NodeInfoRequest - Node ifnormation reuqest
type NodeInfoRequest struct {
	IPAddresses           []string `json:"ipaddresses"`
	LocationType          string   `json:"locationtype"` // Optional.
	OrchestratorType      string   `json:"orchestratortype"`
	InfrastructureNetwork string   `json:"infrastructurenetwork"`
	AZID                  string   `json:"azid"`     // Optional
	NodeType              string   `json:"nodetype"` // Optional.
}

func (n *NodeInfoRequest) String() string {
	ipAddresses := strings.Join(n.IPAddresses, ",")
	return fmt.Sprintf(
		"NodeInfoRequest:[IPAddresses:%s Locationtype:%s OrchestratorType:%s InfrastructureNetwork:%s AZID: %s NodeType: %s]",
		ipAddresses, n.LocationType, n.OrchestratorType, n.InfrastructureNetwork, n.AZID, n.NodeType)
}

// Validate - Validates the node info request
func (n *NodeInfoRequest) Validate(configuration *configuration.Configuration, nodeID string) *JsonErr {
	// Check for infrastructure network
	if configuration.DNCSettings.Managed && !utils.IsValidUUID(n.InfrastructureNetwork) {
		return &JsonErr{
			Code:       InvalidNetworkID,
			Text:       "Infrastructure Network ID must be a valid uuid",
			HTTPStatus: http.StatusBadRequest}
	}

	// validate node type
	if err := validateNodeType(n.NodeType); err != nil {
		return err
	}

	if IsBareMetalNode(n.NodeType) {
		// Check if provided node name is a guid.
		_, parseErr := uuid.Parse(nodeID)
		if parseErr != nil {
			return &JsonErr{
				Code:       InvalidNodeID,
				Text:       NodeIDNonGUIDValue,
				HTTPStatus: http.StatusBadRequest}
		}

		if n.AZID == "" {
			return &JsonErr{
				Code:       InvalidNodeAZID,
				Text:       "AZID field cannot be empty.",
				HTTPStatus: http.StatusBadRequest}
		}
	}

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

	// orchestrator type
	if err := validateOrchestratorType(n.OrchestratorType); err != nil {
		return err
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
