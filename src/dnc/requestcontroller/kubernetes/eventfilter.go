// Copyright 2020 Microsoft. All rights reserved.

package kubernetes

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

//NodeEventFilter implements a predicate to ignore Update events
type NodeEventFilter struct {
	predicate.Funcs // Defaults to allow all events
}

//Update implements a predicate update event. Returns true when Node is being deleted
func (NodeEventFilter) Update(e event.UpdateEvent) bool {
	nodeIsDeleting := !e.MetaNew.GetDeletionTimestamp().IsZero()
	return nodeIsDeleting
}

//NodeNetworkConfigEventFilter implements a predicate to ignore Status updates
type NodeNetworkConfigEventFilter struct {
	predicate.Funcs
}

//Update implements a predicate update event. Returns false if this is a status update, true otherwise
func (NodeNetworkConfigEventFilter) Update(e event.UpdateEvent) bool {
	return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
}
