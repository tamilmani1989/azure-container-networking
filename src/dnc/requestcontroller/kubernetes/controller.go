// Copyright 2020 Microsoft. All rights reserved.

package kubernetes

import (
	"context"
	"io/ioutil"
	"log"
	"time"

	dncclient "dnc/client"
	"dnc/requestcontroller"

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
	NNCFinalizerName = "finalizers.acn.azure.com/dnc-operations"
	VnetGUIDLabel    = "kubernetes.azure.com/podnetwork-resourceguid"
	SubnetNameLabel  = "kubernetes.azure.com/podnetwork-subnet"
)

// RequestController implements requestcontroller.RequestController. RequestController watches Nodes and NetworkConfigs
// and calls DNC functions as appropriate.
type RequestController struct {
	manager.Manager
	nodeReconciler *NodeReconciler
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

	// Read the kubeconfig
	kubeconfigBytes, err := ioutil.ReadFile(config.KubernetesConfig.KubeconfigPath)
	if err != nil {
		return nil, err
	}

	// Create a REST config from the kubeconfig file
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
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

	// Create a manager from the REST config and scheme
	mgr, err := ctrl.NewManager(restConfig, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: "0", //disable metrics, TODO: make this configurable
	})
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

	dncClientAsyncWait, err := time.ParseDuration(config.DNCClientConfig.WaitBetweenAsyncCalls)
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
		Client:                  mgr.GetClient(),
		DncClient:               dncclient.NewDncClient(config.DNCClientConfig.DNCEndpoint),
		DNCClientRequestTimeout: dncClientRequestTimeout,
		DNCClientAsyncWait:      dncClientAsyncWait,
		ScalerConfig:            config.RequestScalerConfig,
		NodeIDToNCID:            cache.New(cache.NoExpiration, cache.NoExpiration),
		RegisteredNodes:         cache.New(cache.NoExpiration, cache.NoExpiration),
	}
	err = ctrl.NewControllerManagedBy(mgr).
		For(&acnv1alpha.NodeNetworkConfig{}).
		WithEventFilter(NodeNetworkConfigEventFilter{}).
		Complete(nodeNetworkConfigReconciler)

	return &RequestController{
		Manager:        mgr,
		nodeReconciler: nodeReconciler,
	}, nil
}

// Start starts the RequestController
func (r *RequestController) Start(exitChan chan struct{}) error {
	log.Printf("starting requestcontroller")
	return r.Manager.Start(exitChan)
}
