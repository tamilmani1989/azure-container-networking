// Copyright 2020 Microsoft. All rights reserved.

package kubernetes

import (
	"context"
	"dnc/requestcontroller"
	"dnc/retry"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	acnv1alpha "github.com/Azure/azure-container-networking/nodenetworkconfig/api/v1alpha"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
)

// MaxBackoffSeconds is the maximum backoff between manager restarts
const MaxBackoffSeconds = 16

// ManagerPool contains a set of Managers (k8s clients) that RC will use to watch for Nodes/NNCs
type ManagerPool struct {
	managers           map[string]*manager
	poolMutex          sync.Mutex
	scheme             *runtime.Scheme
	config             requestcontroller.Config
	subnetInfoProvider requestcontroller.SubnetInfoProvider
	syncDNCClient      requestcontroller.SynchronousDNCClient
	logger             logrus.FieldLogger
}

type manager struct {
	exitChan        chan struct{}
	kubeconfigBytes []byte
	managerMutex    sync.Mutex
	state           managerState
	logger          logrus.FieldLogger
	orchestratorID  string
}

type managerState int

const (
	managerStopped managerState = iota
	managerRunning
)

// RegisterManagers will add all managers in the kubeconfigs map to this ManagerPool and remove the managers that are not in the kubeconfigs map
func (m *ManagerPool) RegisterManagers(kubeconfigs map[string][]byte) {
	m.poolMutex.Lock()
	defer m.poolMutex.Unlock()

	// Add all of the managers in kubeconfigs
	for orchestratorID, kubeconfigBytes := range kubeconfigs {
		mgr, found := m.managers[orchestratorID]
		if found {
			mgr.managerMutex.Lock()
			defer mgr.managerMutex.Unlock()
			mgr.kubeconfigBytes = kubeconfigBytes
		} else {
			mgr = &manager{
				kubeconfigBytes: kubeconfigBytes,
				exitChan:        make(chan struct{}),
				state:           managerStopped,
				logger: m.logger.WithFields(logrus.Fields{
					"orchestratorID": orchestratorID,
				}),
				orchestratorID: orchestratorID,
			}
			m.managers[orchestratorID] = mgr
		}
	}

	// Remove the managers that were removed
	existingMgrs := make([]string, len(m.managers))
	i := 0
	for k := range m.managers {
		existingMgrs[i] = k
		i++
	}

	newMgrs := make([]string, len(kubeconfigs))
	i = 0
	for k := range kubeconfigs {
		newMgrs[i] = k
		i++
	}

	removedMgrs := getRemovedItems(existingMgrs, newMgrs)
	for _, mgrToDelete := range removedMgrs {
		m.managers[mgrToDelete].logger.Infof("Stopping manager for orchestrator %s", mgrToDelete)
		if m.managers[mgrToDelete].exitChan != nil {
			close(m.managers[mgrToDelete].exitChan)
		}

		delete(m.managers, mgrToDelete)
	}
}

// StartAll starts all managers in the pool
func (m *ManagerPool) StartAll() {
	m.poolMutex.Lock()
	defer m.poolMutex.Unlock()

	for orchestratorID, mgr := range m.managers {
		go mgr.runForever(orchestratorID, m.scheme, m.config, m.subnetInfoProvider, m.syncDNCClient)
	}
}

// UnregisterAll stops and removes all managers in this ManagerPool
func (m *ManagerPool) UnregisterAll() {
	m.poolMutex.Lock()
	defer m.poolMutex.Unlock()

	for orchestratorID, mgr := range m.managers {
		mgr.stop()
		delete(m.managers, orchestratorID)
	}
}

func (m *manager) stop() {
	m.logger.Infof("Stopping manager for orchestrator %s", m.orchestratorID)
	m.managerMutex.Lock()
	defer m.managerMutex.Unlock()
	if m.exitChan != nil {
		close(m.exitChan)
		m.exitChan = nil
	}
	m.state = managerStopped
}

// runForever runs a manager until its exit channel is closed, restarting if necessary
func (m *manager) runForever(orchestratorID string, scheme *runtime.Scheme, config requestcontroller.Config, subnetInfoProvider requestcontroller.SubnetInfoProvider, syncDNCClient requestcontroller.SynchronousDNCClient) {
	// Only one goroutine should run this, so check state
	m.managerMutex.Lock()
	if m.state == managerRunning {
		m.managerMutex.Unlock()
		return
	}

	m.state = managerRunning
	m.managerMutex.Unlock()

	errChan := make(chan error)
	defer close(errChan)

	var backoffDuration time.Duration

loop:
	for {
		m.logger.Infof("Stopping manager for orchestrator %s", orchestratorID)
		mgrExitChan := make(chan struct{})

		go func() {
			err := m.run(orchestratorID, scheme, config, subnetInfoProvider, syncDNCClient, mgrExitChan)
			/* If the manager exits without error, that means it was cleaned up successfully. No need to send error on error chan, since we don't want it to restart.
			   It is possible that runForever will restart it if the cluster wasn't removed from the kubeconfigprovider,
			   or if RC wasn't terminated before the credential refresh loop, but since the cleanup configmap will still be present, the reconcilers won't restart.
			*/
			if err != nil {
				errChan <- err
			}
		}()

		select {
		case <-time.After(time.Hour):
			// Restart the manager to refresh credentials - TODO (matlong): Is this necessary?
			close(mgrExitChan)
			backoffDuration = 0 // No backoff because no error
		case err := <-errChan:
			m.logger.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Infof("Manager for orchestrator %s failed with error: %v", orchestratorID, err)
			// Restart the manager after backoff
			close(mgrExitChan)
			backoffDuration = getNextBackoff(backoffDuration)
		case <-m.exitChan:
			m.logger.Infof("Stopping manager for orchestrator %s", orchestratorID)
			close(mgrExitChan)
			// Exit loop since manager entity is now stopped
			break loop
		}

		// m.exitChan might be closed during backoff sleep, so use select here to handle that
		select {
		case <-m.exitChan:
			break loop
		case <-time.After(backoffDuration):
			continue
		}
	}

	m.managerMutex.Lock()
	m.state = managerStopped
	m.managerMutex.Unlock()
}

func (m *manager) run(orchestratorID string, scheme *runtime.Scheme, config requestcontroller.Config, subnetInfoProvider requestcontroller.SubnetInfoProvider, syncDNCClient requestcontroller.SynchronousDNCClient, exitChan chan struct{}) error {
	m.logger.Infof("Starting manager for orchestrator: %s", orchestratorID)
	m.managerMutex.Lock()
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(m.kubeconfigBytes)
	m.managerMutex.Unlock()
	if err != nil {
		return fmt.Errorf("Error building REST config from kubeconfig: %w", err)
	}

	// Ensure CRD definition is installed on cluster before starting up manager
	m.logger.Infof("Ensuring NodeNetworkConfig CRD definition exists on cluster")
	crdClientSet, err := apiextensionsclientset.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("Error creating client for CRD definition creation: %w", err)
	}

	retrier := retry.Retrier{
		Attempts: 10,
		Delay:    3 * time.Second,
	}

	// TODO: wire up this context with exitChan so we can immediately stop retrying when exitChan is closed
	ctx := context.Background()
	err = retrier.Do(ctx, func() error {
		ctxWT, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		return m.registerCRD(ctxWT, crdClientSet)
	})

	if err != nil {
		return fmt.Errorf("Failed to setup manager, failed to ensure CRD definition on cluster with error: %w", err)
	}

	// Get in-cluster config for leader election configmaps
	var leaderElectionConfig *rest.Config
	if config.LeaderElectionSettings.Enabled {
		leaderElectionConfig = ctrl.GetConfigOrDie()
	}

	resyncPeriod := time.Hour

	mgr, err := ctrl.NewManager(restConfig, ctrl.Options{
		Scheme:                  scheme,
		MetricsBindAddress:      "0", //disable metrics for now, TODO: enable via config
		LeaderElection:          config.LeaderElectionSettings.Enabled,
		LeaderElectionNamespace: config.LeaderElectionSettings.Namespace,
		LeaderElectionID:        fmt.Sprintf("%s%s", orchestratorID, leaderElectionConfigmapSuffix),
		LeaderElectionConfig:    leaderElectionConfig,
		SyncPeriod:              &resyncPeriod,
	})

	if err != nil {
		return fmt.Errorf("Error building manager: %w", err)
	}

	// Parse durations for clients
	k8sClientTimeout, err := time.ParseDuration(config.KubernetesConfig.ClientTimeout)
	if err != nil {
		return fmt.Errorf("Error parsing k8sClientTimeout duration from %s: %w", config.KubernetesConfig.ClientTimeout, err)
	}

	dncClientRequestTimeout, err := time.ParseDuration(config.DNCClientConfig.RequestTimeout)
	if err != nil {
		return fmt.Errorf("Error parsing dncClientRequestTimeout duration from %s: %w", config.DNCClientConfig.RequestTimeout, err)
	}

	cleanupPollingInterval, err := time.ParseDuration(config.CleanupConfig.PollingInterval)
	if err != nil {
		return fmt.Errorf("Error parsing cleanup polling interval from %s: %w", config.CleanupConfig.PollingInterval, err)
	}

	// Add the Node reconciler
	nodeReconciler := NewNodeReconciler(NodeReconcilerOptions{
		K8sClient:        mgr.GetClient(),
		DNCClient:        syncDNCClient,
		Namespace:        config.KubernetesConfig.Namespace,
		InitialIPCount:   config.RequestScalerConfig.BatchSize,
		K8sClientTimeout: k8sClientTimeout,
		Logger:           m.logger,
	})

	err = ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		WithEventFilter(NodeEventFilter{}).
		Complete(nodeReconciler)
	if err != nil {
		return fmt.Errorf("Error creating Node reconciler: %w", err)
	}

	// Add the NNC reconciler
	nodeNetworkConfigReconciler := NewNodeNetworkConfigReconciler(NodeNetworkConfigReconcilerOptions{
		K8sClient:          mgr.GetClient(),
		DNCClient:          syncDNCClient,
		ScalerConfig:       config.RequestScalerConfig,
		SubnetInfoProvider: subnetInfoProvider,
		Logger:             m.logger,
		ReconcileTimeout:   dncClientRequestTimeout,
	})

	err = ctrl.NewControllerManagedBy(mgr).
		For(&acnv1alpha.NodeNetworkConfig{}).
		WithEventFilter(NodeNetworkConfigEventFilter{}).
		Complete(nodeNetworkConfigReconciler)
	if err != nil {
		return fmt.Errorf("Error creating NNC reconciler: %w", err)
	}

	// Instantiate the cleaner
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("Error creating kube client for Cleaner: %w", err)
	}

	// Get untyped rest client for cleaner
	crdClient, err := getRESTClientForCRD(restConfig, scheme)
	if err != nil {
		return err
	}

	cleaner := NewCleaner(CleanerOpts{
		KubeCoreV1Client:    clientset.CoreV1(),
		KubeRESTClient:      crdClient,
		SyncDNCClient:       syncDNCClient,
		Logger:              m.logger,
		SignalCheckInterval: cleanupPollingInterval,
		Signal: CleanupSignal{
			ConfigMapName:        config.CleanupConfig.ConfigMapName,
			ConfigMapNamespace:   config.CleanupConfig.ConfigMapNamespace,
			ConfigMapSafetyValue: config.DataStoreSettings.AzureStorageInfo.PartitionKey, // TODO: (matlong) this is always set in AKS scenario, but what if other scenarios need cleanup signal?
		},
		NNCNamespace: config.KubernetesConfig.Namespace,
	})

	/* Check if cleanup signal exists. It is possible that this function is called after a credential refresh, after cleanup error, or that the cluster is deleted before the DNC-RC even starts. If so, we can just
	   kick off the cleanup here instead of restarting the reconcilers.
	*/
	cleanupSignal := cleaner.HasCleanupSignal(ctx)
	errChan := make(chan error)
	if cleanupSignal {
		// cleanup
		return cleaner.Clean(ctx)
	}

	// start the reconcilers
	go func() {
		err := mgr.Start(exitChan)
		errChan <- err
	}()

	// Wait for cleanup signal or error from reconcilers
	cleanupCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	select {
	case err := <-errChan:
		// Manager was stopped, so return to restart
		return err
	case <-cleaner.WaitForCleanupSignal(cleanupCtx):
		// Stop the manager, clean up cluster/DNC
		m.stop()
		return cleaner.Clean(cleanupCtx)
	}
}

// getNextBackoff provides a helper for capped exponential backoff
func getNextBackoff(currentBackoff time.Duration) time.Duration {
	if currentBackoff == 0 {
		return time.Second
	}

	currentBackoff = 2 * currentBackoff
	if currentBackoff > MaxBackoffSeconds*time.Second {
		currentBackoff = MaxBackoffSeconds * time.Second
	}

	return currentBackoff
}

// getRemovedItems returns the items in prev that do not exist in curr
func getRemovedItems(prev, curr []string) []string {
	currSet := make(map[string]bool)
	for _, s := range curr {
		currSet[s] = true
	}

	var res []string
	for _, s := range prev {
		if !currSet[s] {
			res = append(res, s)
		}
	}

	return res
}

func (m *manager) registerCRD(ctx context.Context, clientSet *apiextensionsclientset.Clientset) error {
	crd := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: nncPluralName + "." + acnv1alpha.GroupVersion.Group,
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: acnv1alpha.GroupVersion.Group,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    acnv1alpha.GroupVersion.Version,
					Served:  true,
					Storage: true,
					Subresources: &apiextensionsv1.CustomResourceSubresources{
						Status: &apiextensionsv1.CustomResourceSubresourceStatus{},
					},
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
							Description: "NodeNetworkConfig is the Schema for the nodenetworkconfigs API",
							Type:        "object",
							Properties: map[string]apiextensionsv1.JSONSchemaProps{
								"apiVersion": {
									Description: "APIVersion defines the versioned schema of this representation",
									Type:        "string",
								},
								"kind": {
									Description: "Kind is a string value representing the REST resource this object represents",
									Type:        "string",
								},
								"metadata": {
									Type: "object",
								},
								"spec": {
									Type:        "object",
									Description: "NodeNetworkConfigSpec defines the desired state of NetworkConfig",
									Properties: map[string]apiextensionsv1.JSONSchemaProps{
										"ipsNotInUse": {
											Items: &apiextensionsv1.JSONSchemaPropsOrArray{
												Schema: &apiextensionsv1.JSONSchemaProps{
													Type: "string",
												},
											},
											Type: "array",
										},
										"requestedIPCount": {
											Format: "int64",
											Type:   "integer",
										},
									},
								},
								"status": {
									Type:        "object",
									Description: "NodeNetworkConfigStatus defines the observed state of NetworkConfig",
									Properties: map[string]apiextensionsv1.JSONSchemaProps{
										"networkContainers": {
											Type: "array",
											Items: &apiextensionsv1.JSONSchemaPropsOrArray{
												Schema: &apiextensionsv1.JSONSchemaProps{
													Type: "object",
													Properties: map[string]apiextensionsv1.JSONSchemaProps{
														"defaultGateway": {
															Type: "string",
														},
														"ipAssignments": {
															Type: "array",
															Items: &apiextensionsv1.JSONSchemaPropsOrArray{
																Schema: &apiextensionsv1.JSONSchemaProps{
																	Type: "object",
																	Properties: map[string]apiextensionsv1.JSONSchemaProps{
																		"ip": {
																			Type: "string",
																		},
																		"name": {
																			Type: "string",
																		},
																	},
																},
															},
														},
														"id": {
															Type: "string",
														},
														"primaryIP": {
															Type: "string",
														},
														"subnetAddressSpace": {
															Type: "string",
														},
														"subnetName": {
															Type: "string",
														},
														"version": {
															Type:   "integer",
															Format: "int64",
														},
													},
												},
											},
										},
										"scaler": {
											Description: "Scaler groups IP request params together",
											Type:        "object",
											Properties: map[string]apiextensionsv1.JSONSchemaProps{
												"batchSize": {
													Type:   "integer",
													Format: "int64",
												},
												"releaseThresholdPercent": {
													Type:   "integer",
													Format: "int64",
												},
												"requestThresholdPercent": {
													Type:   "integer",
													Format: "int64",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Plural:     nncPluralName,
				Kind:       reflect.TypeOf(acnv1alpha.NodeNetworkConfig{}).Name(),
				ListKind:   reflect.TypeOf(acnv1alpha.NodeNetworkConfig{}).Name() + "List",
				Singular:   strings.ToLower(reflect.TypeOf(acnv1alpha.NodeNetworkConfig{}).Name()),
				ShortNames: []string{nncShortName},
			},
		},
	}

	_, err := clientSet.ApiextensionsV1().CustomResourceDefinitions().Create(ctx, crd, metav1.CreateOptions{})

	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("Error adding NNC CRD definition to cluster: %w", err)
	}

	return nil
}

func getRESTClientForCRD(config *rest.Config, scheme *runtime.Scheme) (*rest.RESTClient, error) {
	config.ContentConfig.GroupVersion = &acnv1alpha.GroupVersion
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.NewCodecFactory(scheme)
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	return rest.UnversionedRESTClientFor(config)
}
