package datamodel

import (
	"dnc/api"
	"dnc/utils"
)

// NodeConfiguration - struct to hold the node configuration.
type NodeConfiguration struct {
	NodeID                      string
	APIVersion                  string
	NodeInfo                    api.NodeInfoRequest
	DeployedNetworkContainerMap map[string]map[string]string //Map of Network Ids with NCId and ReservationID
	ConstrainedReservations     map[string]string
	NetworkVlanIDMap            map[string]uint16
	Registered                  bool
	NcLimit                     int
	UpdateGreCapable            GreSetting
	InterfaceGreKeyMap          map[string]map[string]uint16 // Map of InterfaceIds with ReservationIDs ( or AllocationIDs  for future use) and GreKeys
}

// DumpDeployedNetworkContainers - Dump deployed containers
func (n *NodeConfiguration) DumpDeployedNetworkContainers() string {
	deployedNCs := "NetworkContainers:["
	if n.DeployedNetworkContainerMap != nil {
		ncByVnets := make(map[string]string)
		for networkID, ncMap := range n.DeployedNetworkContainerMap {
			ncByVnets[networkID] = utils.PrintMapKeyValues(ncMap)
		}

		deployedNCs = deployedNCs + utils.PrintMapKeyValues(ncByVnets)

	}

	deployedNCs = deployedNCs + "]"

	return deployedNCs
}
