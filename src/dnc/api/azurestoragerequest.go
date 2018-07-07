package api

import (
	"fmt"
	"net/http"
)

// AzureStorageRequest - Details to connect to azure storage account.
type AzureStorageRequest struct {
	AccountName  string `json:"AccountName"`
	Key          string `json:"Key"`
	PartitionKey string `json:"PartitionKey"`
}

func (azurestoragerequest *AzureStorageRequest) String() string {
	return fmt.Sprintf("AzureStorageRequest:[AccountName:%s, PartitionKey:%s]", azurestoragerequest.AccountName, azurestoragerequest.PartitionKey)
}

// Validate - Validates the AzureStorageRequest
func (azurestoragerequest *AzureStorageRequest) Validate() *JsonErr {
	if len(azurestoragerequest.AccountName) == 0 {
		return &JsonErr{Code: InvalidAzureStorageInfo, Text: "Azure storage account name cannot be empty.", HTTPStatus: http.StatusBadRequest}
	}

	if len(azurestoragerequest.Key) == 0 {
		return &JsonErr{Code: InvalidAzureStorageInfo, Text: "Azure storage key cannot be empty.", HTTPStatus: http.StatusBadRequest}
	}

	if len(azurestoragerequest.PartitionKey) == 0 {
		return &JsonErr{Code: InvalidAzureStorageInfo, Text: "Azure storage PartitionKey cannot be empty.", HTTPStatus: http.StatusBadRequest}
	}

	return nil
}
