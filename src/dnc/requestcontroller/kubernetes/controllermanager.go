// Copyright 2020 Microsoft. All rights reserved.

package kubernetes

import (
	"context"
	"dnc/requestcontroller"
	"dnc/retry"
	"fmt"
	"strconv"
	"time"

	acnv1alpha "github.com/Azure/azure-container-networking/nodenetworkconfig/api/v1alpha"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlmanager "sigs.k8s.io/controller-runtime/pkg/manager"
)

// TODO: Kubebuilder uses a global var scheme and init func. Investigate if this is necessary or can be done without global
var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = acnv1alpha.AddToScheme(scheme)
}

// NodeFinalizerName is the finalizer that is placed on Nodes
const (
	NNCFinalizerName              = "finalizers.acn.azure.com/dnc-operations"
	InfraVnetIdLabel              = "kubernetes.azure.com/nodenetwork-vnetguid"
	DelegatedSubnetIDLabel        = "kubernetes.azure.com/podnetwork-delegationguid"
	SubnetNameLabel               = "kubernetes.azure.com/podnetwork-subnet"
	MaxVnetPodsLabel              = "kubernetes.azure.com/podnetwork-max-pods"
	leaderElectionConfigmapSuffix = ".resourcelock.acn.azure.com"
	nncPluralName                 = "nodenetworkconfigs"
	nncShortName                  = "nnc"
	swiftIPCountSystemMax         = 250
)

// K8sClient is an interface that wraps only the functions of controller-runtime/pkg/client.Client that the reconcilers use
// Unit tests can implement this interface to mock a K8s client
type K8sClient interface {
	Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error
	Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error
	Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error
	Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error
	Status() client.StatusWriter
}

// controllerManager is an implementation of requestcontroller.ControllerManager
type controllerManager struct {
	ctrlRuntimeMgr             ctrlmanager.Manager
	crdClient                  K8sCRDClient
	syncDNCClient              requestcontroller.SynchronousDNCClient
	logger                     logrus.FieldLogger
	nncNamespace               string
	initialIPCount             int64
	enableStartupNodeReconcile bool
	nodeSet                    string
	metricsClient              requestcontroller.MetricsClient
}

// Start starts the controller and blocks until the exitchan is closed, or an unrecoverable error occurs
func (c *controllerManager) Start(ctx context.Context) error {
	// Ensure CRD definition is installed on cluster before starting up manager
	if err := ensureCRDIsRegistered(ctx, c.crdClient, c.logger); err != nil {
		return err
	}

	// Start up the controllerManager
	ctrlMgrExitChan := make(chan struct{})
	ctrlMgrErrChan := make(chan error)

	// c.controller is stopped when we close mgrExitChan
	go func(mgrExitChan <-chan struct{}, mgrErrChan chan error) {
		if err := c.ctrlRuntimeMgr.Start(mgrExitChan); err != nil {
			mgrErrChan <- err
		}
	}(ctrlMgrExitChan, ctrlMgrErrChan)

	// Reconcile the nodes from dnc list-nodes; client will become valid after controller is started

	if c.enableStartupNodeReconcile {
		rehydrateCtx, rehydrateCancel := context.WithCancel(ctx)
		defer rehydrateCancel()
		go func(ctx context.Context) {
			if err := c.reconcileDNCNodes(ctx); err != nil {
				// TODO: Should this be a fatal error and passed back to caller?
				// for now, just log the error instead of exiting the clusterManager
				c.logger.Errorf("Error rehydrating nodes from DNC: %v", err)
			}
		}(rehydrateCtx)
	}

	// Block until context is cancelled or unrecoverable error from c.controller
	select {
	case <-ctx.Done():
		// stop the controllermanager and return
		close(ctrlMgrExitChan)
		return nil
	case err := <-ctrlMgrErrChan:
		// there was an error running the c.controller, so return it to the caller
		return err
	}
}

func (c *controllerManager) reconcileDNCNodes(ctx context.Context) error {
	nodes, err := c.syncDNCClient.GetNodes(ctx, c.nodeSet)
	if err != nil {
		return fmt.Errorf("unable to rehydrate cluster from DNC nodes due to error: %w", err)
	}

	k8sClient := c.ctrlRuntimeMgr.GetClient()
	retrier := retry.Retrier{
		Attempts: 30,
		Delay:    time.Second,
	}
	for _, node := range nodes.Nodes {
		if err := retrier.Do(ctx, func() error {
			// TODO make a best-effort to get these labels from the NNC
			return createNodeNetworkConfig(ctx, k8sClient, node.Name, c.nncNamespace, c.initialIPCount, nil)
		}); err != nil {
			return err
		}

		c.logger.WithFields(logrus.Fields{
			"object":  fmt.Sprintf("%s/%s", c.nncNamespace, node.Name),
			"ipCount": strconv.FormatInt(c.initialIPCount, 10),
		}).Infof("Created NodeNetworkConfig %s in namespace %s with IP Count %d",
			node.Name, c.nncNamespace, c.initialIPCount)
	}

	return nil
}
