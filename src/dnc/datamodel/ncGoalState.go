package datamodel

import (
	"dnc/nmagentcontracts"
	"encoding/json"
)

// NCGoalState - Struct to cache the Network Container Goal State
type NCGoalState struct {
	NetworkContainerID    string
	Version               uint64
	NetworkID             string
	InfraNetworkID        string
	AllocatedIPSubnet     string
	SecondaryIPs          []string
	BaseSubnet            string
	BaseSubnetName        string
	AccessToken           string
	GatewayIP             string
	DNSServers            []string
	AssociatedNodeID      string
	AssociatedInterfaceID string
	CNetSubnets           []string
	VNetSubnets           []string
	NodeInterfaceID       string
	TagsInfo              []nmagentcontracts.Tag
	ContainerType         string
	OrchestratorContext   json.RawMessage
	VlanID                int
	VnetVersion           uint64
	LocalIP               string
}
