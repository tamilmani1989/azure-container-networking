package kubernetes

import (
	"context"
	"log"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// NodeReconciler watches Nodes, adds/removes a finalizer, and creates/deletes NetworkConfigs
type NodeReconciler struct {
	//Log                             log.Logger
	Client                          client.Client
	OverrideCreateNetworkConfigFunc func(string) error
	OverrideDeleteNetworkConfigFunc func(string) error
}

var _ reconcile.Reconciler = &NodeReconciler{}

// Reconcile ensures a NetworkConfig object exists for each Node, and deletes them on Node deletion
func (n *NodeReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	needToUpdateNode := false // keep track of if we need to call Update func

	// Get the node from the cache
	node := corev1.Node{}
	err := n.Client.Get(context.TODO(), request.NamespacedName, &node)
	if err != nil {
		// ignore not-found error since an immediate requeue won't solve the problem
		log.Printf("%v", err)
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	nodeIsDeleting := !node.DeletionTimestamp.IsZero()
	// Using finalizer so we can cleanup NetworkConfigs on deletion of Node
	if nodeIsDeleting {
		// Node is being deleted. Delete NetworkConfig then remove finalizer
		if containsString(node.Finalizers, NodeFinalizerName) {
			if err := n.deleteNetworkConfig(node.Name); err != nil {
				// return error so we can retry deleting NetworkConfig
				log.Printf("%v", err)
				return reconcile.Result{}, err
			}
			// remove the finalizer
			node.Finalizers = removeStringFromSlice(node.Finalizers, NodeFinalizerName)
			needToUpdateNode = true
		}
	} else {
		// Node isn't being deleted. Add finalizer if not present
		if !containsString(node.Finalizers, NodeFinalizerName) {
			node.Finalizers = append(node.Finalizers, NodeFinalizerName)
			needToUpdateNode = true
		}
		// Create the NetworkConfig object for node
		// this is idempotent, so it can always be called
		if err := n.createNetworkConfig(node.Name); err != nil {
			// return error to requeue and try creating NetworkConfig again
			log.Printf("%v", err)
			return reconcile.Result{}, err
		}
	}

	if needToUpdateNode {
		if err := n.Client.Update(context.TODO(), &node); err != nil {
			log.Printf("%v", err)
			// requeue to try again since the changes didn't persist
			return reconcile.Result{}, err
		}
	}

	// Encountered no errors, so reconcile can exit gracefully
	return reconcile.Result{}, nil
}

func (n *NodeReconciler) createNetworkConfig(name string) error {
	if n.OverrideCreateNetworkConfigFunc != nil {
		return n.OverrideCreateNetworkConfigFunc(name)
	}

	//TODO implement default createNetworkConfig behavior
	return nil
}

func (n *NodeReconciler) deleteNetworkConfig(name string) error {
	if n.OverrideDeleteNetworkConfigFunc != nil {
		return n.OverrideDeleteNetworkConfigFunc(name)
	}

	//TODO implement default deleteNetworkConfig behavior
	return nil
}
