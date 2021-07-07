// Copyright 2020 Microsoft. All rights reserved.

package requestcontroller

import (
	"context"
	"dnc/api"
	"dnc/retry"
	"fmt"
	"time"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
)

const (
	kubeClusterAuthConfigRefreshIntervalSeconds        = 300
	kubeClusterAuthConfigRefreshOnErrorIntervalSeconds = 30
	OrchestratorIDContextKey                           = "orchestratorID"
	TokenExpirationDelta                               = 6 * time.Hour
)

// ContextKey is used as a key to context.WithValue, which should not use built-in types
type ContextKey string

// RequestController handles requests from CNS for IP allocations/deallocations and calls DNC functions
type RequestController struct {
	KubeClusterAuthProvider KubeClusterAuthProvider
	ClusterManagerBuilder   ClusterManagerBuilder
	Logger                  logrus.FieldLogger
	Config                  Config
	MetricsClient           MetricsClient
	clusterManagerPool      clusterManagerPool
	orchestrators           map[string]struct{}
}

type RequestControllerOptions struct {
	KubeClusterAuthProvider KubeClusterAuthProvider
	ManagerBuilder          ClusterManagerBuilder
	Logger                  logrus.FieldLogger
	Config                  Config
	MetricsClient           MetricsClient
}

func NewRequestController(opts RequestControllerOptions) *RequestController {
	return &RequestController{
		KubeClusterAuthProvider: opts.KubeClusterAuthProvider,
		ClusterManagerBuilder:   opts.ManagerBuilder,
		Logger:                  opts.Logger,
		Config:                  opts.Config,
		MetricsClient:           opts.MetricsClient,
		clusterManagerPool: clusterManagerPool{
			clusterManagers: make(map[string]*ClusterManager),
			logger:          opts.Logger,
		},
		orchestrators: make(map[string]struct{}),
	}
}

// Start starts the request controller and blocks until the context is cancelled
func (r *RequestController) Start(ctx context.Context) error {
	// start the kubeconfig refresh loop
	go r.refreshClusterAuthConfigsForever(ctx)

	// Wait for context cancellation
	<-ctx.Done()

	// stop all the managers before returning
	r.clusterManagerPool.removeAll()

	return ctx.Err()
}

// refreshClusterAuthConfigsForever will refresh the kubeconfigs every 30 seconds, updating the managerpool with the result, until the context is cancelled
func (r *RequestController) refreshClusterAuthConfigsForever(ctx context.Context) {
	timeUntilNextRefresh := time.Second * kubeClusterAuthConfigRefreshIntervalSeconds
	if err := r.refreshClusterAuthConfigs(ctx); err != nil {
		// nothing we can do except retry, so just log the error
		r.Logger.Errorf("Unable to refresh cluster auth configs due to error: %v, will retry", err)
		timeUntilNextRefresh = time.Second * kubeClusterAuthConfigRefreshOnErrorIntervalSeconds
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(timeUntilNextRefresh):
			if err := r.refreshClusterAuthConfigs(ctx); err != nil {
				// nothing we can do except retry, so just log the error
				r.Logger.Error("Unable to refresh kubeconfigs, will retry")
				timeUntilNextRefresh = time.Second * kubeClusterAuthConfigRefreshOnErrorIntervalSeconds
			} else {
				timeUntilNextRefresh = time.Second * kubeClusterAuthConfigRefreshIntervalSeconds
			}
		}
	}
}

func (r *RequestController) refreshClusterAuthConfigs(ctx context.Context) error {
	// get the cluster auth configs
	clusterAuthConfigs, err := r.KubeClusterAuthProvider.GetAllClusterAuthConfigs(ctx)
	if err != nil {
		return fmt.Errorf("error getting all auth configs: %w", err)
	}

	orchestratorIDs := make([]string, len(clusterAuthConfigs))
	i := 0
	for orchestratorID := range clusterAuthConfigs {
		orchestratorIDs[i] = orchestratorID
		i++
	}

	r.Logger.Infof("found %d orchestrators: %v", len(clusterAuthConfigs), orchestratorIDs)

	// build a manager for each and add to the manager pool
	for orchestratorID, clusterAuthConfig := range clusterAuthConfigs {
		if r.clusterManagerPool.has(orchestratorID) {
			// If we already have the clusterManager, but its config is about to expire, then refresh it
			if within(clusterAuthConfig.Expires(), TokenExpirationDelta) {
				// Try to refresh the config. If it doesn't work, skip it and try again later
				if err := clusterAuthConfig.Refresh(ctx); err != nil {
					r.Logger.Infof("auth config for orchestrator %s expires in %v, but failed to refresh token with error: %v; will retry next refresh loop", orchestratorID, time.Until(clusterAuthConfig.Expires()), err)
					continue
				}
				// remove from the pool to stop the existing manager, and next code block will re-add
				r.Logger.Infof("auth config for orchestrator %s about to expire, got new one, so removing the old one from the pool", orchestratorID)
				r.clusterManagerPool.remove(orchestratorID)
			} else {
				// skip this orchestrator since it's already in the pool and not expiring
				r.Logger.Infof("token for orchestrator %s still valid, expires in: %v", orchestratorID, time.Until(clusterAuthConfig.Expires()))
				continue
			}

		}
		go func(orchestratorID string, clusterAuthConfig ClusterAuthConfig) {
			retrier := retry.Retrier{
				Attempts: 10,
				Delay:    time.Second,
			}

			buildErr := retrier.Do(ctx, func() error {
				mgr, err := r.ClusterManagerBuilder.Build(ClusterManagerBuilderOptions{
					ClusterAuthConfig: clusterAuthConfig,
					OrchestratorID:    orchestratorID,
					Logger:            r.Logger.WithField("orchestratorID", orchestratorID),
				})

				if err != nil {
					r.Logger.WithField("orchestratorID", orchestratorID).
						WithError(err).
						Error("error building cluster manager, will retry")
					return err
				}

				r.clusterManagerPool.add(mgr, clusterAuthConfig)
				r.Logger.WithField("orchestraotorID", orchestratorID).
					Info("successfully built a cluster manager for the orchestrator")

				return nil
			})

			if buildErr != nil {
				r.Logger.WithField("orchestratorID", orchestratorID).
					WithError(buildErr).
					Error("error building cluster manager, and no more retries left. Will retry when cluster auth configs are refreshed.")
			}
		}(orchestratorID, clusterAuthConfig)

		// save the orchestrator ID to compare against later
		r.orchestrators[orchestratorID] = struct{}{}
	}

	// remove the managers that don't exist anymore
	for orchestratorID := range r.orchestrators {
		if _, found := clusterAuthConfigs[orchestratorID]; !found {
			r.Logger.Infof("removing orchestrator %s from the pool", orchestratorID)
			r.clusterManagerPool.remove(orchestratorID)
			delete(r.orchestrators, orchestratorID)
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
	ClusterAuthConfig ClusterAuthConfig
	OrchestratorID    string
	Logger            logrus.FieldLogger
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
	GetNodes(ctx context.Context, nodeSet string) (*api.NodesResponse, error)
	UpsertNode(ctx context.Context, nodeID, ipAddress, orchestratorType, infraVnet, nodeSet string) error
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

type KubeClusterAuthProvider interface {
	//GetAllRESTConfigs returns a map of orchestratorID->ClusterAuthConfig for each cluster that this RC watches. The ClusterAuthConfig must be refreshed (see `Refresh(ctx) error`) before use.
	GetAllClusterAuthConfigs(ctx context.Context) (map[string]ClusterAuthConfig, error)
}

type ClusterAuthConfig interface {
	GetConfig() *rest.Config
	Refresh(ctx context.Context) error
	Expires() time.Time
}

//SecretProvider provides a named secret
type SecretProvider interface {
	GetLatestSecret(ctx context.Context, name string) (string, error)
}

type Metric struct {
	Name             string
	Value            float64
	CustomDimensions map[string]string
}

type MetricsClient interface {
	SendMetric(Metric)
}

type ConfigSubnetInfoProvider struct {
	Config SubnetInfoProviderConfig
}

func (c *ConfigSubnetInfoProvider) GetSubnetInfo(ctx context.Context, delegatedSubnetID, subnetName string) (SubnetInfo, error) {
	if delegatedSubnetID != c.Config.VnetID {
		return SubnetInfo{}, fmt.Errorf("info for delegeatedSubnetID %s not found, only have info for %s", delegatedSubnetID, c.Config.VnetID)
	}

	return SubnetInfo{
		DelegatedSubnetID: c.Config.VnetID,
		SubnetName:        c.Config.SubnetName,
		PrimaryToken:      c.Config.PrimaryTokenID,
		SecondaryToken:    c.Config.SecondaryTokenID,
		VnetID:            c.Config.VnetID,
	}, nil
}

type TokenProvider interface {
	GetToken(context.Context, TokenOptions) (adal.Token, error)
}

type TokenOptions struct {
	ServerAppID              string
	ClientAppID              string
	ClientCertificateKeyPair []byte
	ServerTenantID           string
	AzureEnvironment         string
}

// within returns whether the time will occur between now and the end of the duration
func within(t time.Time, d time.Duration) bool {
	endTime := time.Now().Add(d)
	return t.Before(endTime)
}
