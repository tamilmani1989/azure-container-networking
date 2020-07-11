package kubernetes

import (
	"context"
	"fmt"
	"testing"
	"time"

	acnv1alpha "github.com/Azure/azure-container-networking/nodenetworkconfig/api/v1alpha"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type mockNodeReconcilerClient struct {
	nodes              map[string]*corev1.Node
	nodeNetworkConfigs map[string]*acnv1alpha.NodeNetworkConfig
}

// Create always creates a NodeNetworkConfig, so this mock type asserts obj to a NodeNetworkConfig
func (m *mockNodeReconcilerClient) Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
	nodeNetworkConfig := obj.(*acnv1alpha.NodeNetworkConfig)
	_, exists := m.nodeNetworkConfigs[nodeNetworkConfig.Name]
	if exists {
		return apierrors.NewAlreadyExists(schema.GroupResource{Group: "acnv1", Resource: "nodenetworkconfig"}, nodeNetworkConfig.Name)
	}

	m.nodeNetworkConfigs[nodeNetworkConfig.Name] = nodeNetworkConfig

	return nil
}

// Get always gets a node, so this mock type asserts obj to a Node
func (m *mockNodeReconcilerClient) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	node := m.nodes[key.String()]
	if node == nil {
		return apierrors.NewNotFound(schema.GroupResource{Group: "corev1", Resource: "node"}, key.String())
	}
	nodePt := obj.(*corev1.Node)
	node.DeepCopyInto(nodePt)
	return nil
}

// Update always updates a node, so this mock type asserts obj to a Node
func (m *mockNodeReconcilerClient) Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
	node := obj.(*corev1.Node)
	_, exists := m.nodes[fmt.Sprintf("/%s", node.Name)]
	if !exists {
		return apierrors.NewNotFound(schema.GroupResource{Group: "corev1", Resource: "node"}, node.Name)
	}

	node.DeepCopyInto(m.nodes[fmt.Sprintf("/%s", node.Name)])

	return nil
}

// Delete always deletes a NodeNetworkConfig, so this mock type asserts obj to a NodeNetworkConfig
func (m *mockNodeReconcilerClient) Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
	nodeNetworkConfig := obj.(*acnv1alpha.NodeNetworkConfig)
	_, exists := m.nodeNetworkConfigs[nodeNetworkConfig.Name]
	if !exists {
		return apierrors.NewNotFound(schema.GroupResource{Group: "acnv1", Resource: "nodenetworkconfig"}, nodeNetworkConfig.Name)
	}

	delete(m.nodeNetworkConfigs, nodeNetworkConfig.Name)

	return nil
}

// TestReconcileNewNode tests that a new Node gets a finalizer and a NodeNetworkConfig is created for it
func TestReconcileNewNode(t *testing.T) {

	testNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
	}

	mockClient := &mockNodeReconcilerClient{
		nodes: map[string]*corev1.Node{
			fmt.Sprintf("/%s", testNode.ObjectMeta.Name): testNode,
		},
		nodeNetworkConfigs: make(map[string]*acnv1alpha.NodeNetworkConfig),
	}

	nodeReconciler := NodeReconciler{
		Client:         mockClient,
		Namespace:      "test",
		InitialIPCount: 8,
	}

	_, err := nodeReconciler.Reconcile(reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "",
			Name:      "test-node",
		},
	})
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Reconcile should have created a NodeNetworkConfig with same name as testNode
	want := true
	_, got := mockClient.nodeNetworkConfigs[testNode.ObjectMeta.Name]

	if want != got {
		t.Errorf("Reconcile should have created a NodeNetworkConfig with same name as testNode")
	}

	// Reconcile should have added a finalizer to testNode
	want = true
	got = containsString(testNode.Finalizers, NodeFinalizerName)

	if want != got {
		t.Errorf("Reconcile should have added a finalizer %s to testNode", NodeFinalizerName)
	}
}

// TestPartiallyReconciledNode tests that Reconcile will ensure that a NodeNetworkConfig exists for the Node even if it already has a finalizer. This can happen
// if the Node was being reconciled, and then RequestController crashed.
func TestPartiallyReconciledNode(t *testing.T) {
	//testNode has the finalizer
	testNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-node",
			Finalizers: []string{NodeFinalizerName},
		},
	}

	// No NodeNetworkConfig object exists for testNode
	mockClient := &mockNodeReconcilerClient{
		nodes: map[string]*corev1.Node{
			fmt.Sprintf("/%s", testNode.ObjectMeta.Name): testNode,
		},
		nodeNetworkConfigs: make(map[string]*acnv1alpha.NodeNetworkConfig),
	}

	nodeReconciler := NodeReconciler{
		Client:         mockClient,
		Namespace:      "test",
		InitialIPCount: 8,
	}

	_, err := nodeReconciler.Reconcile(reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "",
			Name:      "test-node",
		},
	})
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Reconcile should have created a NodeNetworkConfig with same name as testNode
	want := true
	_, got := mockClient.nodeNetworkConfigs[testNode.ObjectMeta.Name]

	if want != got {
		t.Errorf("Reconcile should have created a NodeNetworkConfig with same name as testNode")
	}

	// Reconcile should not have removed the finalizer from testNode
	want = true
	got = containsString(testNode.Finalizers, NodeFinalizerName)

	if want != got {
		t.Errorf("Reconcile should not have removed the finalizer %s from testNode", NodeFinalizerName)
	}
}

//TestReconcileDeletingNode tests that Reconcile will remove the NodeNetworkConfig object and finalizer from a Node with a non-zero deletion timestamp
func TestReconcileDeletingNode(t *testing.T) {
	// Create a Node with non-zero deletion timestamp to simulate Node deletion
	testNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-node",
			DeletionTimestamp: &metav1.Time{time.Now()},
			Finalizers:        []string{NodeFinalizerName},
		},
	}

	testNamespace := "test"
	// Create a NodeNetworkConfig for testNode since it would have already existed on the apiserver
	testNodeNetworkConfig := &acnv1alpha.NodeNetworkConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testNode.ObjectMeta.Name,
			Namespace: testNamespace,
		},
	}

	mockClient := &mockNodeReconcilerClient{
		nodes: map[string]*corev1.Node{
			fmt.Sprintf("/%s", testNode.ObjectMeta.Name): testNode,
		},
		nodeNetworkConfigs: map[string]*acnv1alpha.NodeNetworkConfig{
			testNodeNetworkConfig.ObjectMeta.Name: testNodeNetworkConfig,
		},
	}

	nodeReconciler := NodeReconciler{
		Client:         mockClient,
		Namespace:      testNamespace,
		InitialIPCount: 8,
	}

	_, err := nodeReconciler.Reconcile(reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "",
			Name:      "test-node",
		},
	})
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Reconcile should have deleted the NodeNetworkConfig with same name as testNode
	want := false
	_, got := mockClient.nodeNetworkConfigs[testNode.ObjectMeta.Name]

	if want != got {
		t.Errorf("Reconcile should have deleted the NodeNetworkConfig with same name as testNode")
	}

	// Reconcile should have removed the finalizer from testNode
	want = false
	got = containsString(testNode.Finalizers, NodeFinalizerName)

	if want != got {
		t.Errorf("Reconcile should have removed the finalizer %s from testNode", NodeFinalizerName)
	}
}

// TestReconcilePartiallyDeletedNode tests that Reconcile will remove the finalizer from a Node with a non-zero deletion timestamp where the NodeNetworkConfig has already been deleted.
// This can occur if RequestController crashes while processing a deleting node
func TestReconcilePartiallyDeletedNode(t *testing.T) {
	// Create a Node with non-zero deletion timestamp to simulate Node deletion
	testNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-node",
			DeletionTimestamp: &metav1.Time{time.Now()},
			Finalizers:        []string{NodeFinalizerName},
		},
	}

	testNamespace := "test"

	// NodeNetworkConfig has already been removed, so don't create one here
	mockClient := &mockNodeReconcilerClient{
		nodes: map[string]*corev1.Node{
			fmt.Sprintf("/%s", testNode.ObjectMeta.Name): testNode,
		},
		nodeNetworkConfigs: make(map[string]*acnv1alpha.NodeNetworkConfig),
	}

	nodeReconciler := NodeReconciler{
		Client:         mockClient,
		Namespace:      testNamespace,
		InitialIPCount: 8,
	}

	_, err := nodeReconciler.Reconcile(reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "",
			Name:      "test-node",
		},
	})
	if err != nil {
		t.Fatalf("%v", err)
	}

	// Reconcile should have removed the finalizer from testNode
	want := false
	got := containsString(testNode.Finalizers, NodeFinalizerName)

	if want != got {
		t.Errorf("Reconcile should have removed the finalizer %s from testNode", NodeFinalizerName)
	}
}
