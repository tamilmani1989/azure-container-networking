package cnsinterface

import (
	"dnc/api"
)

// CnsInterface has the interface for DNC to communicate goal state with CNS
type CnsInterface interface {
	GetNCLimitForNode(nodeID string) (int, *api.JsonErr)
}