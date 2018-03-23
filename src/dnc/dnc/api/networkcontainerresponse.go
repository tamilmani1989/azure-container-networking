package api

import (
	"fmt"
)

// NetworkContainerResponse - Strcut to hold network container response.
type NetworkContainerResponse struct {
	IPConfiguration IPConfiguration `json:"IPConfiguration"`
	AssociationInfo AssociationInfo `json:"AssociationInfo"`
	RequestStatus   RequestStatus   `json:"RequestStatus"`
}

func (r *NetworkContainerResponse) String() string {
	return fmt.Sprintf(
		"NetworkContainerResponse:[IPConfiguration:%s, AssociationInfo:%s, RequestStatus:%s]",
		r.IPConfiguration.String(), r.AssociationInfo.String(), r.RequestStatus.String())
}
