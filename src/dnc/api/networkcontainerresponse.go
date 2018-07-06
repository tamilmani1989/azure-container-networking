package api

import (
	"fmt"
)

// NetworkContainerResponse - Strcut to hold network container response.
type NetworkContainerResponse struct {
	IPConfiguration IPConfiguration `json:"IPConfiguration"`
	AssociationInfo AssociationInfo `json:"AssociationInfo"`
	RequestStatus   RequestStatus   `json:"RequestStatus"`
	VlanID          uint16          `json:"VlanID"`
}

func (r *NetworkContainerResponse) String() string {
	return fmt.Sprintf(
		"NetworkContainerResponse:[IPConfiguration:%s, AssociationInfo:%s, RequestStatus:%s, VlanID:%d]",
		r.IPConfiguration.String(), r.AssociationInfo.String(), r.RequestStatus.String(), r.VlanID)
}
