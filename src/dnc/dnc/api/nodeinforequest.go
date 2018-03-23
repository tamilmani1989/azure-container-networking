package api

import (
	"fmt"
	"net"
	"strings"
)

// NodeInfoRequest - Node ifnormation reuqest
type NodeInfoRequest struct {
	IPAddresses  []string `json:"ipaddresses"`
	LocationType string   `json:"locationtype"` // Optional.
}

func (n *NodeInfoRequest) String() string {
	ipAddresses := strings.Join(n.IPAddresses, ",")
	return fmt.Sprintf(
		"NodeInfoRequest:[IPAddresses:%s Locationtype:%s]",
		ipAddresses, n.LocationType)
}

// Validate - Validates the node info request
func (n *NodeInfoRequest) Validate() *JsonErr {
	if len(n.IPAddresses) == 0 {
		return &JsonErr{Code: InvalidNodeIPAddress, Text: "Node IP Addresses cannot be empty"}
	}
	for _, ipAddress := range n.IPAddresses {
		ip := net.ParseIP(ipAddress)
		if ip != nil {
			return &JsonErr{
				Code: InvalidNodeIPAddress,
				Text: "Node IP Addresses is invalid. Parse failed"}
		}
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
