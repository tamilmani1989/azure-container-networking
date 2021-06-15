// Copyright 2020 Microsoft. All rights reserved.

package kubernetes

import (
	"context"
	"dnc/requestcontroller"
	syncdncclient "dnc/requestcontroller/dncclient"
	"fmt"
	"strconv"
	"sync"
	"testing"

	"github.com/Azure/azure-container-networking/cns"
	acnv1alpha "github.com/Azure/azure-container-networking/nodenetworkconfig/api/v1alpha"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

//TODO: don't use a global here
var logger logrus.FieldLogger = logrus.New()

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
	k8sNodeName := "test-node"
	namespace := "test-namespace"
	requestedIPCount := 16
	delegatedSubnetID := "unique-delegated-subnet-id"
	vnetGUID := "93ca9913-dc26-40fe-9e16-9164669590a0"
	subnetName := "test-subnet"
	subnetToken := "bd48a062-efa7-4c50-af4b-9c97add5b8cc"
	orchestratorID := "a57a005f-f169-41b1-897a-ec47a6953279"

	k8sClient := newMockK8sClient()
	dncClient := syncdncclient.MockSynchronousDNCClient{}

	nncReconciler := NewNodeNetworkConfigReconciler(NodeNetworkConfigReconcilerOptions{
		K8sClient: k8sClient,
		DNCClient: &dncClient,
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
		Logger:         logger,
		Recorder:       &record.FakeRecorder{},
		OrchestratorID: orchestratorID,
		NodeSet:        orchestratorID,
	})

	nnc := acnv1alpha.NodeNetworkConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sNodeName,
			Namespace: namespace,
		},
		Spec: acnv1alpha.NodeNetworkConfigSpec{
			RequestedIPCount: int64(requestedIPCount),
		},
	}
	node := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: k8sNodeName,
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
		t.Fatalf("Failed to create NNC %s on apiserver with error %v", k8sNodeName, err)
	}
	err = k8sClient.Create(context.Background(), &node)
	if err != nil {
		t.Fatalf("Failed to create node %s on apiserver with error %v", k8sNodeName, err)
	}

	_, err = nncReconciler.Reconcile(reconcile.Request{types.NamespacedName{Namespace: namespace, Name: k8sNodeName}})

	if err != nil {
		t.Fatalf("Failed to reconcile NNC %s with error %v", k8sNodeName, err)
	}

	dncNodeName, err := nncReconciler.getDNCNodeNameFromNNCName(context.Background(), k8sNodeName)
	if err != nil {
		t.Fatalf("encountered error when getting dnc node name from k8s node name: %+v", err)
	}

	nodeIDs, err := dncClient.GetNodes(context.Background(), orchestratorID)
	if err != nil {
		t.Fatalf("error getting all nodes: %v", err)
	}

	if len(nodeIDs.Nodes) != 1 || nodeIDs.Nodes[0].Name != dncNodeName {
		t.Fatalf("expected 1 node named %s, got %+v", dncNodeName, nodeIDs)
	}

	nodeInfoResp, err := dncClient.GetNode(context.Background(), dncNodeName)
	if err != nil {
		t.Fatalf("Expected to receive node from DNC but received error %+v", err)
	}

	if len(nodeInfoResp.NetworkContainers) != 1 {
		t.Fatalf("Expected 1 NC for node %s but received %d", k8sNodeName, len(nodeInfoResp.NetworkContainers))
	}

	ipCount := len(nodeInfoResp.NetworkContainers[0].IPConfiguration.SecondaryIPs)
	if ipCount != requestedIPCount {
		t.Fatalf("Expected %d IPs for NC but received %d", requestedIPCount, ipCount)
	}

}

// TestReconcileNNCWithLegacyNodeName tests that RC will use the legacy node name format (no orchestrator ID) if the node already exists on DNC
func TestReconcileNNCWithLegacyNodeName(t *testing.T) {
	k8sNodeName := "test-node"
	namespace := "test-namespace"
	requestedIPCount := 16
	delegatedSubnetID := "unique-delegated-subnet-id"
	vnetGUID := "93ca9913-dc26-40fe-9e16-9164669590a0"
	subnetName := "test-subnet"
	subnetToken := "bd48a062-efa7-4c50-af4b-9c97add5b8cc"
	orchestratorID := "c61a5610-8eb8-4e00-aacb-40a3e7b1379a"
	nodeIP := "10.1.0.1"

	k8sClient := newMockK8sClient()
	dncClient := syncdncclient.MockSynchronousDNCClient{}

	// Put the node on DNC with the legacy node name format, <nodeID>
	if err := dncClient.UpsertNode(context.Background(), k8sNodeName, nodeIP, cns.KubernetesCRD, vnetGUID, ""); err != nil {
		t.Fatalf("could not add node to dnc: %+v", err)
	}

	nncReconciler := NewNodeNetworkConfigReconciler(NodeNetworkConfigReconcilerOptions{
		K8sClient: k8sClient,
		DNCClient: &dncClient,
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
		Logger:         logger,
		Recorder:       &record.FakeRecorder{},
		OrchestratorID: orchestratorID,
	})

	nnc := acnv1alpha.NodeNetworkConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sNodeName,
			Namespace: namespace,
		},
		Spec: acnv1alpha.NodeNetworkConfigSpec{
			RequestedIPCount: int64(requestedIPCount),
		},
	}
	node := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: k8sNodeName,
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
		t.Fatalf("Failed to create NNC %s on apiserver with error %v", k8sNodeName, err)
	}
	err = k8sClient.Create(context.Background(), &node)
	if err != nil {
		t.Fatalf("Failed to create node %s on apiserver with error %v", k8sNodeName, err)
	}

	_, err = nncReconciler.Reconcile(reconcile.Request{types.NamespacedName{Namespace: namespace, Name: k8sNodeName}})

	if err != nil {
		t.Fatalf("Failed to reconcile NNC %s with error %v", k8sNodeName, err)
	}

	dncNodeName, err := nncReconciler.getDNCNodeNameFromNNCName(context.Background(), k8sNodeName)
	if err != nil {
		t.Fatalf("encountered error when getting dnc node name from k8s node name: %+v", err)
	}

	if dncNodeName != k8sNodeName {
		// expected that the node name is the legacy <k8s node name>
		t.Fatalf("expected %s, got %s", k8sNodeName, dncNodeName)
	}

	nodeInfoResp, err := dncClient.GetNode(context.Background(), dncNodeName)
	if err != nil {
		t.Fatalf("Expected to receive node from DNC but received error %+v", err)
	}

	if len(nodeInfoResp.NetworkContainers) != 1 {
		t.Fatalf("Expected 1 NC for node %s but received %d", k8sNodeName, len(nodeInfoResp.NetworkContainers))
	}

	ipCount := len(nodeInfoResp.NetworkContainers[0].IPConfiguration.SecondaryIPs)
	if ipCount != requestedIPCount {
		t.Fatalf("Expected %d IPs for NC but received %d", requestedIPCount, ipCount)
	}
}

func TestGetMaxIPCountOfNodeOverLimit(t *testing.T) {
	initialMaxPodIPCount := swiftIPCountSystemMax + 1
	expectedMaxPodIPCount := swiftIPCountSystemMax
	quantity, _ := resource.ParseQuantity(strconv.Itoa(initialMaxPodIPCount))
	nodeName := "test-node"

	// Instantiate a node with one more than the allowable maxIPCount
	node := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
		},
		Status: corev1.NodeStatus{
			Capacity: corev1.ResourceList{
				corev1.ResourcePods: quantity,
			},
		},
	}

	nncReconciler := NewNodeNetworkConfigReconciler(NodeNetworkConfigReconcilerOptions{
		Logger:   logger,
		Recorder: &record.FakeRecorder{},
	})
	givenMaxIPCount := nncReconciler.getMaxIPCountOfNode(node)

	if givenMaxIPCount != int64(expectedMaxPodIPCount) {
		t.Fatalf("expected maxIpCount to be %d, got %d", expectedMaxPodIPCount, givenMaxIPCount)
	}
}

func TestGetMaxIPCountOfNodeUnderMin(t *testing.T) {
	initialMaxPodIPCount := 0
	expectedMaxPodIPCount := swiftIPCountSystemMax
	quantity, _ := resource.ParseQuantity(strconv.Itoa(initialMaxPodIPCount))
	nodeName := "test-node"

	// Instantiate a node with one less than the allowable minimum value for maxIPCount
	node := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
		},
		Status: corev1.NodeStatus{
			Capacity: corev1.ResourceList{
				corev1.ResourcePods: quantity,
			},
		},
	}

	nncReconciler := NewNodeNetworkConfigReconciler(NodeNetworkConfigReconcilerOptions{
		Logger:   logger,
		Recorder: &record.FakeRecorder{},
	})
	givenMaxIPCount := nncReconciler.getMaxIPCountOfNode(node)

	if givenMaxIPCount != int64(expectedMaxPodIPCount) {
		t.Fatalf("expected maxIpCount to be %d, got %d", expectedMaxPodIPCount, givenMaxIPCount)
	}
}

func TestGetMaxIPCountOfNodeWithNoCapacity(t *testing.T) {
	expectedMaxPodIPCount := swiftIPCountSystemMax
	nodeName := "test-node"

	node := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
		},
	}

	nncReconciler := NewNodeNetworkConfigReconciler(NodeNetworkConfigReconcilerOptions{
		Logger:   logger,
		Recorder: &record.FakeRecorder{},
	})
	givenMaxIPCount := nncReconciler.getMaxIPCountOfNode(node)

	if givenMaxIPCount != int64(expectedMaxPodIPCount) {
		t.Fatalf("expected maxIpCount to be %d, got %d", expectedMaxPodIPCount, givenMaxIPCount)
	}
}

func TestGetMaxIPCountOfNodeWithLabel(t *testing.T) {
	nodeName := "test-node"
	maxPodsLabelVal := 60

	node := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
			Labels: map[string]string{
				MaxVnetPodsLabel: fmt.Sprintf("%d", maxPodsLabelVal),
			},
		},
	}

	nncReconciler := NewNodeNetworkConfigReconciler(NodeNetworkConfigReconcilerOptions{
		Logger:   logger,
		Recorder: &record.FakeRecorder{},
	})
	givenMaxIPCount := nncReconciler.getMaxIPCountOfNode(node)

	if givenMaxIPCount != int64(maxPodsLabelVal) {
		t.Fatalf("expected maxIpCount to be %d, got %d", maxPodsLabelVal, givenMaxIPCount)
	}
}

func TestGetMaxIPCountOfNodeWithLabelAndCapacity(t *testing.T) {
	nodeName := "test-node"
	maxPodsLabelVal := 60
	nodeCapacity := 30
	nodeCapacityQ, _ := resource.ParseQuantity(strconv.Itoa(nodeCapacity))

	node := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
			Labels: map[string]string{
				MaxVnetPodsLabel: fmt.Sprintf("%d", maxPodsLabelVal),
			},
		},
		Status: corev1.NodeStatus{
			Capacity: corev1.ResourceList{
				corev1.ResourcePods: nodeCapacityQ,
			},
		},
	}

	nncReconciler := NewNodeNetworkConfigReconciler(NodeNetworkConfigReconcilerOptions{
		Logger:   logger,
		Recorder: &record.FakeRecorder{},
	})
	givenMaxIPCount := nncReconciler.getMaxIPCountOfNode(node)

	if givenMaxIPCount != int64(nodeCapacity) {
		t.Fatalf("expected maxIpCount to be %d, got %d", nodeCapacity, givenMaxIPCount)
	}
}

func TestNodeExistsOnDNC(t *testing.T) {
	infraVnet := "9e445b0d-961a-4378-b239-d92815b87334"
	dncClient := &syncdncclient.MockSynchronousDNCClient{}
	if err := dncClient.UpsertNode(context.Background(), "node1", "127.0.0.1", cns.KubernetesCRD, infraVnet, ""); err != nil {
		t.Fatalf("could not add node to dnc: %+v", err)
	}

	nncReconciler := NewNodeNetworkConfigReconciler(NodeNetworkConfigReconcilerOptions{
		Logger:    logger,
		Recorder:  &record.FakeRecorder{},
		DNCClient: dncClient,
	})

	exists, err := nncReconciler.nodeExistsOnDNC(context.Background(), "node1")
	if err != nil {
		t.Fatalf("unexpected error while checking if node exists on dnc: %+v", err)
	}

	if !exists {
		t.Fatal("expected exists to be true, got false")
	}

	exists, err = nncReconciler.nodeExistsOnDNC(context.Background(), "node2")
	if err != nil {
		t.Fatalf("unexpected error while checking if node exists on dnc: %+v", err)
	}

	if exists {
		t.Fatal("expected exists to be false, got true")
	}
}

func TestGetBatchSizeWhenOverMaxIPCount(t *testing.T) {
	maxIPCount := 30
	expectedBatchSize := maxIPCount
	initialBatchSize := 50
	nodeName := "test-node"
	quantity, _ := resource.ParseQuantity(strconv.Itoa(maxIPCount))

	node := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
		},
		Status: corev1.NodeStatus{
			Capacity: corev1.ResourceList{
				corev1.ResourcePods: quantity,
			},
		},
	}

	nncReconciler := NewNodeNetworkConfigReconciler(NodeNetworkConfigReconcilerOptions{
		ScalerConfig: requestcontroller.RequestScalerConfig{
			BatchSize: int64(initialBatchSize),
		},
		Logger:   logger,
		Recorder: &record.FakeRecorder{},
	})

	outputBatchSize := nncReconciler.getBatchSize(node)

	if outputBatchSize != int64(expectedBatchSize) {
		t.Fatalf("Expected batch size to be maxIPCount (%v) when batch size in CRD is over maxIPCount but got %v", expectedBatchSize, outputBatchSize)
	}
}
