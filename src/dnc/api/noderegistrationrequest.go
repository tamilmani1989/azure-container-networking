package api

import (
	"fmt"
	"net/http"
)

// NodeRegistrationRequest - Struct to hold node registration request.
type NodeRegistrationRequest struct {
	NumCores int `json:"NumCores"`
}

func (nodeRegistrationRequest *NodeRegistrationRequest) String() string {
	return fmt.Sprintf(
		"NodeRegistrationRequest: [NumCores: %d]",
		nodeRegistrationRequest.NumCores)
}

// Validate - Validates node registration request.
func (nodeRegistrationRequest *NodeRegistrationRequest) Validate() *JsonErr {
	if nodeRegistrationRequest.NumCores <= 0 {
		return &JsonErr{
			Code:       InvalidNCLimit,
			Text:       fmt.Sprintf("Invalid NumCores: %d", nodeRegistrationRequest.NumCores),
			HTTPStatus: http.StatusBadRequest}
	}

	return nil
}
