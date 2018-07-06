package api

import (
	"fmt"
)

// NetworkContainerResponse - Struct to hold network container response.
type NetworkContainerResponse struct {
	NcID            string          `json:"NetworkContainerID"`
	IPConfiguration IPConfiguration `json:"IPConfiguration"`
	AssociationInfo AssociationInfo `json:"AssociationInfo"`
	RequestStatus   RequestStatus   `json:"RequestStatus"`
	VlanID          uint16          `json:"VlanID"`
}

func (r *NetworkContainerResponse) String() string {
	return fmt.Sprintf(
		"NetworkContainerResponse:[NetworkContainerID:%s, IPConfiguration:%s, AssociationInfo:%s, RequestStatus:%s, VlanID:%d]",
		r.NcID, r.IPConfiguration.String(), r.AssociationInfo.String(), r.RequestStatus.String(), r.VlanID)
}
