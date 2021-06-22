// Copyright 2020 Microsoft. All rights reserved.

package kubernetes

import (
	"strconv"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func removeStringFromSlice(strings []string, remove string) ([]string, bool) {
	ret := []string{}
	deletedSomething := false
	for _, s := range strings {
		if s != remove {
			ret = append(ret, s)
		} else {
			deletedSomething = true
		}
	}
	return ret, deletedSomething
}

func appendStringIfNotPresent(strings []string, s string) []string {
	for _, t := range strings {
		if t == s {
			return strings
		}
	}

	return append(strings, s)
}

// createEventNodeRef creates an ObjectReference to a Node to be used as the
// target in an Event.
// This is required because the Node Event references must have the UID of the
// ObjectRef set to the name of the Node, *NOT* the UID of the Node Object, so
// that the Events show up in the Event stream of the Node correctly.
// (like when executing a `kubectl describe <node>`)
// Letting the EventRecorder machinery build the ObjectRef automatically
// sets the Node Qbject's actual UID which does not correctly associate the
// Event to the Node.
func createEventNodeRef(nodeName string) *v1.ObjectReference {
	return &v1.ObjectReference{
		Kind:      "Node",
		Name:      string(nodeName),
		UID:       types.UID(nodeName), // not a UID, but that's how it works
		Namespace: "",
	}
}

// getMaxIPCountOfNode gets the max ip count for a node
// returns default max ip count if label is either not found or 0
func getMaxIPCountOfNode(node v1.Node) int64 {
	// Get node capacity from node object
	// Returns 0 if label is not present
	nodeCapacity := node.Status.Capacity.Pods().Value()
	if nodeCapacity <= 0 {
		// if node capacity not present or < 0, set it to system max
		nodeCapacity = swiftIPCountSystemMax
	}

	// Get max desired swift pods from label
	// Start with system max, overwrite if label is found and parsed successfully
	var maxDesiredPods int64 = swiftIPCountSystemMax
	maxDesiredPodsString := node.Labels[MaxVnetPodsLabel]
	if parsedNum, err := strconv.ParseInt(maxDesiredPodsString, 10, 0); err == nil {
		// don't allow value <= 0
		if parsedNum > 0 {
			maxDesiredPods = parsedNum
		}
	}

	// Take the min of nodeCapacity, maxDesiredPods, and systemMax
	return minInt64(minInt64(nodeCapacity, maxDesiredPods), swiftIPCountSystemMax)
}

func minInt64(a, b int64) int64 {
	if a <= b {
		return a
	}

	return b
}
