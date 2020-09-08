// Copyright 2020 Microsoft. All rights reserved.

package kubernetes

import (
	"context"
	"dnc/requestcontroller"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NodeInfoProvider gives subnet/vnet info for nodes
type NodeInfoProvider struct {
	Client           K8sClient
	VnetInfoProvider requestcontroller.VnetInfoProvider
}

var _ requestcontroller.NodeInfoProvider = NodeInfoProvider{}

// GetVnetID gets the vnet id for the given node ID
func (n NodeInfoProvider) GetVnetID(ctx context.Context, nodeID string) (string, error) {
	// client.Client pulls from cache so it is ok to use Client.Get every time
	node := corev1.Node{}
	err := n.Client.Get(ctx, client.ObjectKey{Name: nodeID}, &node)
	if err != nil {
		return "", err
	}

	delegatedSubnetID, found := node.Labels[DelegatedSubnetIDLabel]
	if !found {
		return "", fmt.Errorf("Label %s not found for node %s", DelegatedSubnetIDLabel, nodeID)
	}

	vnetID, err := n.VnetInfoProvider.GetVnetID(ctx, delegatedSubnetID)
	if err != nil {
		return "", err
	}

	return vnetID, nil
}

// GetSubnetName gets the subnet name for the given node
func (n NodeInfoProvider) GetSubnetName(ctx context.Context, nodeID string) (string, error) {
	// client.Client pulls from cache so it is ok to use Client.Get every time
	node := corev1.Node{}
	err := n.Client.Get(ctx, client.ObjectKey{Name: nodeID}, &node)
	if err != nil {
		return "", err
	}

	subnetName, found := node.Labels[SubnetNameLabel]
	if !found {
		return "", fmt.Errorf("Label %s not found for node %s", SubnetNameLabel, nodeID)
	}

	return subnetName, nil
}

type NodeLabelVnetInfoProvider struct{}

var _ requestcontroller.VnetInfoProvider = NodeLabelVnetInfoProvider{}

func (n NodeLabelVnetInfoProvider) GetVnetID(ctx context.Context, delegatedSubnetID string) (string, error) {
	// For AKS scenario, delegatedSubnetID label is the vnet guid, so just return that here
	return delegatedSubnetID, nil
}
