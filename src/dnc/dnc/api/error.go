package api

import (
	"fmt"
)

// JsonErr - Struct to hold err info
type JsonErr struct {
	Code       ErrorCode `json:"ErrorCode"`
	Text       string    `json:"Text"`
	HTTPStatus int       `json:"-"`
}

func (j *JsonErr) String() string {
	return fmt.Sprintf(
		"JsonError:[Code:%s, Text:%s, HTTPStatus:%d]", j.Code.String(), j.Text, j.HTTPStatus)
}
