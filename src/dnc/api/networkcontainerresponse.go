package api

import (
	"time"
)

// NetworkContainerResponse - Struct to hold network container response.
type NetworkContainerResponse struct {
	NcID                       string                 `json:"NetworkContainerID"`
	IPConfiguration            IPConfiguration        `json:"IPConfiguration"`
	AssociationInfo            AssociationInfo        `json:"AssociationInfo"`
	RequestStatus              RequestStatus          `json:"RequestStatus"`
	VlanID                     uint16                 `json:"VlanID"`
	LocalIP                    string                 `json:"LocalIP"`
	MacAddress                 string                 `json:"MacAddress"`
	NCVersion                  int                    `json:"NCVersion"`
	AllowHostToNCCommunication bool                   `json:"AllowHostToNCCommunication"`
	AllowNCToHostCommunication bool                   `json:"AllowNCToHostCommunication"`
	CreatedDateTime            time.Time              `json:"CreatedDateTime,omitempty"`
	StateTransitionLog         []StateTransitionStats `json:"StateTransitionLog,omitempty"`
}

type StateTransitionStats struct {
	State       string    `json:"State"`
	CompletedOn time.Time `json:"CompletedOn"`
}
