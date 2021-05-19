// Copyright 2020 Microsoft. All rights reserved.

package kubernetes

import (
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

func containsString(strings []string, s string) bool {
	for _, t := range strings {
		if t == s {
			return true
		}
	}

	return false
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
