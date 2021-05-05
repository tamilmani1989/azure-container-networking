// Copyright 2020 Microsoft. All rights reserved.

package requestcontroller

import (
	"context"
	"dnc/api"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

/* Using a global lock since loggershim has a global var that needs to be initialized
   in a threadsafe way. This will be removed when we refactor loggershim to remove the
   global pointer
*/
var LoggershimMutex sync.Mutex

const (
	kubeconfigRefreshIntervalSeconds        = 300
	kubeconfigRefreshOnErrorIntervalSeconds = 30
)

// RequestController handles requests from CNS for IP allocations/deallocations and calls DNC functions
type RequestController struct {
	KubeconfigProvider    KubeconfigProvider
	ClusterManagerBuilder ClusterManagerBuilder
	Logger                logrus.FieldLogger
	Config                Config
	clusterManagerPool    clusterManagerPool
	orchestrators         map[string]struct{}
}

type RequestControllerOptions struct {
	KubeconfigProvider KubeconfigProvider
	ManagerBuilder     ClusterManagerBuilder
	Logger             logrus.FieldLogger
	Config             Config
}

func NewRequestController(opts RequestControllerOptions) *RequestController {
	return &RequestController{
		KubeconfigProvider:    opts.KubeconfigProvider,
		ClusterManagerBuilder: opts.ManagerBuilder,
		Logger:                opts.Logger,
		Config:                opts.Config,
		clusterManagerPool: clusterManagerPool{
			clusterManagers: make(map[string]*ClusterManager),
			logger:          opts.Logger,
		},
		orchestrators: make(map[string]struct{}),
	}
}

// Start starts the request controller and blocks until the context is cancelled or until an unrecoverable error occurs
func (r *RequestController) Start(ctx context.Context) error {
	// start the kubeconfig refresh loop
	go r.refreshKubeconfigsForever(ctx)

	// Wait for context cancellation
	<-ctx.Done()

	// stop all the managers before returning
	r.clusterManagerPool.removeAll()

	return ctx.Err()
}

// refreshKubeconfigsForever will refresh the kubeconfigs every 30 seconds, updating the managerpool with the result, until the context is cancelled
func (r *RequestController) refreshKubeconfigsForever(ctx context.Context) {
	timeUntilNextRefresh := time.Second * kubeconfigRefreshIntervalSeconds
	if err := r.refreshKubeconfigs(ctx); err != nil {
		// nothing we can do except retry, so just log the error
		r.Logger.Error("Unable to refresh kubeconfigs, will retry")
		timeUntilNextRefresh = time.Second * kubeconfigRefreshOnErrorIntervalSeconds
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(timeUntilNextRefresh):
			if err := r.refreshKubeconfigs(ctx); err != nil {
				// nothing we can do except retry, so just log the error
				r.Logger.Error("Unable to refresh kubeconfigs, will retry")
				timeUntilNextRefresh = time.Second * kubeconfigRefreshOnErrorIntervalSeconds
			} else {
				timeUntilNextRefresh = time.Second * kubeconfigRefreshIntervalSeconds
			}
		}
	}
}

func (r *RequestController) refreshKubeconfigs(ctx context.Context) error {
	// get the kubeconfigs
	// TODO: make child ctx
	kubeconfigs, err := r.KubeconfigProvider.GetBytesForAll(ctx)
	if err != nil {
		return err
	}

	// build a manager for each and add to the manager pool
	for orchestratorID, kubeconfig := range kubeconfigs {
		mgr, err := r.ClusterManagerBuilder.Build(ClusterManagerBuilderOptions{
			Kubeconfig:     kubeconfig,
			OrchestratorID: orchestratorID,
		})
		if err != nil {
			return err
		}

		r.clusterManagerPool.add(mgr)

		// save the orchestrator ID to compare against later
		r.orchestrators[orchestratorID] = struct{}{}
	}

	// remove the managers that don't exist anymore
	for orchestratorID := range r.orchestrators {
		if _, found := kubeconfigs[orchestratorID]; !found {
			r.clusterManagerPool.remove(orchestratorID)
		}
	}

	return nil
}

// ConfigProvider provides a Config via GetConfig
type ConfigProvider interface {
	ApplyConfig(*Config) error
}

type ClusterManagerBuilder interface {
	Build(ClusterManagerBuilderOptions) (*ClusterManager, error)
}

type ClusterManagerBuilderOptions struct {
	Kubeconfig     []byte
	OrchestratorID string
}

type SubnetInfo struct {
	DelegatedSubnetID string
	SubnetName        string
	PrimaryToken      string
	SecondaryToken    string
	VnetID            string
}

//SynchronousDNCClient is an idempotent, syncronous DNC client.
type SynchronousDNCClient interface {
	GetNode(ctx context.Context, nodeID string) (*api.NodeInfoResponse, error)
	GetNodes(ctx context.Context) (*api.NodesResponse, error)
	UpsertNode(ctx context.Context, nodeID, ipAddress, orchestratorType, infraVnet string) error
	DeleteNode(ctx context.Context, nodeID string) error
	GetNC(ctx context.Context, networkID, ncID string) (*api.NetworkContainerResponse, error)
	UpsertNcV2(ctx context.Context, networkID, ncID string, req api.NetworkContainerRequestV2) error
	DeleteNC(ctx context.Context, networkID, ncID string) error
	RegisterVnet(ctx context.Context, networkID string) error
	UnregisterVnet(ctx context.Context, networkID string) error
	RegisterSubnet(ctx context.Context, networkID, subnetID, authenticationToken string) error
	GetVnets(ctx context.Context) (*api.NetworksResponse, error)
}

//SubnetInfoProvider get vnet/subnet config for given node. A SubnetInfoProvider's GetSubnetInfo may be called by mutliple concurrent goroutines,
// so its implementation must be threadsafe
type SubnetInfoProvider interface {
	GetSubnetInfo(ctx context.Context, delegatedSubnetID, subnetName string) (SubnetInfo, error)
}

//KubeconfigProvider gets all kubeconfig files as bytes that this RequestController will use to watch the API server
type KubeconfigProvider interface {
	//GetBytesForAll returns a map of {controller guid -> kubeconfig bytes} for each cluster that this RC watches
	GetBytesForAll(ctx context.Context) (map[string][]byte, error)
}

//SecretProvider provides a named secret
type SecretProvider interface {
	GetLatestSecret(ctx context.Context, name string) (string, error)
}

type MetricsPublisher interface {
	SendMetric()
	GetDuration() time.Duration
}

type ConfigSubnetInfoProvider struct {
	Config SubnetInfoProviderConfig
}

func (c *ConfigSubnetInfoProvider) GetSubnetInfo(ctx context.Context, delegatedSubnetID, subnetName string) (SubnetInfo, error) {
	if delegatedSubnetID != c.Config.VnetID {
		return SubnetInfo{}, fmt.Errorf("Info for delegeatedSubnetID %s not found, only have info for %s", delegatedSubnetID, c.Config.VnetID)
	}

	return SubnetInfo{
		DelegatedSubnetID: c.Config.VnetID,
		SubnetName:        c.Config.SubnetName,
		PrimaryToken:      c.Config.PrimaryTokenID,
		SecondaryToken:    c.Config.SecondaryTokenID,
		VnetID:            c.Config.VnetID,
	}, nil
}
