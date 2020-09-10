// Copyright 2020 Microsoft. All rights reserved.

package kubernetes

import (
	"context"
	"dnc/requestcontroller"
	syncdncclient "dnc/requestcontroller/dncclient"
	"fmt"
	"sync"
	"testing"

	acnv1alpha "github.com/Azure/azure-container-networking/nodenetworkconfig/api/v1alpha"
	cache "github.com/patrickmn/go-cache"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type mockSubnetInfoProvider struct {
	subnetInfo map[string]map[string]requestcontroller.SubnetInfo // delegatedSubnetID -> {subnetName -> SubnetInfo, subnetName -> SubnetInfo}
	mutex      sync.Mutex
}

func (m *mockSubnetInfoProvider) GetSubnetInfo(ctx context.Context, delegatedSubnetID, subnetName string) (requestcontroller.SubnetInfo, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, foundDelegatedSubnet := m.subnetInfo[delegatedSubnetID]; foundDelegatedSubnet {
		if subnetInfo, foundSubnetName := m.subnetInfo[delegatedSubnetID][subnetName]; foundSubnetName {
			return subnetInfo, nil
		}
	}

	return requestcontroller.SubnetInfo{}, fmt.Errorf("subnet %s info not found for delegatedSubnetID %s", subnetName, delegatedSubnetID)
}

// TestReconcileNewNNC tests that the node and NC are created on DNC
func TestReconcileNewNNC(t *testing.T) {
	nodeName := "test-node"
	namespace := "test-namespace"
	requestedIPCount := 16
	delegatedSubnetID := "unique-delegated-subnet-id"
	vnetGUID := "93ca9913-dc26-40fe-9e16-9164669590a0"
	subnetName := "test-subnet"
	subnetToken := "bd48a062-efa7-4c50-af4b-9c97add5b8cc"

	k8sClient := newMockK8sClient()
	dncClient := syncdncclient.MockSynchronousDNCClient{}

	nncReconciler := NodeNetworkConfigReconciler{
		Client:    k8sClient,
		DncClient: &dncClient,
		ScalerConfig: requestcontroller.RequestScalerConfig{
			BatchSize:               16,
			ReleaseThresholdPercent: 50,
			RequestThresholdPercent: 150,
		},
		SubnetInfoProvider: &mockSubnetInfoProvider{
			subnetInfo: map[string]map[string]requestcontroller.SubnetInfo{
				delegatedSubnetID: {
					subnetName: requestcontroller.SubnetInfo{
						DelegatedSubnetID: delegatedSubnetID,
						SubnetName:        subnetName,
						VnetID:            vnetGUID,
						PrimaryToken:      subnetToken,
					},
				},
			},
		},
		NodeIDToNCID:      cache.New(cache.NoExpiration, cache.NoExpiration),
		RegisteredNodes:   cache.New(cache.NoExpiration, cache.NoExpiration),
		RegisteredVnets:   cache.New(cache.NoExpiration, cache.NoExpiration),
		RegisteredSubnets: cache.New(cache.NoExpiration, cache.NoExpiration),
	}

	nnc := acnv1alpha.NodeNetworkConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nodeName,
			Namespace: namespace,
		},
		Spec: acnv1alpha.NodeNetworkConfigSpec{
			RequestedIPCount: int64(requestedIPCount),
		},
	}
	node := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
			Labels: map[string]string{
				DelegatedSubnetIDLabel: delegatedSubnetID,
				SubnetNameLabel:        subnetName,
			},
		},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{
					Type:    corev1.NodeInternalIP,
					Address: "10.1.0.1",
				},
			},
		},
	}

	err := k8sClient.Create(context.Background(), &nnc)
	if err != nil {
		t.Fatalf("Failed to create NNC %s on apiserver with error %v", nodeName, err)
	}
	err = k8sClient.Create(context.Background(), &node)
	if err != nil {
		t.Fatalf("Failed to create node %s on apiserver with error %v", nodeName, err)
	}

	_, err = nncReconciler.Reconcile(reconcile.Request{types.NamespacedName{Namespace: namespace, Name: nodeName}})

	if err != nil {
		t.Fatalf("Failed to reconcile NNC %s with error %v", nodeName, err)
	}

	nodeInfoResp, err := dncClient.GetNode(context.Background(), nodeName)
	if err != nil {
		t.Fatalf("Expected to receive node from DNC but received error %+v", err)
	}

	if len(nodeInfoResp.NetworkContainers) != 1 {
		t.Fatalf("Expected 1 NC for node %s but received %d", nodeName, len(nodeInfoResp.NetworkContainers))
	}

	ipCount := len(nodeInfoResp.NetworkContainers[0].IPConfiguration.SecondaryIPs)
	if ipCount != requestedIPCount {
		t.Fatalf("Expected %d IPs for NC but received %d", requestedIPCount, ipCount)
	}

}
