package api

import (
	"fmt"
	"net/http"
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

func NotFound(code ErrorCode, msg string) *JsonErr {
	return &JsonErr{Code: code, Text: msg, HTTPStatus: http.StatusNotFound}
}

func NotFoundf(code ErrorCode, format string, args ...interface{}) *JsonErr {
	return &JsonErr{Code: code, Text: fmt.Sprintf(format, args...), HTTPStatus: http.StatusNotFound}
}

func BadRequest(code ErrorCode, msg string) *JsonErr {
	return &JsonErr{Code: code, Text: msg, HTTPStatus: http.StatusBadRequest}
}

func BadRequestf(code ErrorCode, format string, args ...interface{}) *JsonErr {
	return &JsonErr{Code: code, Text: fmt.Sprintf(format, args...), HTTPStatus: http.StatusBadRequest}
}

func InternalServerError(code ErrorCode, msg string) *JsonErr {
	return &JsonErr{Code: code, Text: msg, HTTPStatus: http.StatusInternalServerError}
}

func InternalServerErrorf(code ErrorCode, format string, args ...interface{}) *JsonErr {
	return &JsonErr{Code: code, Text: fmt.Sprintf(format, args...), HTTPStatus: http.StatusInternalServerError}
}

func Conflict(code ErrorCode, msg string) *JsonErr {
	return &JsonErr{Code: code, Text: msg, HTTPStatus: http.StatusConflict}
}

func UnprocessableEntity(code ErrorCode, msg string) *JsonErr {
	return &JsonErr{Code: code, Text: msg, HTTPStatus: http.StatusUnprocessableEntity}
}

func PreconditionFailed(code ErrorCode, msg string) *JsonErr {
	return &JsonErr{Code: code, Text: msg, HTTPStatus: http.StatusPreconditionFailed}
}
