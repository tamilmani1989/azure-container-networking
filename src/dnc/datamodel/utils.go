package datamodel

import (
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/storage"
)

// SubnetKeyPlaceholder is an empty string which is used as subnet key
// This is used as a value in subnet config map. It avoids adding redundant data
// in the table where the max char count per entry is capped at 32K
const SubnetKeyPlaceholder = ""

//GreSetting type for UpdateGreCapable in NodeCfg
type GreSetting int

const (
	// GreCapableNotSet is checked for Node
	GreCapableNotSet GreSetting = iota

	//NmAgentIncapable i.e. does not support GRE alloc by DNC
	NmAgentIncapable

	//ExistingNCsWithoutGRE i.e. Node has NCs without GRE allocs
	ExistingNCsWithoutGRE

	//GreCapable Node can get GRE alloc
	GreCapable
)

func (gs GreSetting) String() string {
	switch gs {
	case GreCapableNotSet:
		return "GreCapableNotSet"

	case NmAgentIncapable:
		return "NmAgentIncapable"

	case ExistingNCsWithoutGRE:
		return "ExistingNCsWithoutGRE"

	case GreCapable:
		return "GreCapable"

	default:
		return "UnknownSetting"
	}
}

// GetSubnetKey - Helper function to generate subnet configuration key.
func GetSubnetKey(networkID string, ipSubnet string) string {
	subnetCfgKey := fmt.Sprintf(subnetConfigurationKey, networkID, ipSubnet)
	return strings.Replace(subnetCfgKey, "/", "-", -1)
}

// GetReservationKey - Helper function to generate reservation key.
func GetReservationKey(networkID, reservationID string) string {
	return fmt.Sprintf(ipReservationKey, networkID, reservationID)
}

// GetReservationIDFromKey - Helper function to get reservation ID from key.
func GetReservationIDFromKey(key string) string {
	// ipReservationKey = Network_%s_Reservation_%s
	// Splitting on _ gives [1] as NetworkID and [3] as ReservationID
	return strings.Split(key, "_")[3]
}

// GetNodeKey - Helper function to generate node key.
func GetNodeKey(nodeID string) string {
	return fmt.Sprintf(nodeKey, nodeID)
}

// GetNetworkKey - Helper function to generate network key.
func GetNetworkKey(networkID string) string {
	return fmt.Sprintf(networkKey, networkID)
}

// GetNetworkIdFromKey - Helper function to get network id from network key.
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

// GetNCIdFromKey - Helper function to get network container id from key.
func GetNCIdFromKey(key string) string {
	// ncKey = Network_%s_ID_%s
	// Splitting on _ gives [1] as NetworkID and [3] as ncID
	return strings.Split(key, "_")[3]
}

// GetNcPendingPublishContextKey - Helper function to generate key.
func GetNcPendingPublishContextKey(networkID, ncID string, pubsubAzId string) string {
	return fmt.Sprintf(ncPendingPublishContextKey, networkID, ncID, pubsubAzId)
}

//CheckGreConditions checks for conditions required to assign a GreKey
func CheckGreConditions(
	nodeID string,
	ncID string,
	nodeGreCapability GreSetting,
	ncGreKey uint16,
	vlanID uint16) error {

	if ncGreKey != 0 {
		return fmt.Errorf("NC has GreKey assigned NodeID:%s NC:%s GreKey:%v",
			nodeID, ncID, ncGreKey)
	}

	if vlanID == 0 {
		return fmt.Errorf("NC's network has no VLANID assigned NodeID:%s NC:%s",
			nodeID, ncID)
	}

	return nil
}

func ExtractStringProperty(e *storage.Entity, propName string) string {
	prop, ok := e.Properties[propName]
	if !ok {
		return ""
	}

	// checking the two returns from a type conversion avoids the panic.
	// the first return is zero value of the target type even if the second return is false,
	// so it's safe to just return it directly.
	v, _ := prop.(string)
	return v
}

func ExtractFloat64Property(e *storage.Entity, propName string) (float64, error) {
	prop, ok := e.Properties[propName]
	if !ok {
		return 0, fmt.Errorf("could not find %s in storage entity: %+v", propName, *e)
	}

	v, ok := prop.(float64)
	if !ok {
		return 0, fmt.Errorf("incorrect property type for %s. Expected float64", propName)
	}

	return v, nil
}

func ExtractBoolProperty(e *storage.Entity, propName string) (bool, error) {
	prop, ok := e.Properties[propName]
	if !ok {
		return false, fmt.Errorf("could not find %s in storage entity: %+v", propName, *e)
	}

	v, ok := prop.(bool)
	if !ok {
		return false, fmt.Errorf("incorrect property type for %s. Expected bool", propName)
	}

	return v, nil
}
