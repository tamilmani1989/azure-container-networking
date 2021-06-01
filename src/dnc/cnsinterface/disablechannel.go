package cnsinterface

import (
	"dnc/api"
	"dnc/datamodel"
)

// nodeInfoFetcher gets node info, wraps dnc/controller.Controller
// Need to use an interface here because we can't import dnc/controller directly
type nodeInfoFetcher interface {
	GetNodeInfo(string) (*api.NodeInfoResponse, *api.JsonErr)
}

// DisabledChannel bypasses CNS interaction, and returns fixed values
type DisabledChannel struct {
	NodeInfoFetcher nodeInfoFetcher
	// NICsPerVirtualCore
	NICsPerVirtualCore int
	// DefaultNCLimit NC limit to use when not provided in node info request
	DefaultNCLimit int
}

func (n *DisabledChannel) GetNCLimitForNode(nodeID string) (int, *api.JsonErr) {
	// If we have the info in NodeInfo, use it
	nodeInfo, jsonErr := n.NodeInfoFetcher.GetNodeInfo(nodeID)
	if jsonErr != nil {
		return 0, jsonErr
	}

	if nodeInfo.NodeInfo.NumCores != 0 {
		return nodeInfo.NodeInfo.NumCores * n.NICsPerVirtualCore, nil
	}

	return n.DefaultNCLimit, nil
}

func (n *DisabledChannel) DeleteNC(_ *datamodel.NCConfiguration, _ map[string]string) *api.JsonErr {
	return nil
}

func (n *DisabledChannel) SetOrchestratorType(_, _, _ string) *api.JsonErr {
	return nil
}
