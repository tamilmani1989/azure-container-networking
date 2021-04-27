// Copyright 2020 Microsoft. All rights reserved.

package kubernetes

import (
	"context"
	"fmt"
	"time"

	"dnc/requestcontroller"

	acnv1alpha "github.com/Azure/azure-container-networking/nodenetworkconfig/api/v1alpha"
	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

// NodeFinalizerName is the finalizer that is placed on Nodes
const (
	NNCFinalizerName                        = "finalizers.acn.azure.com/dnc-operations"
	InfraVnetIdLabel                        = "kubernetes.azure.com/nodenetwork-vnetguid"
	DelegatedSubnetIDLabel                  = "kubernetes.azure.com/podnetwork-delegationguid"
	SubnetNameLabel                         = "kubernetes.azure.com/podnetwork-subnet"
	leaderElectionConfigmapSuffix           = ".resourcelock.acn.azure.com"
	kubeconfigRefreshIntervalSeconds        = 300
	kubeconfigRefreshOnErrorIntervalSeconds = 30
	nncPluralName                           = "nodenetworkconfigs"
	nncShortName                            = "nnc"
	defaultMaxIPCount                       = 250
	maxIPCountLowerBound                    = 1
	maxIPCountUpperBound                    = 250
)

// RequestController implements requestcontroller.RequestController. RequestController watches Nodes and NetworkConfigs
// and calls DNC functions as appropriate.
type RequestController struct {
	managerPool        *ManagerPool
	kubeconfigProvider requestcontroller.KubeconfigProvider
	logger             logrus.FieldLogger
}

// RequestControllerOptions options used to create requestcontroller instance
type RequestControllerOptions struct {
	KubeconfigProvider requestcontroller.KubeconfigProvider
	DNCClient          requestcontroller.SynchronousDNCClient
	SubnetInfoProvider requestcontroller.SubnetInfoProvider
	Config             requestcontroller.Config
	Logger             logrus.FieldLogger
}

var _ requestcontroller.RequestController = &RequestController{}

// K8sClient is an interface that wraps only the functions of controller-runtime/pkg/client.Client that the reconcilers use
// Unit tests can implement this interface to mock a K8s client
type K8sClient interface {
	Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error
	Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error
	Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error
	Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error
	Status() client.StatusWriter
}

// NewRequestController creates a new RequestController from the given ConfigProvider
func NewRequestController(opts RequestControllerOptions) (*RequestController, error) {
	// Add the CRD to the scheme
	scheme := runtime.NewScheme()
	err := clientgoscheme.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}

	err = acnv1alpha.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}

	requestController := RequestController{
		managerPool: &ManagerPool{
			managers:           make(map[string]*manager),
			scheme:             scheme,
			subnetInfoProvider: opts.SubnetInfoProvider,
			config:             opts.Config,
			syncDNCClient:      opts.DNCClient,
			logger:             opts.Logger,
		},
		kubeconfigProvider: opts.KubeconfigProvider,
		logger:             opts.Logger,
	}

	return &requestController, nil
}

// Start starts the RequestController and blocks until there is an error or exitChan is closed
func (r *RequestController) Start() {
	exitChan := signals.SetupSignalHandler()

	// Register all managers
	ctx, cancel := context.WithCancel(context.Background())
	go r.registerAndStartManagersForever(ctx)

	//block until exitChan is closed
	select {
	case <-exitChan:
		r.logger.Info("Stopping all managers")
		cancel()
		r.managerPool.UnregisterAll()
	}
}

func (r *RequestController) registerAndStartManagers(ctx context.Context) error {
	kubeconfigsBytes, err := r.kubeconfigProvider.GetBytesForAll(ctx)
	if err != nil {
		return fmt.Errorf("Error getting kubeconfigs: %w", err)
	}

	// Register the orchestrators with the manager pool
	r.managerPool.RegisterManagers(kubeconfigsBytes)

	// Start all the managers (in case there are new ones)
	r.managerPool.StartAll()

	return nil
}

func (r *RequestController) registerAndStartManagersForever(ctx context.Context) {
	err := r.registerAndStartManagers(ctx)
	if err != nil {
		r.logger.Info("Failed to get k8s clients: %v, will retry in 30s", err)
	}
	for {
		select {
		case <-ctx.Done():
			r.logger.Info("register managers loop was cancelled, no longer checking for new k8s clusters")
			return
		case <-time.After(30 * time.Second):
			// Refresh kubeconfigs after 30 seconds
			err := r.registerAndStartManagers(ctx)
			if err != nil {
				r.logger.Info("Failed to get k8s clients: %v, will retry in 30s", err)
			}
		}
	}
}
