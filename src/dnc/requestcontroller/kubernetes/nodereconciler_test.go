// Copyright 2020 Microsoft. All rights reserved.

package kubernetes

import (
	"context"
	"testing"
	"time"

	syncdncclient "dnc/requestcontroller/dncclient"

	acnv1alpha "github.com/Azure/azure-container-networking/nodenetworkconfig/api/v1alpha"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// TestReconcileNewNode tests that a NodeNetworkConfig is created for a new Node
func TestReconcileNewNode(t *testing.T) {
	namespace := "test"
	nodeName := "test-node"
	mockClient := newMockK8sClient()
	nodeReconciler := NewNodeReconciler(NodeReconcilerOptions{
		K8sClient:      mockClient,
		DNCClient:      &syncdncclient.MockSynchronousDNCClient{},
		Namespace:      namespace,
		InitialIPCount: 8,
		Logger:         logger,
		Recorder:       &record.FakeRecorder{},
	})

	node := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
			Labels: map[string]string{
				DelegatedSubnetIDLabel: "b7806e74-9fd3-4ff8-8906-6e2278f4415b",
				SubnetNameLabel:        "subnet1",
			},
		},
	}

	err := mockClient.Create(context.Background(), &node)
	if err != nil {
		t.Fatalf("Mock client Create failed with error: %v", err)
	}

	nodeKey := types.NamespacedName{
		Namespace: "",
		Name:      nodeName,
	}
	expectedNNCKey := types.NamespacedName{
		Namespace: namespace,
		Name:      nodeName,
	}

	_, err = nodeReconciler.Reconcile(reconcile.Request{nodeKey})
	if err != nil {
		t.Fatalf("Reconcile failed with error: %v", err)
	}

	// Reconcile should have created a NodeNetworkConfig with same name as testNode in the test namespace
	nnc := acnv1alpha.NodeNetworkConfig{}
	err = mockClient.Get(context.Background(), expectedNNCKey, &nnc)
	if err != nil {
		t.Fatalf("Reconcile failed with error: %v", err)
	}

	if nnc.Name != nodeName {
		t.Fatalf("Reconcile should have created a NodeNetworkConfig with name %s but created %s", nodeName, nnc.Name)
	}
}

//TestReconcileDeletingNode tests that Reconcile will remove the NodeNetworkConfig for a deleting Node that has not yet been removed from the cache
func TestReconcileDeletingNode(t *testing.T) {
	namespace := "test"
	nodeName := "test-node"
	mockClient := newMockK8sClient()
	nodeReconciler := NewNodeReconciler(NodeReconcilerOptions{
		K8sClient:      mockClient,
		DNCClient:      &syncdncclient.MockSynchronousDNCClient{},
		Namespace:      namespace,
		InitialIPCount: 8,
		Logger:         logger,
		Recorder:       &record.FakeRecorder{},
	})

	node := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:              nodeName,
			DeletionTimestamp: &metav1.Time{time.Now()},
		},
	}

	nnc := acnv1alpha.NodeNetworkConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nodeName,
			Namespace: namespace,
		},
	}

	err := mockClient.Create(context.Background(), &node)
	if err != nil {
		t.Fatalf("Mock client Create failed with error: %v", err)
	}

	err = mockClient.Create(context.Background(), &nnc)
	if err != nil {
		t.Fatalf("Mock client Create failed with error: %v", err)
	}

	nodeKey := types.NamespacedName{
		Namespace: "",
		Name:      nodeName,
	}

	nncKey := types.NamespacedName{
		Namespace: namespace,
		Name:      nodeName,
	}

	_, err = nodeReconciler.Reconcile(reconcile.Request{nodeKey})
	if err != nil {
		t.Fatalf("Reconcile failed with error: %v", err)
	}

	// Reconcile should have deleted the NodeNetworkConfig with same name as testNode
	err = mockClient.Get(context.Background(), nncKey, &nnc)
	if err == nil {
		t.Fatalf("Reconcile should have deleted NNC %s but it was found", nncKey.Name)
	}

	if !apierrors.IsNotFound(err) {
		t.Fatalf("Mock client should have returned NotFound error but returned with error: %v", err)
	}
}

//TestReconcileDeletedNode tests that Reconcile will remove the NodeNetworkConfig for a Node that has been deleted and removed from the cache
func TestReconcileDeletedNode(t *testing.T) {
	namespace := "test"
	nodeName := "test-node"
	mockClient := newMockK8sClient()
	nodeReconciler := NewNodeReconciler(NodeReconcilerOptions{
		K8sClient:      mockClient,
		DNCClient:      &syncdncclient.MockSynchronousDNCClient{},
		Namespace:      namespace,
		InitialIPCount: 8,
		Logger:         logger,
		Recorder:       &record.FakeRecorder{},
	})

	nnc := acnv1alpha.NodeNetworkConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nodeName,
			Namespace: namespace,
		},
	}

	err := mockClient.Create(context.Background(), &nnc)
	if err != nil {
		t.Fatalf("Mock client Create failed with error: %v", err)
	}

	nodeKey := types.NamespacedName{
		Namespace: "",
		Name:      nodeName,
	}

	nncKey := types.NamespacedName{
		Namespace: namespace,
		Name:      nodeName,
	}

	_, err = nodeReconciler.Reconcile(reconcile.Request{nodeKey})
	if err != nil {
		t.Fatalf("Reconcile failed with error: %v", err)
	}

	// Reconcile should have deleted the NodeNetworkConfig with same name as testNode
	err = mockClient.Get(context.Background(), nncKey, &nnc)
	if err == nil {
		t.Fatalf("Reconcile should have deleted NNC %s but it was found", nncKey.Name)
	}

	if !apierrors.IsNotFound(err) {
		t.Fatalf("Mock client should have returned NotFound error but returned with error: %v", err)
	}
}
