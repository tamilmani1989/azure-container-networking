package api

import (
	"fmt"
)

// JsonErr - Struct to hold err info
type JsonErr struct {
	Code       ErrorCode `json:"ErrorCode"`
	Text       string    `json:"Text"`
	HTTPStatus int       `json:"HttpStatusCode"`
}

func (j *JsonErr) String() string {
	return fmt.Sprintf(
		"JsonError:[Code:%s, Text:%s, HTTPStatus:%d]", j.Code.String(), j.Text, j.HTTPStatus)
}

// Error makes JsonErr implement the error interface
func (j *JsonErr) Error() string {
	return j.String()
}
