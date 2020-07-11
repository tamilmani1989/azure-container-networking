package kubernetes

import (
	"context"
	"io/ioutil"
	"log"
	"time"

	"dnc/requestcontroller"

	acnv1alpha "github.com/Azure/azure-container-networking/nodenetworkconfig/api/v1alpha"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// NodeFinalizerName is the finalizer that is placed on Nodes
const NodeFinalizerName = "finalizers.acn.azure.com/delete-after-nodenetworkconfig"

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
		Scheme: scheme,
	})
	if err != nil {
		return nil, err
	}

	k8sClientTimeout, err := time.ParseDuration(config.KubernetesConfig.ClientTimeout)
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
		Client: mgr.GetClient(),
	}
	err = ctrl.NewControllerManagedBy(mgr).
		For(&acnv1alpha.NodeNetworkConfig{}).
		Complete(nodeNetworkConfigReconciler)

	return &RequestController{
		Manager:        mgr,
		nodeReconciler: nodeReconciler,
	}, nil
}

// WithOverrideCreateNodeNetworkConfigFunc overrides the default NodeNetworkConfig creation
func (r *RequestController) WithOverrideCreateNodeNetworkConfigFunc(f func(context.Context, string) error) *RequestController {
	// check for nil here so we can chain function calls
	if r == nil {
		return nil
	}

	if r.nodeReconciler == nil {
		return r
	}

	r.nodeReconciler.OverrideCreateNetworkConfigFunc = f
	return r
}

// WithOverrideDeleteNodeNetworkConfigFunc overrides the default NodeNetworkConfig deletion
func (r *RequestController) WithOverrideDeleteNodeNetworkConfigFunc(f func(context.Context, string) error) *RequestController {
	// check for nil here so we can chain function calls
	if r == nil {
		return nil
	}

	if r.nodeReconciler == nil {
		return r
	}

	r.nodeReconciler.OverrideDeleteNetworkConfigFunc = f
	return r
}

// Start starts the RequestController
func (r *RequestController) Start(exitChan chan struct{}) error {
	log.Printf("starting requestcontroller")
	return r.Manager.Start(exitChan)
}
