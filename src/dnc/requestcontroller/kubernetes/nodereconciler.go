package kubernetes

import (
	"context"
	"log"
	"time"

	acnv1alpha "github.com/Azure/azure-container-networking/nodenetworkconfig/api/v1alpha"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// NodeReconciler watches Nodes, adds/removes a finalizer, and creates/deletes NetworkConfigs
type NodeReconciler struct {
	//Log                             log.Logger
	Client                          K8sClient
	OverrideCreateNetworkConfigFunc func(context.Context, string) error
	OverrideDeleteNetworkConfigFunc func(context.Context, string) error
	Namespace                       string
	InitialIPCount                  int64
	K8sClientTimeout                time.Duration
}

var _ reconcile.Reconciler = &NodeReconciler{}

// Reconcile ensures a NetworkConfig object exists for each Node, and deletes them on Node deletion
func (n *NodeReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Printf("Reconciling: %s", request.NamespacedName)
	needToUpdateNode := false // keep track of if we need to call Update func

	// Get the node from the cache
	node := corev1.Node{}
	parentCtx := context.Background()
	ctx, cancel := context.WithTimeout(parentCtx, n.K8sClientTimeout)
	defer cancel()
	err := n.Client.Get(ctx, request.NamespacedName, &node)
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
			ctx, cancel := context.WithTimeout(parentCtx, n.K8sClientTimeout)
			defer cancel()
			if err := n.deleteNetworkConfig(ctx, node.Name); err != nil {
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
		ctx, cancel := context.WithTimeout(parentCtx, n.K8sClientTimeout)
		defer cancel()
		if err := n.createNetworkConfig(ctx, node.Name); err != nil {
			// return error to requeue and try creating NetworkConfig again
			log.Printf("%v", err)
			return reconcile.Result{}, err
		}
	}

	if needToUpdateNode {
		ctx, cancel := context.WithTimeout(parentCtx, n.K8sClientTimeout)
		defer cancel()
		if err := n.Client.Update(ctx, &node); err != nil {
			log.Printf("%v", err)
			// requeue to try again since the changes didn't persist
			return reconcile.Result{}, err
		}
	}

	// Encountered no errors, so reconcile can exit gracefully
	return reconcile.Result{}, nil
}

// This function is idempotent. If a NodeNetworkConfig already exists with this name, this will do nothing, and will return nil.
func (n *NodeReconciler) createNetworkConfig(ctx context.Context, name string) error {
	if n.OverrideCreateNetworkConfigFunc != nil {
		return n.OverrideCreateNetworkConfigFunc(ctx, name)
	}

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

func (n *NodeReconciler) deleteNetworkConfig(ctx context.Context, name string) error {
	if n.OverrideDeleteNetworkConfigFunc != nil {
		return n.OverrideDeleteNetworkConfigFunc(ctx, name)
	}

	objMeta := metav1.ObjectMeta{
		Name:      name,
		Namespace: n.Namespace,
	}

	nodeNetworkConfig := acnv1alpha.NodeNetworkConfig{
		ObjectMeta: objMeta,
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
