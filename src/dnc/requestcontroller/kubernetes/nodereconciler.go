// Copyright 2020 Microsoft. All rights reserved.

package kubernetes

import (
	"context"
	"dnc/api"
	"dnc/requestcontroller"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	acnv1alpha "github.com/Azure/azure-container-networking/nodenetworkconfig/api/v1alpha"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// NodeReconciler watches Nodes and creates/deletes NetworkConfigs
type NodeReconciler struct {
	k8sClient        K8sClient
	dncClient        requestcontroller.SynchronousDNCClient
	namespace        string
	initialIPCount   int64
	k8sClientTimeout time.Duration
	logger           logrus.FieldLogger
}

type NodeReconcilerOptions struct {
	K8sClient        K8sClient
	DNCClient        requestcontroller.SynchronousDNCClient
	Namespace        string
	InitialIPCount   int64
	K8sClientTimeout time.Duration
	Logger           logrus.FieldLogger
}

var _ reconcile.Reconciler = &NodeReconciler{}

func NewNodeReconciler(opts NodeReconcilerOptions) *NodeReconciler {
	logger := opts.Logger.WithFields(logrus.Fields{
		"component": "NodeReconciler",
	})

	return &NodeReconciler{
		k8sClient:        opts.K8sClient,
		dncClient:        opts.DNCClient,
		namespace:        opts.Namespace,
		initialIPCount:   opts.InitialIPCount,
		k8sClientTimeout: opts.K8sClientTimeout,
		logger:           logger,
	}
}

// Reconcile ensures a NetworkConfig object exists for each Node, and deletes them on Node deletion
func (n *NodeReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	name := request.NamespacedName.String()

	logger := n.logger.WithFields(logrus.Fields{
		"object": name,
	})

	logger.Infof("Reconciling: %s", request.NamespacedName)

	// Get the node from the cache
	node := corev1.Node{}
	nodeFound := true
	parentCtx := context.Background()
	ctx, cancel := context.WithTimeout(parentCtx, n.k8sClientTimeout)
	defer cancel()
	err := n.k8sClient.Get(ctx, request.NamespacedName, &node)
	if err != nil {
		// NotFound error will be treated as deletion event
		if apierrors.IsNotFound(err) {
			nodeFound = false
		} else {
			logger.WithError(err).Infof("Error getting Node: %v", err)
			return reconcile.Result{}, err
		}

	}

	nodeIsDeleting := !nodeFound || !node.DeletionTimestamp.IsZero()

	if nodeIsDeleting {
		// Node is being deleted so delete NodeNetworkConfig
		ctx, cancel := context.WithTimeout(parentCtx, n.k8sClientTimeout)
		defer cancel()
		if err := n.deleteNodeNetworkConfig(ctx, request.Name); err != nil {
			// return error so we can retry deleting NodeNetworkConfig
			logger.WithError(err).Infof("Error deleting NNC: %v", err)
			return reconcile.Result{}, err
		}
	} else {
		// Node isn't being deleted, create the NodeNetworkConfig object for node
		ctx, cancel := context.WithTimeout(parentCtx, n.k8sClientTimeout)
		defer cancel()

		// Check that the delegatedSubnetID label and subnetName label are present
		_, delegatedSubnetLabelFound := node.Labels[DelegatedSubnetIDLabel]
		_, subnetNameLabelFound := node.Labels[SubnetNameLabel]

		if !delegatedSubnetLabelFound {
			logger.Infof("Label %s not found for node %s", DelegatedSubnetIDLabel, node.Name)
		}

		if !subnetNameLabelFound {
			logger.Infof("Label %s not found for node %s", SubnetNameLabel, node.Name)
		}

		if !delegatedSubnetLabelFound || !subnetNameLabelFound {
			// a label is missing, so we can skip this node if it doesn't exist on DNC
			_, err := n.dncClient.GetNode(ctx, node.Name)
			var jsonErr *api.JsonErr
			if err != nil && errors.As(err, &jsonErr) {
				if jsonErr.HTTPStatus == http.StatusNotFound {
					logger.Infof("Skipping NNC creation for node %s, it was not found on DNC and is missing label(s)", node.Name)
					return reconcile.Result{}, nil
				}
			}
			msg := fmt.Sprintf("Label(s) not found for Node %s but the node was found on DNC", node.Name)
			logger.Info(msg)
			return reconcile.Result{}, errors.New(msg)
		}

		if err := n.createNodeNetworkConfig(ctx, request.Name); err != nil {
			// return error to requeue and try creating NetworkConfig again
			logger.WithError(err).Infof("Error creating NNC: %v", err)
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
			Namespace: n.namespace,
		},
		Spec: acnv1alpha.NodeNetworkConfigSpec{
			RequestedIPCount: n.initialIPCount,
		},
	}

	n.logger.WithFields(logrus.Fields{
		"object":  fmt.Sprintf("%s/%s", n.namespace, name),
		"ipCount": strconv.FormatInt(nodeNetworkConfig.Spec.RequestedIPCount, 10),
	}).Infof("Creating NodeNetworkConfig %s in namespace %s with IP Count %d",
		nodeNetworkConfig.ObjectMeta.Name, nodeNetworkConfig.ObjectMeta.Namespace,
		nodeNetworkConfig.Spec.RequestedIPCount)

	err := n.k8sClient.Create(ctx, &nodeNetworkConfig)
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
			Namespace: n.namespace,
		},
	}

	n.logger.WithFields(logrus.Fields{
		"object": fmt.Sprintf("%s/%s", n.namespace, name),
	}).Infof("Deleting NodeNetworkConfig %s in namespace %s",
		nodeNetworkConfig.ObjectMeta.Name, nodeNetworkConfig.ObjectMeta.Namespace)

	err := n.k8sClient.Delete(ctx, &nodeNetworkConfig)
	if err != nil {
		// return the error unless it is a NotFound error
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
