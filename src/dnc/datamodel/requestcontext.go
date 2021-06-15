package datamodel

import (
	"time"
)

// RequestContext - Request context
type RequestContext struct {
	NetworkID               string
	NodeID                  string
	SubnetName              string
	Type                    RequestType
	CreatedDateTime         time.Time
	StateTransitionDateTime time.Time
	LastRetriedDateTime     time.Time
	RetryInterval           time.Duration
}
