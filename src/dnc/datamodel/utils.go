package datamodel

import (
	"fmt"
	"strings"
)

// GetSubnetKey - Helper function to generate subnet configuration key.
func GetSubnetKey(networkID string, ipSubnet string) string {
	subnetCfgKey := fmt.Sprintf(subnetConfigurationKey, networkID, ipSubnet)
	return strings.Replace(subnetCfgKey, "/", "-", -1)
}

// GetReservationKey - Helper function to generate reservation key.
func GetReservationKey(networkID, reservationID string) string {
	return fmt.Sprintf(ipReservationKey, networkID, reservationID)
}

// GetNodeKey - Helper function to generate node key.
func GetNodeKey(nodeID string) string {
	return fmt.Sprintf(nodeKey, nodeID)
}

// GetNetworkKey - Helper function to generate network key.
func GetNetworkKey(networkID string) string {
	return fmt.Sprintf(networkKey, networkID)
}

// GetNetworkKey - Helper function to get network id from network key.
func GetNetworkIdFromKey(key string) string {
	emptyKey := fmt.Sprintf(networkKey, "")
	return strings.Split(key, emptyKey)[1]
}

// GetNCKey - Helper function to generate network container key.
func GetNCKey(networkID, ncID string) string {
	return fmt.Sprintf(ncKey, networkID, ncID)
}

// GetTagDefintionKey - Helper function to generate tag defintion key.
func GetTagDefintionKey(tagType, tagName string) string {
	return fmt.Sprintf(tagDefinitionKey, tagType, tagName)
}

// GetNodeIDFromKey - Helepr to retrieve node ID from node key.
func GetNodeIDFromKey(key string) string {
	emptyKey := fmt.Sprintf(nodeKey, "")
	return strings.Split(key, emptyKey)[1]
}
