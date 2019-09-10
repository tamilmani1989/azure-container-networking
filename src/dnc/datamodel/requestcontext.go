package datamodel

import (
	"fmt"
	"time"
)

// RequestContext - Request context
type RequestContext struct {
	NetworkID               string
	NodeID                  string
	Type                    RequestType
	CreatedDateTime         time.Time
	StateTransitionDateTime time.Time
	LastRetriedDateTime     time.Time
	RetryInterval           time.Duration
}

// String - ToString implementation of request context.
func (requestContext *RequestContext) String() string {
	return fmt.Sprintf("RequestContext:[NetworkID:%s, NodeID:%s, Type:%s, CreatedDateTime:%s, StateTransitionDateTime:%s, LastRetriedDateTime:%s, RetryInterval:%+v]",
		requestContext.NetworkID,
		requestContext.NodeID,
		requestContext.Type,
		requestContext.CreatedDateTime.String(),
		requestContext.StateTransitionDateTime.String(),
		requestContext.LastRetriedDateTime.String(),
		requestContext.RetryInterval)
}
