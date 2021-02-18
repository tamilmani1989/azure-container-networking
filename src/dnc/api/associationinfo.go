package api

import (
	"dnc/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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
	case cns.COW:
	case BareMetalContainerType:
	default:
		return &JsonErr{Code: InvalidContainerType, Text: "Invalid container type.", HTTPStatus: http.StatusBadRequest}
	}

	var podInfo cns.KubernetesPodInfo
	err := json.Unmarshal(associationInfo.OrchestratorContext, &podInfo)

	if err != nil {
		return BadRequestf(InvalidOrchestratorContext,"Invalid container context")
	}

	if podInfo.PodName == "" || podInfo.PodNamespace == "" {
		return BadRequestf(InvalidOrchestratorContext, "PodName/PodNameSpace should be non-empty")
	}

	// For baremetal container pod name should be of format  <podname>_<logical container id>.
	// example - testcontainer_18b534aa-7190-4185-9988-af7feace7e47
	// It is caller's responsibility to pass in logical container id like this
	if IsBareMetalContainer(associationInfo.ContainerType) {
		var splits []string
		splits = strings.Split(podInfo.PodName, ContainerIdDelimiter)
		podNamelength := len(splits)
		if podNamelength != 2 || !utils.IsValidUUID(splits[podNamelength-1]) {
			return BadRequestf(
				InvalidOrchestratorContext,
				"Invalid PodName format. Should be in format <podname>_<logical container id>")
		}
	}

	return nil
}