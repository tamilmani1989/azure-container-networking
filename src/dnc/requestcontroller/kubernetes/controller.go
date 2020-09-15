// Copyright 2020 Microsoft. All rights reserved.

package kubernetes

import (
	"context"
	"fmt"
	"log"
	"time"

	dncclient "dnc/client"

	"dnc/requestcontroller"
	"dnc/requestcontroller/datastore"
	syncdncclient "dnc/requestcontroller/dncclient"

	acnv1alpha "github.com/Azure/azure-container-networking/nodenetworkconfig/api/v1alpha"
	cache "github.com/patrickmn/go-cache"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// NodeFinalizerName is the finalizer that is placed on Nodes
const (
	NNCFinalizerName       = "finalizers.acn.azure.com/dnc-operations"
	DelegatedSubnetIDLabel = "kubernetes.azure.com/podnetwork-resourceguid"
	SubnetNameLabel        = "kubernetes.azure.com/podnetwork-subnet"
)

// RequestController implements requestcontroller.RequestController. RequestController watches Nodes and NetworkConfigs
// and calls DNC functions as appropriate.
type RequestController struct {
	Managers map[string]manager.Manager
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

// NewRequestControllerWithConfigProvider creates a new RequestController from the given ConfigProvider
func NewRequestControllerWithConfigProvider(cp requestcontroller.ConfigProvider) (*RequestController, error) {
	config, err := cp.GetConfig()
	if err != nil {
		return nil, err
	}

	// Add the CRD to the scheme
	scheme := runtime.NewScheme()
	err = clientgoscheme.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}

	err = acnv1alpha.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}

	// Choose a kubeconfig provider
	var kubeconfigProvider requestcontroller.KubeconfigProvider
	kubeconfigInfo := config.KubernetesConfig.KubeconfigInfo //later code references properties a lot, so make future line shorter
	log.Printf("Creating kubeconfig provider for config mode %s", kubeconfigInfo.Mode)
	switch mode := kubeconfigInfo.Mode; mode {
	case requestcontroller.KubeconfigFileMode:
		kubeconfigProvider = requestcontroller.KubeconfigFromFileProvider{kubeconfigInfo.File}
	case requestcontroller.KubeconfigDatastoreMode:
		kubeconfigProvider = datastore.NewKubeconfigProvider(kubeconfigInfo.ManagedDNCGUID, config.DataStoreSettings,
			config.LoggingConfig, kubeconfigInfo.ClientCertificatePath, kubeconfigInfo.ClientAppID, kubeconfigInfo.KubeloginBinaryPath,
			kubeconfigInfo.AzureEnvironment)
	default:
		return nil, fmt.Errorf("Error creating kubeconfig provider from config mode: %s is not a valid mode for config.kubernetesConfig.kubeconfigInfo.mode", mode)
	}

	kubeconfigsBytes, err := kubeconfigProvider.GetBytesForAll(context.TODO())
	if err != nil {
		return nil, err
	}

	k8sClientTimeout, err := time.ParseDuration(config.KubernetesConfig.ClientTimeout)
	if err != nil {
		return nil, err
	}

	dncClientRequestTimeout, err := time.ParseDuration(config.DNCClientConfig.RequestTimeout)
	if err != nil {
		return nil, err
	}

	dncClientPollingInterval, err := time.ParseDuration(config.DNCClientConfig.PollingInterval)
	if err != nil {
		return nil, err
	}

	syncDncClient := syncdncclient.NewSynchronousDNCClient(dncclient.NewDncClient(config.DNCClientConfig.DNCEndpoint), dncClientPollingInterval)

	subnetInfoProvider := datastore.NewSubnetInfoProvider(config.DataStoreSettings, config.LoggingConfig)

	requestController := RequestController{
		Managers: make(map[string]manager.Manager),
	}

	for controllerID, kubeconfigBytes := range kubeconfigsBytes {
		// Create a REST config from the kubeconfig bytes
		log.Printf("Creating client for controller %s", controllerID)
		restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
		if err != nil {
			return nil, err
		}

		// Create a manager from the REST config and scheme
		mgr, err := ctrl.NewManager(restConfig, ctrl.Options{
			Scheme:             scheme,
			MetricsBindAddress: "0", //disable metrics, TODO: make this configurable
		})
		if err != nil {
			return nil, err
		}

		// Add the Node reconciler
		nodeReconciler := &NodeReconciler{
			Client:           mgr.GetClient(),
			Namespace:        config.KubernetesConfig.Namespace,
			InitialIPCount:   config.RequestScalerConfig.BatchSize,
			K8sClientTimeout: k8sClientTimeout,
		}
		err = ctrl.NewControllerManagedBy(mgr).
			For(&corev1.Node{}).
			WithEventFilter(NodeEventFilter{}).
			Complete(nodeReconciler)

			// Add the NodeNetworkConfig reconciler
		nodeNetworkConfigReconciler := &NodeNetworkConfigReconciler{
			Client:             mgr.GetClient(),
			DncClient:          syncDncClient,
			ScalerConfig:       config.RequestScalerConfig,
			SubnetInfoProvider: subnetInfoProvider,
			NodeIDToNCID:       cache.New(cache.NoExpiration, cache.NoExpiration),
			RegisteredNodes:    cache.New(cache.NoExpiration, cache.NoExpiration),
			RegisteredVnets:    cache.New(cache.NoExpiration, cache.NoExpiration),
			RegisteredSubnets:  cache.New(cache.NoExpiration, cache.NoExpiration),
			ReconcileTimeout:   dncClientRequestTimeout,
		}
		err = ctrl.NewControllerManagedBy(mgr).
			For(&acnv1alpha.NodeNetworkConfig{}).
			WithEventFilter(NodeNetworkConfigEventFilter{}).
			Complete(nodeNetworkConfigReconciler)

		requestController.Managers[controllerID] = mgr
	}

	return &requestController, nil
}

// Start starts the RequestController and blocks until there is an error or exitChan is closed
func (r *RequestController) Start(exitChan chan struct{}) error {
	errChan := make(chan error)
	defer close(errChan)
	for controllerID, mgr := range r.Managers {
		log.Printf("Starting manager for controller %s", controllerID)
		go func(m manager.Manager) {
			err := m.Start(exitChan)
			if err != nil {
				errChan <- err
			}
		}(mgr)
	}

	//block until error or exitChan is closed (which stops all managers)
	select {
	case err := <-errChan:
		return err
	case <-exitChan:
		return nil
	}
}
