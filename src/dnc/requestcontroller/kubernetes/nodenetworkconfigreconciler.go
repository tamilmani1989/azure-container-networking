package kubernetes

import (
	"context"
	"log"

	acnv1alpha "github.com/Azure/azure-container-networking/nodenetworkconfig/api/v1alpha"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type NodeNetworkConfigReconciler struct {
	Client K8sClient
}

var _ reconcile.Reconciler = &NodeNetworkConfigReconciler{}

func (n *NodeNetworkConfigReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Printf("Reconciling %s", request.NamespacedName)
	// Get the NodeNetworkConfig from the cache
	nodeNetworkConfig := acnv1alpha.NodeNetworkConfig{}

	err := n.Client.Get(context.TODO(), request.NamespacedName, &nodeNetworkConfig)
	if err != nil {
		// ignore not found since an immediate requeue won't solve problem
		log.Printf("%v", err)
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	return reconcile.Result{}, nil
}
