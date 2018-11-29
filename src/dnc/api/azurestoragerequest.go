package api

import (
	"fmt"
	"net/http"
	"crypto/sha256"
)

// AzureStorageRequest - Details to connect to azure storage account.
type AzureStorageRequest struct {
	AccountName  string `json:"AccountName"`
	Key          string `json:"Key"`
	PartitionKey string `json:"PartitionKey"`
}

// AzureStorageRequestV2 - Details to connect to azure storage account along with SAS key for regional azure storage account.
type AzureStorageRequestV2 struct {
	AccountName                     string `json:"AccountName"`
	Key                             string `json:"Key"`
	PartitionKey                    string `json:"PartitionKey"`
	RegionalStorageConnectionString string `json:"RegionalStorageConnectionString"`
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

func (azureStorageRequestV2 *AzureStorageRequestV2) String() string {
	return fmt.Sprintf("AzureStorageRequestV2:[AccountName:%s, PartitionKey:%s, RegionalStorageConnectionString:%x]",
		azureStorageRequestV2.AccountName, azureStorageRequestV2.PartitionKey, sha256.Sum256([]byte(azureStorageRequestV2.RegionalStorageConnectionString)))
}

// Validate - Validates the AzureStorageRequestV2
func (azureStorageRequestV2 *AzureStorageRequestV2) Validate() *JsonErr {
	if len(azureStorageRequestV2.AccountName) == 0 {
		return &JsonErr{Code: InvalidAzureStorageInfo, Text: "Azure storage account name cannot be empty.", HTTPStatus: http.StatusBadRequest}
	}

	if len(azureStorageRequestV2.Key) == 0 {
		return &JsonErr{Code: InvalidAzureStorageInfo, Text: "Azure storage key cannot be empty.", HTTPStatus: http.StatusBadRequest}
	}

	if len(azureStorageRequestV2.PartitionKey) == 0 {
		return &JsonErr{Code: InvalidAzureStorageInfo, Text: "Azure storage PartitionKey cannot be empty.", HTTPStatus: http.StatusBadRequest}
	}

	if len(azureStorageRequestV2.RegionalStorageConnectionString) == 0 {
		return &JsonErr{Code: InvalidAzureStorageInfo, Text: "Regional Azure Storage connection string cannot be empty.",
			HTTPStatus: http.StatusBadRequest}
	}

	return nil
}
