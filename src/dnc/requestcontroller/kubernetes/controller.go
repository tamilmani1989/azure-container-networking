package kubernetes

import (
	"io/ioutil"
	"log"

	"dnc/requestcontroller"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
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

	// Create a manager from the REST config
	mgr, err := ctrl.NewManager(restConfig, ctrl.Options{})
	if err != nil {
		return nil, err
	}

	// Add the Node reconciler
	nodeReconciler := &NodeReconciler{
		Client: mgr.GetClient(),
	}
	err = ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		WithEventFilter(NodeEventFilter{}).
		Complete(nodeReconciler)

	// TODO Add the NodeNetworkConfig reconciler

	return &RequestController{
		Manager:        mgr,
		nodeReconciler: nodeReconciler,
	}, nil
}

// WithOverrideCreateNodeNetworkConfigFunc overrides the default NodeNetworkConfig creation
func (r *RequestController) WithOverrideCreateNodeNetworkConfigFunc(f func(string) error) *RequestController {
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
func (r *RequestController) WithOverrideDeleteNodeNetworkConfigFunc(f func(string) error) *RequestController {
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
