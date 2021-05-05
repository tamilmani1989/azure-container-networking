package kubernetes

import (
	"context"
	"dnc/requestcontroller"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"

	acnv1alpha "github.com/Azure/azure-container-networking/nodenetworkconfig/api/v1alpha"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrlrun "sigs.k8s.io/controller-runtime"
	ctrl "sigs.k8s.io/controller-runtime/pkg/controller"
)

type ClusterManagerBuilder struct {
	Config             requestcontroller.Config
	SyncDNCClient      requestcontroller.SynchronousDNCClient
	Logger             logrus.FieldLogger
	SubnetInfoProvider requestcontroller.SubnetInfoProvider
}

func (c *ClusterManagerBuilder) Build(opts requestcontroller.ClusterManagerBuilderOptions) (*requestcontroller.ClusterManager, error) {
	// Build a REST config from the kubeconfig
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(opts.Kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("Error building REST config from kubeconfig: %w", err)
	}

	// Build a crd client
	crdClient, err := getCRDClientForKubeconfig(opts.Kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("Error creating client for CRD definition creation: %w", err)
	}

	// Get in-cluster config for leader election configmaps
	var leaderElectionConfig *rest.Config
	if c.Config.LeaderElectionSettings.Enabled {
		leaderElectionConfig = ctrlrun.GetConfigOrDie()
	}

	// Parse durations for clients
	k8sClientTimeout, err := time.ParseDuration(c.Config.KubernetesConfig.ClientTimeout)
	if err != nil {
		return nil, fmt.Errorf("Error parsing k8sClientTimeout duration from %s: %w", c.Config.KubernetesConfig.ClientTimeout, err)
	}

	dncClientRequestTimeout, err := time.ParseDuration(c.Config.DNCClientConfig.RequestTimeout)
	if err != nil {
		return nil, fmt.Errorf("Error parsing dncClientRequestTimeout duration from %s: %w", c.Config.DNCClientConfig.RequestTimeout, err)
	}

	nncReconcilerMetricPublishInterval, err := time.ParseDuration(c.Config.MetricsConfig.NNCReconcilerMetricPublishInterval)
	if err != nil {
		return nil, fmt.Errorf("Error parsing NNC Processed Metric publishing interval from %s: %w", c.Config.MetricsConfig.NNCReconcilerMetricPublishInterval, err)
	}

	cleanupPollingInterval, err := time.ParseDuration(c.Config.CleanupConfig.PollingInterval)
	if err != nil {
		return nil, fmt.Errorf("Error parsing cleanup polling interval from %s: %w", c.Config.CleanupConfig.PollingInterval, err)
	}

	// resync reconcilers every hour
	resyncPeriod := time.Hour

	ctrlRuntimeMgr, err := ctrlrun.NewManager(restConfig, ctrlrun.Options{
		Scheme:                  scheme,
		MetricsBindAddress:      "0", //disable metrics for now, TODO: enable via config
		LeaderElection:          c.Config.LeaderElectionSettings.Enabled,
		LeaderElectionNamespace: c.Config.LeaderElectionSettings.Namespace,
		LeaderElectionID:        fmt.Sprintf("%s%s", opts.OrchestratorID, leaderElectionConfigmapSuffix),
		LeaderElectionConfig:    leaderElectionConfig,
		SyncPeriod:              &resyncPeriod,
	})

	if err != nil {
		return nil, err
	}

	// build the node reconciler and add it to the manager
	nodeReconciler := NewNodeReconciler(NodeReconcilerOptions{
		K8sClient:        ctrlRuntimeMgr.GetClient(),
		DNCClient:        c.SyncDNCClient,
		Namespace:        c.Config.KubernetesConfig.Namespace,
		InitialIPCount:   c.Config.RequestScalerConfig.BatchSize,
		K8sClientTimeout: k8sClientTimeout,
		Logger:           c.Logger,
	})

	if err := ctrlrun.NewControllerManagedBy(ctrlRuntimeMgr).
		For(&corev1.Node{}).
		WithEventFilter(NodeEventFilter{}).
		WithOptions(ctrl.Options{
			MaxConcurrentReconciles: c.Config.KubernetesConfig.ConcurrentNodeReconcilers,
		}).
		Complete(nodeReconciler); err != nil {
		return nil, fmt.Errorf("Error creating Node reconciler: %w", err)
	}

	// build the NNC reconciler and add it to the manager
	nodeNetworkConfigReconciler := NewNodeNetworkConfigReconciler(NodeNetworkConfigReconcilerOptions{
		K8sClient:                   ctrlRuntimeMgr.GetClient(),
		DNCClient:                   c.SyncDNCClient,
		ScalerConfig:                c.Config.RequestScalerConfig,
		SubnetInfoProvider:          c.SubnetInfoProvider,
		Logger:                      c.Logger,
		ReconcileTimeout:            dncClientRequestTimeout,
		ProcessedNNCMetricsDuration: nncReconcilerMetricPublishInterval,
		OrchestratorID:              opts.OrchestratorID,
		SkipSubnetJoin:              c.Config.DNCClientConfig.SkipSubnetJoin,
	})

	if err := ctrlrun.NewControllerManagedBy(ctrlRuntimeMgr).
		For(&acnv1alpha.NodeNetworkConfig{}).
		WithEventFilter(NodeNetworkConfigEventFilter{}).
		WithOptions(ctrl.Options{
			MaxConcurrentReconciles: c.Config.KubernetesConfig.ConcurrentNNCReconcilers,
		}).
		Complete(nodeNetworkConfigReconciler); err != nil {
		return nil, fmt.Errorf("Error creating NNC reconciler: %w", err)
	}

	//build the ConfigMapCleaner
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("Error creating kube client for Cleaner: %w", err)
	}

	// Get untyped rest client for cleaner
	unversionedRESTClient, err := getUnversionedRESTClient(opts.Kubeconfig)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &requestcontroller.ClusterManager{
		RunningContext: requestcontroller.CancellableContext{
			Context: ctx,
			Cancel:  cancel,
		},
		ControllerManager: &controllerManager{
			ctrlRuntimeMgr:    ctrlRuntimeMgr,
			crdClient:         crdClient,
			syncDNCClient:     c.SyncDNCClient,
			logger:            c.Logger,
			metricsPublishers: nodeNetworkConfigReconciler.getMetricsPublishers(),
			cleaner: NewConfigmapCleaner(CleanerOpts{
				KubeCoreV1Client:    clientset.CoreV1(),
				KubeRESTClient:      unversionedRESTClient,
				SyncDNCClient:       c.SyncDNCClient,
				Logger:              c.Logger,
				SignalCheckInterval: cleanupPollingInterval,
				Signal: CleanupSignal{
					ConfigMapName:        c.Config.CleanupConfig.ConfigMapName,
					ConfigMapNamespace:   c.Config.CleanupConfig.ConfigMapNamespace,
					ConfigMapSafetyValue: c.Config.CleanupConfig.SuccessSafetyValue,
				},
				NNCNamespace: c.Config.KubernetesConfig.Namespace,
			}),
			nncNamespace:   c.Config.KubernetesConfig.Namespace,
			initialIPCount: c.Config.RequestScalerConfig.BatchSize,
		},
		OrchestratorID: opts.OrchestratorID,
	}, nil
}

func getUnversionedRESTClient(kubeconfig []byte) (*rest.RESTClient, error) {
	/* When reusing the restConfig that is given to the managers, we see an issue where the scheme gets corrupted
	   and starts dumping lots of errors to stdout. While these errors seem benign (rc keeps working), we can just build
	   a new restConfig here so we don't dump a bunch of errors to stdout */
	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("Error building REST config from kubeconfig: %w", err)
	}

	newScheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(newScheme); err != nil {
		return nil, err
	}
	if err := acnv1alpha.AddToScheme(newScheme); err != nil {
		return nil, err
	}
	config.ContentConfig.GroupVersion = &acnv1alpha.GroupVersion
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.NewCodecFactory(newScheme)
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	return rest.UnversionedRESTClientFor(config)
}
