package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Azure/azure-container-networking/cns"
)

// AssociationInfo - Struct to hold network container association info.
type AssociationInfo struct {
	NodeID              string          `json:"NodeID"`
	InterfaceID         string          `json:"InterfaceID"` // IPAddress of the interface.
	ContainerType       string          `json:"ContainerType"`
	OrchestratorContext json.RawMessage `json:"OrchestratorContext"`
}

func (associationInfo *AssociationInfo) String() string {
	return fmt.Sprintf(
		"AssociationInfo:[NodeID:%s, InterfaceID:%s, ContainerType:%s, OrchestratorContext:%s]",
		associationInfo.NodeID, associationInfo.InterfaceID, associationInfo.ContainerType, associationInfo.OrchestratorContext)
}

// Validate - Validates the AssociationInfo
func (associationInfo *AssociationInfo) Validate() *JsonErr {
	if len(associationInfo.NodeID) == 0 {
		return &JsonErr{Code: InvalidNodeID, Text: "Node ID cannot be empty", HTTPStatus: http.StatusBadRequest}
	}

	switch associationInfo.ContainerType {
	case cns.AzureContainerInstance:
	case cns.WebApps:
	case cns.Docker:
	case cns.JobObject:
	default:
		return &JsonErr{Code: InvalidContainerType, Text: "Invalid container type.", HTTPStatus: http.StatusBadRequest}
	}

	return nil
}
