package nmagentcontracts

import "fmt"

// TagResponse - Defines the tag response struct
type TagResponse struct {
	ResponseCode string `json:"httpStatusCode"`
	Name         string `json:"tagName"`
	Value        string `json:"addresses"`
}

func (tagResponse *TagResponse) String() string {
	return fmt.Sprintf("ResponseCode:%s, Name:%s, Value:%s", tagResponse.ResponseCode, tagResponse.Name, tagResponse.Value)
}
