package cnsinterface

import (
	"dnc/api"
	"dnc/datamodel"
)

// CnsInterface has the interface for DNC to communicate goal state with CNS
type CnsInterface interface {
	GetNCLimitForNode(nodeID string) (int, *api.JsonErr)
	DeleteNC(nc *datamodel.NCConfiguration, ncLock map[string]string) *api.JsonErr
	SetOrchestratorType(nodeID, nodeAddress, orchestratorType string) *api.JsonErr
}

const (
	Direct   = "direct"
	Managed  = "managed"
	Disabled = "disabled"
)
