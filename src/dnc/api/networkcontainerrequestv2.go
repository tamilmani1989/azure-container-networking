package api

import (
	"fmt"
)

// NetworkContainerRequestV2 - Struct to hold network container request.
type NetworkContainerRequestV2 struct {
	AllocationRequest          AllocationRequest `json:"AllocationRequest"`
	AssociationInfo            AssociationInfo   `json:"AssociationInfo"`
	AllowHostToNCCommunication bool              `json:"AllowHostToNCCommunication"`
}

func (networkContainerRequestV2 *NetworkContainerRequestV2) String() string {
	return fmt.Sprintf(
		"NetworkContainerRequestV1:[AllocationRequest:%s, AssociationInfo:%s, AllowHostToNCCommunication:%t]",
		networkContainerRequestV2.AllocationRequest.String(), networkContainerRequestV2.AssociationInfo.String(), networkContainerRequestV2.AllowHostToNCCommunication)
}

// Validate - Validates network container request V2.
func (networkContainerRequestV2 *NetworkContainerRequestV2) Validate() *JsonErr {
	err := networkContainerRequestV2.AllocationRequest.Validate()
	if err != nil {
		return err
	}

	return networkContainerRequestV2.AssociationInfo.Validate()
}
