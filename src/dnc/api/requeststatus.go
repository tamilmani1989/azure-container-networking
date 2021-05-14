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

// RequestStatus - Struct to hold request NC status.
type NCRequestStatus struct {
	State         string  `json:"State"`
	Status        string  `json:"Status"`
	NICStatus     string  `json:"NICStatus"`
	FailureDetail JsonErr `json:"FailureDetail"`
}

func (r *NCRequestStatus) String() string {
	return fmt.Sprintf(
		"RequestStatus:[State:%s, Status:%s, NICStatus:%s, FailureDetail:%s]",
		r.State, r.Status, r.NICStatus, r.FailureDetail.String())
}
