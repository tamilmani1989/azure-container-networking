package cnsinterface

import (
	"dnc/api"
	"dnc/datamodel"
	"dnc/utils"
	"fmt"
	"net/http"
)

// ManagedChannel - CNS interactions for managed DNC
type ManagedChannel struct {
	sessionManager  *utils.SessionManager
	enforceNicLimit bool
}

// NewManagedChannel - Creates a new managed channel
func NewManagedChannel(sessionManager *utils.SessionManager, enforceNicLimit bool) *ManagedChannel {
	cl := &ManagedChannel{
		sessionManager:  sessionManager,
		enforceNicLimit: enforceNicLimit,
	}
	return cl
}

// GetNCLimitForNode returns how many NCs can be created in this node
func (client *ManagedChannel) GetNCLimitForNode(nodeID string) (int, *api.JsonErr) {
	if !client.enforceNicLimit {
		return utils.MaxContainersPerNode, nil
	}

	nodeConfig, err := datamodel.TryQueryNodeConfiguration(nodeID)

	var ncLimit int
	if nodeConfig != nil && err == nil {
		ncLimit = nodeConfig.NcLimit
		if ncLimit == 0 {
			return ncLimit, &api.JsonErr{
				Code:       api.NodeNotRegistered,
				Text:       fmt.Sprintf("Node %s has not registered with DNC", nodeConfig.NodeID),
				HTTPStatus: http.StatusBadRequest}
		}
	} else {
		return 0, &api.JsonErr{
			Code: api.StatusInternalServerError,
			Text: fmt.Sprintf("Unable to get NC limit for the node because "+
				"node configuration for node: %s couldn't be retrieved due to error: %v", nodeID, err),
			HTTPStatus: http.StatusInternalServerError}
	}

	return ncLimit, err
}

// todo: not sure what managed DNC channel behavior should be for nc deletion, implement
func (client *ManagedChannel) DeleteNC(nc *datamodel.NCConfiguration, ncLock map[string]string) *api.JsonErr {
	return nil
}
