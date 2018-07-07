package api

import (
	"crypto/sha256"
	"fmt"
	"net/http"
)

// SubnetRequest - Struct to hold subnet request.
type SubnetRequest struct {
	SubnetName          string `json:"SubnetName"` // Optional.
	AuthenticationToken string `json:"authenticationtoken"`
}

func (r *SubnetRequest) String() string {
	return fmt.Sprintf(
		"SubnetRequest:[SubnetName:%s, AuthenticationToken:%x]",
		r.SubnetName, sha256.Sum256([]byte(r.AuthenticationToken)))
}

// Validate - Validates the subnet request
func (r *SubnetRequest) Validate() *JsonErr {
	if len(r.AuthenticationToken) == 0 {
		return &JsonErr{Code: InvalidAuthenticationToken, Text: "AuthenticationToken cannot be empty", HTTPStatus: http.StatusBadRequest}
	}

	return nil
}

// SubnetRequests - Struct to hold subnet request.
type SubnetRequests []SubnetRequest

func (r SubnetRequests) String() string {
	subnetRequestsString := "SubnetRequests:"
	for _, subnetRequest := range r {
		subnetRequestsString += subnetRequest.String()
	}
	return subnetRequestsString
}
