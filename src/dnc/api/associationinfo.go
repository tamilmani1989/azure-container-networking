package api

import (
	"fmt"
)

// AssociationInfo - Struct to hold network container association info.
type AssociationInfo struct {
	NodeID      string `json:"NodeID"`
	InterfaceID string `json:"InterfaceID"` // IPAddress of the interface.
}

func (associationInfo *AssociationInfo) String() string {
	return fmt.Sprintf(
		"AssociationInfo:[NodeID:%s, InterfaceID:%s]",
		associationInfo.NodeID, associationInfo.InterfaceID)
}

// Validate - Validates the AssociationInfo
func (associationInfo *AssociationInfo) Validate() *JsonErr {
	if len(associationInfo.NodeID) == 0 {
		return &JsonErr{Code: InvalidNodeID, Text: "Node ID cannot be empty"}
	}

	return nil
}
