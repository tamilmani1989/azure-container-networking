package cnsinterface

import (
	"dnc/api"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/utils"
	"fmt"
	"net/http"
)

// DirectClient - Struct to store the variables which client needs
type DirectClient struct {
	sessionManager  *utils.SessionManager
	enforceNicLimit bool
}

// NewDirectClient - Create a new CNS shim
func NewDirectClient(sessionManager *utils.SessionManager, enforceNicLimit bool) *DirectClient {
	directClient := &DirectClient{
		sessionManager:  sessionManager,
		enforceNicLimit: enforceNicLimit,
	}
	return directClient
}

// GetNCLimitForNode returns how many NCs can be created in this node
func (client *DirectClient) GetNCLimitForNode(nodeID string) (int, *api.JsonErr) {
	if !client.enforceNicLimit {
		return utils.MaxContainersPerNode, nil
	}

	nodeConfig, err := datamodel.TryQueryNodeConfiguration(nodeID)

	if err != nil {
		errMsg := fmt.Sprintf("Unable to get NC limit for the node because "+
			"failure to get the node configuration for node: %s due to error: %v", nodeID, err)
		return 0, &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       errMsg,
			HTTPStatus: http.StatusInternalServerError}
	}

	if nodeConfig == nil || &nodeConfig.NodeInfo == nil || nodeConfig.NodeInfo.IPAddresses == nil || len(nodeConfig.NodeInfo.IPAddresses) < 1 {
		errMsg := fmt.Sprintf("Config for node: %s doesn't have IP address info", nodeID)
		return 0, &api.JsonErr{
			Code:       api.InvalidNodeIPAddress,
			Text:       errMsg,
			HTTPStatus: http.StatusInternalServerError}
	}

	loggershim.LS.Debugf("Try to retrieve NC limit for node: %s with node config %+v", nodeID, nodeConfig)
	ncLimit, respErr := datamodel.GetNCLimitForNode(nodeConfig.NodeInfo.IPAddresses[0])
	if respErr != nil {
		loggershim.LS.Debugf("Failed to retrieve NC limit for node: %s with error: %v", nodeID, respErr.String())
		return 0, respErr
	}

	return ncLimit, nil
}
