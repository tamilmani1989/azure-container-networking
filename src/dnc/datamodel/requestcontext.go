package datamodel

import (
	"fmt"
	"time"
)	

// RequestContext - Request context
type RequestContext struct {
	NetworkID string
	NodeID    string
	Type      RequestType
	CreatedDateTime time.Time 
	LastRetriedDateTime time.Time
}

// String - ToString implementation of request context.
func (requestContext *RequestContext) String() string {
	return fmt.Sprintf("RequestContext:[NetworkID:%s, NodeID:%s, Type:%s, CreatedDateTime:%v, LastRetriedDateTime:%v]",
		requestContext.NetworkID,
		requestContext.NodeID,
		requestContext.Type,
		requestContext.CreatedDateTime.String(),
		requestContext.LastRetriedDateTime.String())
}
