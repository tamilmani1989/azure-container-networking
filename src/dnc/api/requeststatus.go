package api

import (
	"fmt"
)

// RequestStatus - Struct to hold request status.
type RequestStatus struct {
	State         string  `json:"State"`
	Status        string  `json:"Status"`
	FailureDetail JsonErr `json:"FailureDetail"`
}

func (r *RequestStatus) String() string {
	return fmt.Sprintf(
		"RequestStatus:[State:%s, Status:%s, FailureDetail:%s]",
		r.State, r.Status, r.FailureDetail.String())
}
