package api

import (
	"fmt"
	"net/http"

	"github.com/Azure/azure-container-networking/cns"
)

// Checks for all the allowed values of location types
func validateNodeType(nodeType string) *JsonErr {
	switch nodeType {
	case "":
		fallthrough
	case BareMetalNodeType:
		break
	default:
		return &JsonErr{
			Code:       InvalidNodeType,
			Text:       fmt.Sprintf("Invalid Node type %v", nodeType),
			HTTPStatus: http.StatusBadRequest}
	}

	return nil
}

// Checks for all the allowed values of orchestrator types
func validateOrchestratorType(orchestratorType string) *JsonErr {
	switch orchestratorType {
	case cns.WebApps:
	case cns.Kubernetes:
	case cns.KubernetesCRD:
	case cns.Batch:
	case cns.ServiceFabric:
	case cns.DBforPostgreSQL:
	case cns.AzureFirstParty:
	default:
		return &JsonErr{
			Code:       InvalidOrchestratorType,
			Text:       "Node Orchestrator Type is invalid.",
			HTTPStatus: http.StatusBadRequest}
	}

	return nil
}

// IsBareMetalNode - Helper function to check if node type is baremetal or not?
func IsBareMetalNode(nodeType string) bool {
	return nodeType == BareMetalNodeType
}

// IsBareMetalContainer - Helper function to check if container type is baremetal or not?
func IsBareMetalContainer(containerType string) bool {
	return containerType == BareMetalContainerType
}
