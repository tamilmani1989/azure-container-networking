package api

import (
	"fmt"
)

// NetworkContainerResponse - Struct to hold network container response.
type NetworkContainerResponse struct {
	NcID                       string          `json:"NetworkContainerID"`
	IPConfiguration            IPConfiguration `json:"IPConfiguration"`
	AssociationInfo            AssociationInfo `json:"AssociationInfo"`
	RequestStatus              RequestStatus   `json:"RequestStatus"`
	VlanID                     uint16          `json:"VlanID"`
	LocalIP                    string          `json:"LocalIP"`
	MacAddress                 string          `json:"MacAddress"`
	NCVersion                  int             `json:"NCVersion"`
	AllowHostToNCCommunication bool            `json:"AllowHostToNCCommunication"`
	AllowNCToHostCommunication bool            `json:"AllowNCToHostCommunication"`
}

func (r *NetworkContainerResponse) String() string {
	return fmt.Sprintf(
		`NetworkContainerResponse:[NetworkContainerID:%s, IPConfiguration:%s, AssociationInfo:%s,
		RequestStatus:%s, VlanID:%d, LocalIP:%s, MacAddress:%s, NCVersion:%d, AllowHostToNCCommunication:%t, AllowNCToHostCommunication:%t]`,
		r.NcID, r.IPConfiguration.String(), r.AssociationInfo.String(), r.RequestStatus.String(), r.VlanID, r.LocalIP, r.MacAddress, r.NCVersion, r.AllowHostToNCCommunication,
		r.AllowNCToHostCommunication)
}
