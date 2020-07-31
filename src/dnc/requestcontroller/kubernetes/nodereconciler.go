// Copyright 2020 Microsoft. All rights reserved.

package kubernetes

import (
	"context"
	"log"
	"time"

	acnv1alpha "github.com/Azure/azure-container-networking/nodenetworkconfig/api/v1alpha"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// NodeReconciler watches Nodes and creates/deletes NetworkConfigs
type NodeReconciler struct {
	//Log                             log.Logger
	Client           K8sClient
	Namespace        string
	InitialIPCount   int64
	K8sClientTimeout time.Duration
}

var _ reconcile.Reconciler = &NodeReconciler{}

// Reconcile ensures a NetworkConfig object exists for each Node, and deletes them on Node deletion
func (n *NodeReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Printf("Reconciling: %s", request.NamespacedName)

	// Get the node from the cache
	node := corev1.Node{}
	nodeFound := true
	parentCtx := context.Background()
	ctx, cancel := context.WithTimeout(parentCtx, n.K8sClientTimeout)
	defer cancel()
	err := n.Client.Get(ctx, request.NamespacedName, &node)
	if err != nil {
		// NotFound error will be treated as deletion event
		if apierrors.IsNotFound(err) {
			nodeFound = false
		} else {
			log.Printf("%v", err)
			return reconcile.Result{}, err
		}

	}

	nodeIsDeleting := !nodeFound || !node.DeletionTimestamp.IsZero()

	if nodeIsDeleting {
		// Node is being deleted so delete NodeNetworkConfig
		ctx, cancel := context.WithTimeout(parentCtx, n.K8sClientTimeout)
		defer cancel()
		if err := n.deleteNodeNetworkConfig(ctx, request.Name); err != nil {
			// return error so we can retry deleting NodeNetworkConfig
			log.Printf("%v", err)
			return reconcile.Result{}, err
		}
	} else {
		// Node isn't being deleted, create the NodeNetworkConfig object for node
		ctx, cancel := context.WithTimeout(parentCtx, n.K8sClientTimeout)
		defer cancel()
		if err := n.createNodeNetworkConfig(ctx, request.Name); err != nil {
			// return error to requeue and try creating NetworkConfig again
			log.Printf("%v", err)
			return reconcile.Result{}, err
		}
	}

	// Encountered no errors, so reconcile can exit gracefully
	return reconcile.Result{}, nil
}

// This function is idempotent. If a NodeNetworkConfig already exists with this name, this will do nothing, and will return nil.
func (n *NodeReconciler) createNodeNetworkConfig(ctx context.Context, name string) error {
	nodeNetworkConfig := acnv1alpha.NodeNetworkConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: n.Namespace,
		},
		Spec: acnv1alpha.NodeNetworkConfigSpec{
			RequestedIPCount: n.InitialIPCount,
		},
	}

	log.Printf("Creating NodeNetworkConfig %s in namespace %s with IP Count %d",
		nodeNetworkConfig.ObjectMeta.Name, nodeNetworkConfig.ObjectMeta.Namespace,
		nodeNetworkConfig.Spec.RequestedIPCount)

	err := n.Client.Create(ctx, &nodeNetworkConfig)
	if err != nil {
		// return the error unless it is an AlreadyExists error
		if !apierrors.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}

func (n *NodeReconciler) deleteNodeNetworkConfig(ctx context.Context, name string) error {
	nodeNetworkConfig := acnv1alpha.NodeNetworkConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: n.Namespace,
		},
	}

	log.Printf("Deleting NodeNetworkConfig %s in namespace %s",
		nodeNetworkConfig.ObjectMeta.Name, nodeNetworkConfig.ObjectMeta.Namespace)

	err := n.Client.Delete(ctx, &nodeNetworkConfig)
	if err != nil {
		// return the error unless it is a NotFound error
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
