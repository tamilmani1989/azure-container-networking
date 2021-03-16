package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/azure-container-networking/cns"
)

// NetworkContainerRequestV2 - Struct to hold network container request.
type NetworkContainerRequestV2 struct {
	AllocationRequest          AllocationRequest                     `json:"AllocationRequest"`
	AssociationInfo            AssociationInfo                       `json:"AssociationInfo"`
	AllowHostToNCCommunication bool                                  `json:"AllowHostToNCCommunication"`
	AllowNCToHostCommunication bool                                  `json:"AllowNCToHostCommunication"`
	EndpointPolicies           []cns.NetworkContainerRequestPolicies `json:"EndpointPolicies,omitempty"`
	ToBeDeletedIPIds           []string                              `json:"ToBeDeletedIPIds"`
	BmContainerPhysicalInstID  string                                `json:"InstanceID"` // physical instance id of baremetal container
}

func (networkContainerRequestV2 *NetworkContainerRequestV2) String() string {
	return fmt.Sprintf(
		"NetworkContainerRequestV2:[AllocationRequest:%s, AssociationInfo:%s, AllowHostToNCCommunication:%t, AllowNCToHostCommunication:%t,"+
			"EndpointPolicies:%+v, ToBeDeletedIPIds:%s ]",
		networkContainerRequestV2.AllocationRequest.String(),
		networkContainerRequestV2.AssociationInfo.String(), networkContainerRequestV2.AllowHostToNCCommunication,
		networkContainerRequestV2.AllowNCToHostCommunication,
		networkContainerRequestV2.EndpointPolicies,
		strings.Join(networkContainerRequestV2.ToBeDeletedIPIds, ","))
}

// Validate - Validates network container request V2.
func (networkContainerRequestV2 *NetworkContainerRequestV2) Validate() *JsonErr {
	err := networkContainerRequestV2.AllocationRequest.Validate()
	if err != nil {
		return err
	}
	if err = networkContainerRequestV2.AssociationInfo.Validate(); err != nil {
		return err
	}

	if networkContainerRequestV2.EndpointPolicies != nil {
		for _, requestedPolicy := range networkContainerRequestV2.EndpointPolicies {
			if err := requestedPolicy.Validate(); err != nil {
				return &JsonErr{
					Code:       InvalidEndpointPolicy,
					Text:       fmt.Sprintf("Invalid endpoint policy %s", err),
					HTTPStatus: http.StatusBadRequest}
			}
		}
	}

	return nil
}
