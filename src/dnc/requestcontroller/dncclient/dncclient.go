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
)

//SynchronousDNCClient is a DNC client wrapping async http calls into synchronous functions
type SynchronousDNCClient struct {
	dncClient *client.DncClient
	retrier   retry.Retrier
}

var _ requestcontroller.SynchronousDNCClient = SynchronousDNCClient{}

// NewSynchronousDNCClient returns a new SynchronousDNCClient
func NewSynchronousDNCClient(dncClient *client.DncClient, pollingInterval time.Duration) SynchronousDNCClient {
	retrier := retry.Retrier{
		Attempts: math.MaxInt32,
		Delay:    pollingInterval,
	}
	syncDNCClient := SynchronousDNCClient{
		dncClient: dncClient,
		retrier:   retrier,
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

// UpsertNode adds/updates the Node on DNC
func (s SynchronousDNCClient) UpsertNode(ctx context.Context, nodeID, ipAddress, orchestratorType, infraVnet string) error {
	// Send the AddNode request
	addNodeRespErr := s.dncClient.AddNode(nodeID, ipAddress, orchestratorType, infraVnet)
	// http.StatusOK means request is accepted, anything else means there was an error
	if addNodeRespErr.HTTPStatus != http.StatusOK {
		return addNodeRespErr
	}

	var respErr *api.JsonErr

	err := s.retrier.Do(ctx, func() error {
		_, respErr = s.dncClient.GetNode(nodeID)
		// http.StatusAccepted means request is still pending, anything else means it is done
		if respErr.HTTPStatus != http.StatusAccepted {
			return nil
		}
		return respErr
	})

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
	// Send the DeleteNode request
	deleteNodeResp := s.dncClient.DeleteNode(nodeID)
	// http.StatusOK means request is accepted, anything else means there was an error
	if deleteNodeResp.HTTPStatus != http.StatusOK {
		return deleteNodeResp
	}

	var respErr *api.JsonErr

	err := s.retrier.Do(ctx, func() error {
		_, respErr := s.dncClient.GetNode(nodeID)
		// http.StatusAccepted means request is still pending, anything else means it is done
		if respErr.HTTPStatus != http.StatusAccepted {
			return nil
		}
		return respErr
	})

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
	// TODO: update this to use context when dnc/client functions accept context
	ncResp, respErr := s.dncClient.GetNC(networkID, ncID)
	if respErr != nil && respErr.HTTPStatus == http.StatusOK {
		return ncResp, nil
	}

	return ncResp, respErr
}

//UpsertNcV2 adds/updates an NC on DNC and returns an error if the processing fails
func (s SynchronousDNCClient) UpsertNcV2(ctx context.Context, networkID, ncID string, req api.NetworkContainerRequestV2) error {
	upsertNCRespErr := s.dncClient.UpsertNcV2(networkID, ncID, req)
	// http.StatusOK means request is accepted, anything else means there was an error
	if upsertNCRespErr != nil && upsertNCRespErr.HTTPStatus != http.StatusOK {
		return upsertNCRespErr
	}

	var ncStatus *api.RequestStatus
	var respErr *api.JsonErr

	err := s.retrier.Do(ctx, func() error {
		ncStatus, respErr = s.dncClient.GetNCStatus(networkID, ncID)
		// http.StatusAccepted means request is still pending, anything else means it is done
		if respErr.HTTPStatus != http.StatusAccepted {
			return nil
		}
		return respErr
	})

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
	deleteNCResp := s.dncClient.DeleteNC(networkID, ncID)
	if deleteNCResp.HTTPStatus != http.StatusOK {
		return deleteNCResp
	}

	var respErr *api.JsonErr

	err := s.retrier.Do(ctx, func() error {
		_, respErr = s.dncClient.GetNC(networkID, ncID)
		// http.StatusAccepted means request is still pending, anything else means it is done
		if respErr.HTTPStatus != http.StatusAccepted {
			return nil
		}
		return respErr
	})

	// If the context was cancelled or max attempts reached
	if err != nil {
		return err
	}

	if respErr.HTTPStatus != http.StatusNotFound {
		return fmt.Errorf("Delete nc %s failed with response: %+v", ncID, respErr)
	}

	return nil
}

//RegisterVnet registers the vnet on DNC
func (s SynchronousDNCClient) RegisterVnet(ctx context.Context, networkID string) error {
	addNetworkRespErr := s.dncClient.AddNetwork(networkID)
	// http.StatusOK means request is accepted, anything else means there was an error
	if addNetworkRespErr.HTTPStatus != http.StatusOK {
		return addNetworkRespErr
	}

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

	// If the context was cancelled or max attempts reached
	if err != nil {
		return err
	}

	if vnetStatus.State != "Join" || vnetStatus.Status != "Completed" {
		return fmt.Errorf("Register network %s failed with response: %+v", networkID, vnetStatus)
	}

	return nil
}

//RegisterSubnet registers the subnet on DNC
func (s SynchronousDNCClient) RegisterSubnet(ctx context.Context, networkID, subnetID, authenticationToken string) error {
	addSubnetRespErr := s.dncClient.AddSubnet(networkID, subnetID, authenticationToken)
	// http.StatusOK means request is accepted, anything else means there was an error
	if addSubnetRespErr.HTTPStatus != http.StatusOK {
		return addSubnetRespErr
	}

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
