package nmagentcontracts

import (
	"fmt"
	"strings"
)

// TagResponse - Defines the tag response struct
type TagResponse struct {
	ResponseCode string   `json:"httpStatusCode"`
	Name         string   `json:"tagName"`
	Values       []string `json:"addresses"`
}

func (tagResponse *TagResponse) String() string {
	return fmt.Sprintf("ResponseCode:%s, Name:%s, Values:%s", tagResponse.ResponseCode, tagResponse.Name, strings.Join(tagResponse.Values, ","))
}
