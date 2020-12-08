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

	interfaceGreMapString := ""
	if n.InterfaceGreKeyMap != nil {
		interfaceGreMap := make(map[string]string)
		for interfaceID, greMap := range n.InterfaceGreKeyMap {
			if greMap != nil {
				interfaceGreMap[interfaceID] = utils.PrintMapKeyValuesUint16(greMap)
			}
		}
		interfaceGreMapString = utils.PrintMapKeyValues(interfaceGreMap)
	}

	return fmt.Sprintf(
		"[NodeID:%s, NodeInfo:%s NetworkContainers:%s ConstrainedReservations:%s Registered:%t NcLimit:%d GreMap:%s",
		n.NodeID, n.NodeInfo.String(), deployedNCs, reservationKeys, n.Registered, n.NcLimit, interfaceGreMapString)
}
