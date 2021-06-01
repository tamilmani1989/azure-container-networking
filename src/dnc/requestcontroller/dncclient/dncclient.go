//Copyright 2020 Microsoft. All rights reserved.

//Package dncclient implements a synchronous DNC client
//TODO: move this to dnc/client package
package dncclient

import (
	"context"
	"dnc/api"
	"dnc/client"
	"dnc/requestcontroller"
	"dnc/retry"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

//SynchronousDNCClient is a DNC client wrapping async http calls into synchronous functions
type SynchronousDNCClient struct {
	dncClient *client.DncClient
	retrier   retry.Retrier
	logger    logrus.FieldLogger
}

var _ requestcontroller.SynchronousDNCClient = SynchronousDNCClient{}

// NewSynchronousDNCClient returns a new SynchronousDNCClient
func NewSynchronousDNCClient(dncClient *client.DncClient, pollingInterval time.Duration, logger logrus.FieldLogger) SynchronousDNCClient {
	retrier := retry.Retrier{
		Attempts: math.MaxInt32,
		Delay:    pollingInterval,
	}
	syncDNCClient := SynchronousDNCClient{
		dncClient: dncClient,
		retrier:   retrier,
		logger:    logger,
	}

	return syncDNCClient
}

//GetNode wraps dnc/client's GetNode to present errors as standard error type
func (s SynchronousDNCClient) GetNode(ctx context.Context, nodeID string) (*api.NodeInfoResponse, error) {
	// TODO: update this to use context when dnc/client functions accept context
	nodeResp, respErr := s.dncClient.GetNode(nodeID)
	if respErr != nil && respErr.HTTPStatus == http.StatusOK {
		return nodeResp, nil
	}

	return nodeResp, respErr
}

//GetNodes wraps dnc/client's GetNodes to present errors as standard error type
func (s SynchronousDNCClient) GetNodes(ctx context.Context) (*api.NodesResponse, error) {
	// TODO: update this to use context when dnc/client functions accept context
	nodesResp, respErr := s.dncClient.GetNodes("") // TODO: support nodeSets in DNC-RC
	if respErr != nil && respErr.HTTPStatus == http.StatusOK {
		return nodesResp, nil
	}

	return nodesResp, respErr
}

// UpsertNode adds/updates the Node on DNC
func (s SynchronousDNCClient) UpsertNode(ctx context.Context, nodeID, ipAddress, orchestratorType, infraVnet string) error {
	logger := s.logger.WithFields(logrus.Fields{
		"nodeID":             nodeID,
		"ipAddress":          ipAddress,
		"orchestratorType":   orchestratorType,
		"infrastructureVnet": infraVnet,
	})

	// Send the AddNode request
	logger.Infof("DNC call to Add node: NodeID: %s, ipAddress:%s, orchType:%s, infravnet:%s", nodeID, ipAddress,
		orchestratorType, infraVnet)
	addNodeRespErr := s.dncClient.AddNode(nodeID, ipAddress, orchestratorType, infraVnet, "", "", "", 0) // TODO: support nodeSets in DNC-RC
	// http.StatusOK means request is accepted, anything else means there was an error
	if addNodeRespErr.HTTPStatus != http.StatusOK {
		return addNodeRespErr
	}

	logger.Infof("Add Node succeeded for %s. Call dnc get node", nodeID)

	var respErr *api.JsonErr

	err := s.retrier.Do(ctx, func() error {
		_, respErr = s.dncClient.GetNode(nodeID)

		// http.StatusAccepted means request is still pending, anything else means it is done
		if respErr.HTTPStatus != http.StatusAccepted {
			return nil
		}
		return respErr
	})

	logger.Infof("UpsertNode: DNC call GetNode status for nodeID:%s is :%+v", nodeID, respErr)
	// If the context was cancelled or max attempts reached
	if err != nil {
		return err
	}

	if respErr.HTTPStatus != http.StatusOK {
		return respErr
	}

	return nil
}

//DeleteNode deletes the node
func (s SynchronousDNCClient) DeleteNode(ctx context.Context, nodeID string) error {
	logger := s.logger.WithFields(logrus.Fields{
		"nodeID": nodeID,
	})
	// Send the DeleteNode request
	logger.Infof("DNC call to DeleteNode node: NodeID: %s", nodeID)
	deleteNodeResp := s.dncClient.DeleteNode(nodeID)
	// http.StatusOK means request is accepted, anything else means there was an error
	if deleteNodeResp.HTTPStatus != http.StatusOK {
		return deleteNodeResp
	}

	logger.Infof("Delete Node succeeded for %s. Call dnc get node", nodeID)
	var respErr *api.JsonErr

	err := s.retrier.Do(ctx, func() error {
		_, respErr = s.dncClient.GetNode(nodeID)
		// http.StatusAccepted means request is still pending, anything else means it is done
		if respErr.HTTPStatus != http.StatusAccepted {
			return nil
		}
		return respErr
	})

	logger.Infof("DeleteNode: DNC call GetNode status for nodeID:%s is :%+v", nodeID, respErr)
	// If the context was cancelled or max attempts reached
	if err != nil {
		return err
	}

	if respErr.HTTPStatus != http.StatusNotFound {
		return respErr
	}

	return nil
}

//GetNC wraps dnc/client's GetNC to present errors as standard error type
func (s SynchronousDNCClient) GetNC(ctx context.Context, networkID, ncID string) (*api.NetworkContainerResponse, error) {
	logger := s.logger.WithFields(logrus.Fields{
		"vnetID": networkID,
		"ncID":   ncID,
	})
	logger.Infof("DNC call to Get NC ncID:%s, networkID:%s", ncID, networkID)
	// TODO: update this to use context when dnc/client functions accept context
	ncResp, respErr := s.dncClient.GetNC(networkID, ncID)
	if respErr != nil && respErr.HTTPStatus == http.StatusOK {
		return ncResp, nil
	}

	return ncResp, respErr
}

//UpsertNcV2 adds/updates an NC on DNC and returns an error if the processing fails
func (s SynchronousDNCClient) UpsertNcV2(ctx context.Context, networkID, ncID string, req api.NetworkContainerRequestV2) error {
	logger := s.logger.WithFields(logrus.Fields{
		"vnetID": networkID,
		"ncID":   ncID,
	})
	logger.Infof("DNC call to Create NC ncID:%s, networkID:%s", ncID, networkID)
	upsertNCRespErr := s.dncClient.UpsertNcV2(networkID, ncID, req)
	// http.StatusOK means request is accepted, anything else means there was an error
	if upsertNCRespErr != nil && upsertNCRespErr.HTTPStatus != http.StatusOK {
		return upsertNCRespErr
	}

	logger.Infof("Create NC succeeded for %s. Call dnc get nc", ncID)

	var ncStatus *api.NCRequestStatus
	var respErr *api.JsonErr

	err := s.retrier.Do(ctx, func() error {
		ncStatus, respErr = s.dncClient.GetNCStatus(networkID, ncID)
		// http.StatusAccepted means request is still pending, anything else means it is done
		if respErr.HTTPStatus != http.StatusAccepted {
			return nil
		}
		return respErr
	})

	logger.Infof("UpsertNcV2: DNC call Get NC status for id %s is :%+v, resperr:%v", ncID, ncStatus, respErr)
	// If the context was cancelled or max attempts reached
	if err != nil {
		return err
	}

	if respErr.HTTPStatus != http.StatusOK {
		return fmt.Errorf("UpsertNC %s failed for vnet %s with error %+v", ncID, networkID, respErr)
	}

	if ncStatus.Status != "Completed" || (ncStatus.State != "Program" && ncStatus.State != "RefreshAssociation") {
		return fmt.Errorf("UpsertNC %s failed for vnet %s with error %+v", ncID, networkID, ncStatus)
	}

	return nil
}

//DeleteNC deletes the NC on DNC
func (s SynchronousDNCClient) DeleteNC(ctx context.Context, networkID, ncID string) error {
	logger := s.logger.WithFields(logrus.Fields{
		"vnetID": networkID,
		"ncID":   ncID,
	})
	logger.Infof("DNC call to Delete NC ncID:%s, networkID:%s", ncID, networkID)
	deleteNCResp := s.dncClient.DeleteNC(networkID, ncID)
	if deleteNCResp.HTTPStatus != http.StatusOK {
		logger.Infof("DeleteNC: DNC call resperr:%+v", deleteNCResp)
		return deleteNCResp
	}

	logger.Infof("Delete NC succeeded for %s. Call dnc get nc cleanup status nc", ncID)

	var respErr *api.JsonErr
	var getNCCleanupStatusResp api.NetworkContainerCleanupResponse

	err := s.retrier.Do(ctx, func() error {
		getNCCleanupStatusResp, respErr = s.dncClient.GetNetworkContainerCleanupStatus(networkID, ncID)
		// 200 status code and {completed: true} body means cleanup is done, anything else means it is not
		if respErr.HTTPStatus == http.StatusOK && getNCCleanupStatusResp.Completed {
			return nil
		}

		// Log if we received a server error
		if respErr.HTTPStatus == http.StatusInternalServerError {
			logger.WithError(respErr).Errorf("Received an internal server error while polling for NC cleanup status: %+v", respErr)
		}

		return respErr
	})

	logger.Infof("DeleteNC: DNC call Get NC cleanup status for id %s, resp: %+v, resperr: %+v", ncID, getNCCleanupStatusResp, respErr)

	// If the context was cancelled or max attempts reached, err will be non-nil
	return err
}

//RegisterVnet registers the vnet on DNC
func (s SynchronousDNCClient) RegisterVnet(ctx context.Context, networkID string) error {
	logger := s.logger.WithFields(logrus.Fields{
		"vnetID": networkID,
	})
	logger.Infof("DNC call to register vnet networkID:%s", networkID)
	addNetworkRespErr := s.dncClient.AddNetwork(networkID)
	// http.StatusOK means request is accepted, anything else means there was an error
	if addNetworkRespErr.HTTPStatus != http.StatusOK {
		return addNetworkRespErr
	}

	logger.Infof("Register Vnet succeeded for %s. Call dnc get vnet", networkID)

	var vnetStatus *api.RequestStatus
	var respErr *api.JsonErr

	err := s.retrier.Do(ctx, func() error {
		vnetStatus, respErr = s.dncClient.GetNetworkStatus(networkID)
		// http.StatusAccepted means request is still pending, anything else means it is done
		if respErr.HTTPStatus != http.StatusAccepted {
			return nil
		}
		return respErr
	})

	logger.Infof("DNC call Get vnet status for id:%s is :%+v resperr:%v", networkID, vnetStatus, respErr)
	// If the context was cancelled or max attempts reached
	if err != nil {
		return err
	}

	if vnetStatus.State != "Join" || vnetStatus.Status != "Completed" {
		return fmt.Errorf("Register network %s failed with response: %+v", networkID, vnetStatus)
	}

	return nil
}

//UnregisterVnet unregisters the vnet on DNC
func (s SynchronousDNCClient) UnregisterVnet(ctx context.Context, networkID string) error {
	logger := s.logger.WithFields(logrus.Fields{
		"vnetGUID": networkID,
	})
	logger.Infof("DNC call to unregister vnet networkID:%s", networkID)
	deleteNetworkRespErr := s.dncClient.DeleteNetwork(networkID)
	// http.StatusOK means request is accepted, anything else means there was an error
	if deleteNetworkRespErr.HTTPStatus != http.StatusOK {
		return deleteNetworkRespErr
	}

	logger.Infof("Unregister Vnet succeeded for %s. Call dnc get vnet", networkID)

	var vnetStatus *api.RequestStatus
	var respErr *api.JsonErr

	err := s.retrier.Do(ctx, func() error {
		vnetStatus, respErr = s.dncClient.GetNetworkStatus(networkID)
		// http.StatusNotFound means it is done, anything else means it is not
		if respErr.HTTPStatus == http.StatusNotFound {
			return nil
		}
		return respErr
	})

	logger.Infof("DNC call Get vnet status for id:%s is :%+v resperr:%v", networkID, vnetStatus, respErr)

	// If the context was cancelled or max attempts reached, err will be non-nil
	return err
}

//RegisterSubnet registers the subnet on DNC
func (s SynchronousDNCClient) RegisterSubnet(ctx context.Context, networkID, subnetID, authenticationToken string) error {
	logger := s.logger.WithFields(logrus.Fields{
		"vnetID":   networkID,
		"subnetID": subnetID,
	})
	logger.Infof("Dnc call to register subnet for networkID:%s subnetID:%s", networkID, subnetID)
	addSubnetRespErr := s.dncClient.AddSubnet(networkID, subnetID, authenticationToken, "")
	// http.StatusOK means request is accepted, anything else means there was an error
	if addSubnetRespErr.HTTPStatus != http.StatusOK {
		return addSubnetRespErr
	}

	logger.Infof("Register Subnet succeeded for %s. Call dnc get subnet", subnetID)

	var subnetStatus *api.RequestStatus
	var respErr *api.JsonErr

	err := s.retrier.Do(ctx, func() error {
		subnetStatus, respErr = s.dncClient.GetSubnetStatus(networkID, subnetID)
		// http.StatusAccepted means request is still pending, anything else means it is done
		if respErr.HTTPStatus != http.StatusAccepted {
			return nil
		}

		return respErr
	})

	logger.Infof("DNC call GetSubnet response for subnetid %s is :%+v, resperr:%+v", subnetID, subnetStatus, respErr)

	// If the context was cancelled or max attempts reached
	if err != nil {
		return err
	}

	if subnetStatus == nil {
		return fmt.Errorf("Unable to retrieve subnet status, error: %+v", respErr)
	}

	if subnetStatus.State != "Join" || subnetStatus.Status != "Completed" {
		return fmt.Errorf("Register subnet %s on network %s failed with response: %+v", subnetID, networkID, subnetStatus)
	}

	return nil
}

// GetVnets returns a list of registered networks
func (s SynchronousDNCClient) GetVnets(ctx context.Context) (*api.NetworksResponse, error) {
	s.logger.Infof("DNC call to get vnet ids")
	vnetsResp, respErr := s.dncClient.GetNetworks()
	if respErr != nil && respErr.HTTPStatus != http.StatusOK {
		return nil, respErr
	}

	return vnetsResp, nil
}
