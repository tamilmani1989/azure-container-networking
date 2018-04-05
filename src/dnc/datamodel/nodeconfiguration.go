package datamodel

import (
	"dnc/api"
	"dnc/utils"
	"fmt"
)

// NodeConfiguration - struct to hold the node configuration.
type NodeConfiguration struct {
	NodeID                      string
	APIVersion                  string
	NodeInfo                    api.NodeInfoRequest
	DeployedNetworkContainerMap map[string]map[string]string
	ConstrainedReservations     map[string]string
	NetworkVlanIDMap            map[string]uint16
}

// DumpDeployedNetworkContainers - Dump deployed containers
func (n *NodeConfiguration) DumpDeployedNetworkContainers() string {
	deployedNCs := "NetworkContainers:["
	if n.DeployedNetworkContainerMap != nil {
		ncByVnets := make(map[string]string)
		for networkID, ncMap := range n.DeployedNetworkContainerMap {
			ncByVnets[networkID] = utils.PrintMapKeyValues(ncMap)
		}

		deployedNCs = utils.PrintMapKeyValues(ncByVnets)
	}
	deployedNCs = "]"

	return deployedNCs
}

func (n *NodeConfiguration) String() string {
	reservationKeys := ""
	if n.ConstrainedReservations != nil {
		reservationKeys = utils.PrintMapKeyValues(n.ConstrainedReservations)
	}

	deployedNCs := ""
	if n.DeployedNetworkContainerMap != nil {
		ncByVnets := make(map[string]string)
		for networkID, ncMap := range n.DeployedNetworkContainerMap {
			ncByVnets[networkID] = utils.PrintMapKeyValues(ncMap)
		}

		deployedNCs = utils.PrintMapKeyValues(ncByVnets)
	}
	return fmt.Sprintf(
		"[NodeID:%s, NodeInfo:%s NetworkContainers:%s ConstrainedReservations:%s",
		n.NodeID, n.NodeInfo.String(), deployedNCs, reservationKeys)
}
