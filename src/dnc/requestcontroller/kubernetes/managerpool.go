// Copyright 2020 Microsoft. All rights reserved.

package kubernetes

import (
	"dnc/requestcontroller"
	"fmt"
	"log"
	"sync"
	"time"

	acnv1alpha "github.com/Azure/azure-container-networking/nodenetworkconfig/api/v1alpha"
	cache "github.com/patrickmn/go-cache"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
)

const MaxBackoffSeconds = 16

// ManagerPool contains a set of Managers (k8s clients) that RC will use to watch for Nodes/NNCs
type ManagerPool struct {
	managers           map[string]*manager
	poolMutex          sync.Mutex
	scheme             *runtime.Scheme
	config             requestcontroller.Config
	subnetInfoProvider requestcontroller.SubnetInfoProvider
	syncDNCClient      requestcontroller.SynchronousDNCClient
}

type manager struct {
	exitChan        chan struct{}
	kubeconfigBytes []byte
	managerMutex    sync.Mutex
	state           managerState
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
		log.Printf("Stopping manager for orchestrator %s", mgrToDelete)
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
		log.Printf("Stopping manager for orchestrator %s", orchestratorID)
		if mgr.exitChan != nil {
			close(mgr.exitChan)
		}

		delete(m.managers, orchestratorID)
	}
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
		log.Printf("Starting manager for orchestrator %s", orchestratorID)
		mgrExitChan := make(chan struct{})

		go func() {
			err := m.run(orchestratorID, scheme, config, subnetInfoProvider, syncDNCClient, mgrExitChan)
			if err != nil {
				errChan <- err
			}
		}()

		select {
		case <-time.After(time.Hour):
			// Restart the manager to refresh credentials
			close(mgrExitChan)
			backoffDuration = 0 // No backoff because no error
		case err := <-errChan:
			log.Printf("Manager for orchestrator %s failed with error: %v", orchestratorID, err)
			// Restart the manager after backoff
			close(mgrExitChan)
			backoffDuration = getNextBackoff(backoffDuration)
		case <-m.exitChan:
			log.Printf("Stopping manager for orchestrator %s", orchestratorID)
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
	m.managerMutex.Lock()
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(m.kubeconfigBytes)
	m.managerMutex.Unlock()
	if err != nil {
		return fmt.Errorf("Error building REST config from kubeconfig: %w", err)
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
	if err != nil {
		return fmt.Errorf("Error creating Node reconciler: %w", err)
	}

	// Add the NNC reconciler
	nodeNetworkConfigReconciler := &NodeNetworkConfigReconciler{
		Client:             mgr.GetClient(),
		DncClient:          syncDNCClient,
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
	if err != nil {
		return fmt.Errorf("Error creating NNC reconciler: %w", err)
	}

	return mgr.Start(exitChan)
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
