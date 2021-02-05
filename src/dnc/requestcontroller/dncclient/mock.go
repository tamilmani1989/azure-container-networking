//Copyright 2020 Microsoft. All rights reserved.

package dncclient

import (
	"context"
	"dnc/api"
	"dnc/requestcontroller"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

type MockSynchronousDNCClient struct {
	vnets   map[string]bool
	subnets map[string]bool
	nodes   map[string]*api.NodeInfoResponse
}

var _ requestcontroller.SynchronousDNCClient = &MockSynchronousDNCClient{}

func (m *MockSynchronousDNCClient) GetNode(ctx context.Context, nodeID string) (*api.NodeInfoResponse, error) {
	fmt.Printf("GetNode(ctx, %s)\n", nodeID)
	node, found := m.nodes[nodeID]
	if !found {
		return nil, &api.JsonErr{
			Code:       api.NodeNotFound,
			Text:       "Node not found",
			HTTPStatus: http.StatusNotFound,
		}
	}
	return node, nil
}

func (m *MockSynchronousDNCClient) UpsertNode(ctx context.Context, nodeID, ipAddress, orchestratorType, infraVnet string) error {
	fmt.Printf("UpsertNode(ctx, %s, %s, %s, %s)\n", nodeID, ipAddress, orchestratorType, infraVnet)
	node, found := m.nodes[nodeID]
	var ncs []api.NetworkContainerResponse
	if found {
		for _, nc := range node.NetworkContainers {
			ncs = append(ncs, nc)
		}
	}

	node = &api.NodeInfoResponse{
		NodeInfo: api.NodeInfoRequest{
			IPAddresses:           []string{ipAddress},
			OrchestratorType:      orchestratorType,
			InfrastructureNetwork: infraVnet,
		},
		NetworkContainers: ncs,
	}

	if m.nodes == nil {
		m.nodes = make(map[string]*api.NodeInfoResponse)
	}
	m.nodes[nodeID] = node

	return nil
}

func (m *MockSynchronousDNCClient) DeleteNode(ctx context.Context, nodeID string) error {
	fmt.Printf("DeleteNode(ctx, %s)\n", nodeID)
	node, found := m.nodes[nodeID]
	if !found {
		return nil
	}

	if len(node.NetworkContainers) != 0 {
		return &api.JsonErr{
			Code:       api.NodeInUse,
			Text:       "Node still has NC(s)",
			HTTPStatus: http.StatusBadRequest, // TODO: validate this status code on real DNC
		}
	}

	delete(m.nodes, nodeID)
	return nil
}

func (m *MockSynchronousDNCClient) GetNC(ctx context.Context, networkID, ncID string) (*api.NetworkContainerResponse, error) {
	fmt.Printf("GetNC(ctx, %s, %s)\n", networkID, ncID)
	for _, nodeInfo := range m.nodes {
		if nodeInfo.NodeInfo.InfrastructureNetwork == networkID {
			for _, nc := range nodeInfo.NetworkContainers {
				if nc.NcID == ncID {
					return &nc, nil
				}
			}
		}
	}

	return nil, &api.JsonErr{
		Code:       api.NetworkContainerNotFound,
		Text:       "nc not found",
		HTTPStatus: http.StatusNotFound,
	}
}

func (m *MockSynchronousDNCClient) UpsertNcV2(ctx context.Context, networkID, ncID string, req api.NetworkContainerRequestV2) error {
	fmt.Printf("UpsertNcV2(ctx, %s, %s, %+v)\n", networkID, ncID, req)
	nodeID := req.AssociationInfo.NodeID
	node, found := m.nodes[nodeID]
	if !found {
		return &api.JsonErr{
			Code:       api.NodeNotRegistered,
			Text:       "node not registered, cannot add nc",
			HTTPStatus: http.StatusBadRequest,
		}
	}

	secondaryIPs := make(map[string]api.IPInfo)
	for i := 5; i < 5+req.AllocationRequest.SecondaryIPCount; i++ {
		secondaryIPs[uuid.New().String()] = api.IPInfo{
			IPAddress: fmt.Sprintf("10.0.0.%d", i),
		}
	}

	indexOfExistingNC := -1
	for i, nc := range node.NetworkContainers {
		if nc.NcID == ncID {
			indexOfExistingNC = i
			break
		}
	}

	nc := api.NetworkContainerResponse{
		NcID: ncID,
		IPConfiguration: api.IPConfiguration{
			NetworkID:    networkID,
			SubnetName:   req.AllocationRequest.SubnetName,
			IPSubnet:     "10.0.0.0/24",
			IPAddress:    "10.0.0.4/32",
			GatewayIP:    "10.0.0.1",
			DNSServers:   []string{"8.8.8.8"},
			SecondaryIPs: secondaryIPs,
		},
	}

	if indexOfExistingNC >= 0 {
		node.NetworkContainers[indexOfExistingNC] = nc
	} else {
		node.NetworkContainers = append(node.NetworkContainers, nc)
	}
	m.nodes[nodeID] = node

	return nil
}

func (m *MockSynchronousDNCClient) DeleteNC(ctx context.Context, networkID, ncID string) error {
	fmt.Printf("DeleteNC(ctx, %s, %s)\n", networkID, ncID)
	for nodeID, nodeInfo := range m.nodes {
		if nodeInfo.NodeInfo.InfrastructureNetwork == networkID {
			var ncs []api.NetworkContainerResponse
			for _, nc := range nodeInfo.NetworkContainers {
				if nc.NcID == ncID {
					continue
				}
				ncs = append(ncs, nc)
			}
			m.nodes[nodeID].NetworkContainers = ncs
		}
	}

	return nil
}

func (m *MockSynchronousDNCClient) RegisterVnet(ctx context.Context, networkID string) error {
	fmt.Printf("RegisterVnet(ctx, %s)\n", networkID)
	// Not doing anything with this yet, but store it anyway
	if m.vnets == nil {
		m.vnets = make(map[string]bool)
	}
	m.vnets[networkID] = true
	return nil
}

func (m *MockSynchronousDNCClient) UnregisterVnet(ctx context.Context, networkID string) error {
	fmt.Printf("UnregisterVnet(ctx, %s)\n", networkID)
	if m.vnets != nil {
		delete(m.vnets, networkID)
	}

	return nil
}

func (m *MockSynchronousDNCClient) RegisterSubnet(ctx context.Context, networkID, subnetID, authenticationToken string) error {
	fmt.Printf("RegisterSubnet(ctx, %s, %s, %s)\n", networkID, subnetID, authenticationToken)
	// Not doing anything with this yet, but store it anyway
	if m.subnets == nil {
		m.subnets = make(map[string]bool)
	}
	m.subnets[subnetID] = true
	return nil
}

// GetVnets returns a list of registered networks
func (m *MockSynchronousDNCClient) GetVnets(ctx context.Context) (*api.NetworksResponse, error) {
	var networkInfos []api.NetworkInfo

	for k, _ := range m.vnets {
		networkInfos = append(networkInfos, api.NetworkInfo{GUID: k})
	}

	return &api.NetworksResponse{
		NetworkInfos: networkInfos,
	}, nil
}
