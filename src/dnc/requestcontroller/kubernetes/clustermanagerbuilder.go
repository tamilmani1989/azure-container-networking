package kubernetes

import (
	"context"
	"dnc/requestcontroller"
	"fmt"
	"time"

	acnv1alpha "github.com/Azure/azure-container-networking/nodenetworkconfig/api/v1alpha"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	ctrlrun "sigs.k8s.io/controller-runtime"
	ctrl "sigs.k8s.io/controller-runtime/pkg/controller"
)

type ClusterManagerBuilder struct {
	Config             requestcontroller.Config
	SyncDNCClient      requestcontroller.SynchronousDNCClient
	Logger             logrus.FieldLogger
	SubnetInfoProvider requestcontroller.SubnetInfoProvider
	MetricsClient      requestcontroller.MetricsClient
}

func (c *ClusterManagerBuilder) Build(opts requestcontroller.ClusterManagerBuilderOptions) (*requestcontroller.ClusterManager, error) {
	// refresh the ClusterAuthConfig to make sure it is fresh
	if err := opts.ClusterAuthConfig.Refresh(context.TODO()); err != nil {
		return nil, fmt.Errorf("could not refresh cluster auth config: %w", err)
	}

	// Build a crd client
	crdClient, err := getCRDClientForRESTConfig(opts.ClusterAuthConfig.GetConfig())
	if err != nil {
		return nil, fmt.Errorf("error creating client for CRD definition creation: %w", err)
	}

	// Get in-cluster config for leader election configmaps
	var leaderElectionConfig *rest.Config
	if c.Config.LeaderElectionSettings.Enabled {
		leaderElectionConfig = ctrlrun.GetConfigOrDie()
	}

	// Parse durations for clients
	k8sClientTimeout, err := time.ParseDuration(c.Config.KubernetesConfig.ClientTimeout)
	if err != nil {
		return nil, fmt.Errorf("error parsing k8sClientTimeout duration from %s: %w", c.Config.KubernetesConfig.ClientTimeout, err)
	}

	dncClientRequestTimeout, err := time.ParseDuration(c.Config.DNCClientConfig.RequestTimeout)
	if err != nil {
		return nil, fmt.Errorf("error parsing dncClientRequestTimeout duration from %s: %w", c.Config.DNCClientConfig.RequestTimeout, err)
	}

	nncReconcilerMetricPublishInterval, err := time.ParseDuration(c.Config.MetricsConfig.NNCReconcilerMetricPublishInterval)
	if err != nil {
		return nil, fmt.Errorf("error parsing NNC Processed Metric publishing interval from %s: %w", c.Config.MetricsConfig.NNCReconcilerMetricPublishInterval, err)
	}

	// resync reconcilers every hour
	resyncPeriod := time.Hour

	ctrlRuntimeMgr, err := ctrlrun.NewManager(opts.ClusterAuthConfig.GetConfig(), ctrlrun.Options{
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
		MetricsClient:    c.MetricsClient,
		Recorder:         ctrlRuntimeMgr.GetEventRecorderFor("dnc-rc/node-reconciler"),
	})

	if err := ctrlrun.NewControllerManagedBy(ctrlRuntimeMgr).
		For(&corev1.Node{}).
		WithEventFilter(NodeEventFilter{}).
		WithOptions(ctrl.Options{
			MaxConcurrentReconciles: c.Config.KubernetesConfig.ConcurrentNodeReconcilers,
		}).
		Complete(nodeReconciler); err != nil {
		return nil, fmt.Errorf("error creating Node reconciler: %w", err)
	}

	var nodeSet string
	if c.Config.DNCClientConfig.UseNodeSets {
		nodeSet = opts.OrchestratorID
	}

	// build the NNC reconciler and add it to the manager
	nodeNetworkConfigReconciler := NewNodeNetworkConfigReconciler(NodeNetworkConfigReconcilerOptions{
		K8sClient:                   ctrlRuntimeMgr.GetClient(),
		DNCClient:                   c.SyncDNCClient,
		ScalerConfig:                c.Config.RequestScalerConfig,
		SubnetInfoProvider:          c.SubnetInfoProvider,
		Logger:                      c.Logger,
		MetricsClient:               c.MetricsClient,
		ReconcileTimeout:            dncClientRequestTimeout,
		ProcessedNNCMetricsDuration: nncReconcilerMetricPublishInterval,
		OrchestratorID:              opts.OrchestratorID,
		SkipSubnetJoin:              c.Config.DNCClientConfig.SkipSubnetJoin,
		Recorder:                    ctrlRuntimeMgr.GetEventRecorderFor("dnc-rc/nnc-reconciler"),
		NodeSet:                     nodeSet,
	})

	if err := ctrlrun.NewControllerManagedBy(ctrlRuntimeMgr).
		For(&acnv1alpha.NodeNetworkConfig{}).
		WithEventFilter(NodeNetworkConfigEventFilter{}).
		WithOptions(ctrl.Options{
			MaxConcurrentReconciles: c.Config.KubernetesConfig.ConcurrentNNCReconcilers,
		}).
		Complete(nodeNetworkConfigReconciler); err != nil {
		return nil, fmt.Errorf("error creating NNC reconciler: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &requestcontroller.ClusterManager{
		RunningContext: requestcontroller.CancellableContext{
			Context: ctx,
			Cancel:  cancel,
		},
		ControllerManager: &controllerManager{
			ctrlRuntimeMgr:             ctrlRuntimeMgr,
			crdClient:                  crdClient,
			syncDNCClient:              c.SyncDNCClient,
			logger:                     c.Logger.WithField("orchestratorID", opts.OrchestratorID),
			nncNamespace:               c.Config.KubernetesConfig.Namespace,
			initialIPCount:             c.Config.RequestScalerConfig.BatchSize,
			enableStartupNodeReconcile: c.Config.FeatureFlags.EnableStartupNodeReconcile,
			nodeSet:                    nodeSet,
			metricsClient:              c.MetricsClient,
		},
		OrchestratorID: opts.OrchestratorID,
	}, nil
}
