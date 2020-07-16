package datamodel

import (
	"fmt"
	"time"
)

// NcOperationType - Type of the NC Operation type
type NcOperationType int

const (
	// InvalidNetworkContainerRequest - Not Used
	InvalidNetworkContainerRequest NcOperationType = iota

	// NetworkContainerPublishRequest - Publish NC request
	NetworkContainerPublishRequest

	// NetworkContainerUnpublishRequest - Unpublish NC request
	NetworkContainerUnpublishRequest
)

func (operationType NcOperationType) String() string {
	switch operationType {

	case NetworkContainerPublishRequest:
		return "NetworkContainerPublishRequest"

	case NetworkContainerUnpublishRequest:
		return "NetworkContainerUnpublishRequest"

	case InvalidNetworkContainerRequest:
		return "InvalidNetworkContainerRequest"
	}

	return "Unknown Status"
}

// NcPendingPublishRequestContext - Request context for pending publish requests for NC
type NcPendingPublishRequestContext struct {
	Type                NcOperationType
	NetworkID           string
	NetworkContainerID  string
	InfraNetworkID      string
	NodeInterfaceIP     string
	PubSubID            int
	NcData              []byte
	CreatedDateTime     time.Time
	LastRetriedDateTime time.Time
	RetryInterval       time.Duration
}

// String - ToString implementation of request context.
func (requestContext *NcPendingPublishRequestContext) String() string {
	return fmt.Sprintf("NcPendingPublishRequestContext:[Type: %s, NetworkID: %s, NCID: %s, InfraNetworkID: %s, "+
		"NodeInterfaceIP: %s, PubSubID:%d, NcData: %v, CreatedDateTime: %s, LastRetriedDateTime: %s, RetryInterval: %v]",
		requestContext.Type,
		requestContext.NetworkID,
		requestContext.NetworkContainerID,
		requestContext.InfraNetworkID,
		requestContext.NodeInterfaceIP,
		requestContext.PubSubID,
		requestContext.NcData,
		requestContext.CreatedDateTime.String(),
		requestContext.LastRetriedDateTime.String(),
		requestContext.RetryInterval)
}
