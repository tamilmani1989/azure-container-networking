package cnsinterface

import (
	"dnc/api"
	internalcns "dnc/cns"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/utils"
	"fmt"
	"net/http"
)

// DirectChannel - Struct to store the variables which client needs
type DirectChannel struct {
	SessionManager  *utils.SessionManager
	EnforceNicLimit bool
	CNSClient       *internalcns.Client
	Logger          loggershim.LoggerInterface
}

func (c *DirectChannel) DeleteNC(nc *datamodel.NCConfiguration, ncLock map[string]string) *api.JsonErr {
	nodeID := nc.GetAssociationInfo().NodeID
	nodeConfig, apiErr := datamodel.NodeConfigRepo.TryQueryNodeConfiguration(nodeID)
	if apiErr != nil {
		return apiErr
	}
	if nodeConfig == nil {
		return api.BadRequestf(api.NodeNotFound, "no node config found for node id %s", nodeID)
	}
	if len(nodeConfig.NodeInfo.IPAddresses) < 1 {
		return api.BadRequestf(api.InvalidNodeIPAddress, "no ip addresses found for node id %s", nodeID)
	}

	// todo: verify that this release/reacquiring the nc lock is really necessary
	c.SessionManager.EndWriteSession(ncLock)
	res, err := c.CNSClient.DeleteNetworkContainer(nodeConfig.NodeInfo.IPAddresses[0], nc.NetworkContainerID)
	c.SessionManager.BeginWriteSession(ncLock)

	if err != nil {
		return api.InternalServerErrorf(api.DisAssociationFailed, "failed to delete NC in CNS: %+v", err)
	}
	if res.Response.ReturnCode != 0 {
		return api.BadRequestf(api.DisAssociationFailed, "could not disassociate NC %s. return code %d. message: %v", nc.NetworkContainerID, res.Response.ReturnCode, res.Response.Message)
	}
	return nil
}

// todo: use DirectChannel dependencies instead of globals
// GetNCLimitForNode returns how many NCs can be created in this node
func (c *DirectChannel) GetNCLimitForNode(nodeID string) (int, *api.JsonErr) {
	if !c.EnforceNicLimit {
		return utils.MaxContainersPerNode, nil
	}

	nodeConfig, err := datamodel.NodeConfigRepo.TryQueryNodeConfiguration(nodeID)

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

	ncLimit, respErr := datamodel.GetNCLimitForNode(nodeConfig.NodeInfo.IPAddresses[0])
	if respErr != nil {
		loggershim.LS.Debugf("Failed to retrieve NC limit for node: %s with error: %v", nodeID, respErr.String())
		return 0, respErr
	}

	return ncLimit, nil
}

func (c *DirectChannel) SetOrchestratorType(nodeID, nodeAddress, orchestratorType string) *api.JsonErr {
	return c.CNSClient.SetOrchestratorType(nodeID, nodeAddress, orchestratorType)
}
