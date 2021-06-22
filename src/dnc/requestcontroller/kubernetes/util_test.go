package kubernetes

import (
	"fmt"
	"strconv"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

	givenMaxIPCount := getMaxIPCountOfNode(node)

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

	givenMaxIPCount := getMaxIPCountOfNode(node)

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

	givenMaxIPCount := getMaxIPCountOfNode(node)

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

	givenMaxIPCount := getMaxIPCountOfNode(node)

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

	givenMaxIPCount := getMaxIPCountOfNode(node)

	if givenMaxIPCount != int64(nodeCapacity) {
		t.Fatalf("expected maxIpCount to be %d, got %d", nodeCapacity, givenMaxIPCount)
	}
}
