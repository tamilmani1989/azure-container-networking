// Copyright 2020 Microsoft. All rights reserved.

package requestcontroller

import (
	"context"
	"dnc/api"
	"dnc/configuration"
)

const (
	defaultConfigKubeconfigPath                = "/etc/kubernetes/config.yaml"
	defaultConfigNamespace                     = "default"
	defaultConfigK8sClientTimeout              = "10s"
	defaultConfigScalerBatchSize               = int64(16)
	defaultConfigScalerReleaseThresholdPercent = int64(150)
	defaultConfigScalerRequestThresholdPercent = int64(50)
	defaultConfigDNCClientRequestTimeout       = "15s"
	defaultConfigDNCClientPollingInterval      = "3s"
	defaultConfigDNCClientDNCEP                = "http://127.0.0.1:9000"
)

// RequestController handles requests from CNS for IP allocations/deallocations and calls DNC functions
type RequestController interface {
	Start(chan struct{}) error
}

// Config config for RequestController
type Config struct {
	KubernetesConfig    KubernetesConfig    `json:"kubernetesConfig,omitempty"`
	RequestScalerConfig RequestScalerConfig `json:"requestScalerConfig,omitempty"`
	DNCClientConfig     DNCClientConfig     `json:"dncClientConfig,omitempty"`
	DataStoreSettings   configuration.DataStoreSettings
	LoggingConfig       LoggingConfig `json:"loggingConfig,omitempty"`
}

// KubernetesConfig config for interacting with kubernetes cluster
type KubernetesConfig struct {
	KubeconfigPath string `json:"kubeconfigPath,omitempty"`
	Namespace      string `json:"namespace,omitempty"`
	ClientTimeout  string `json:"clientTimeout,omitempty"`
}

// RequestScalerConfig config for CNS requests
type RequestScalerConfig struct {
	BatchSize               int64 `json:"batchSize,omitempty"`
	ReleaseThresholdPercent int64 `json:"releaseThresholdPercent,omitempty"`
	RequestThresholdPercent int64 `json:"requestThresholdPercent,omitempty"`
}

// DNCClientConfig config for DNC client
type DNCClientConfig struct {
	RequestTimeout  string `json:"requestTimeout,omitempty"`
	PollingInterval string `json:"pollingInterval,omitempty"`
	DNCEndpoint     string `json:"dncEndpoint,omitempty"`
}

type LoggingConfig struct {
	ThresholdForLoggingDataStoreOpLatencyMs int `json:"thresholdForLoggingDataStoreOpLatencyMs,omitempty"`
}

// ConfigProvider provides a Config via GetConfig
type ConfigProvider interface {
	GetConfig() (Config, error)
}

type SubnetInfo struct {
	DelegatedSubnetID string
	SubnetName        string
	PrimaryToken      string
	SecondaryToken    string
	VnetID            string
}

// ApplyDefaults applies the default values for configs if they are not set
func (c *Config) ApplyDefaults() {
	if c.KubernetesConfig.KubeconfigPath == "" {
		c.KubernetesConfig.KubeconfigPath = defaultConfigKubeconfigPath
	}

	if c.KubernetesConfig.Namespace == "" {
		c.KubernetesConfig.Namespace = "default"
	}

	if c.KubernetesConfig.ClientTimeout == "" {
		c.KubernetesConfig.ClientTimeout = defaultConfigK8sClientTimeout
	}

	if c.RequestScalerConfig.BatchSize == 0 {
		c.RequestScalerConfig.BatchSize = defaultConfigScalerBatchSize
	}

	if c.RequestScalerConfig.RequestThresholdPercent == 0 {
		c.RequestScalerConfig.RequestThresholdPercent = defaultConfigScalerRequestThresholdPercent
	}

	if c.RequestScalerConfig.ReleaseThresholdPercent == 0 {
		c.RequestScalerConfig.ReleaseThresholdPercent = defaultConfigScalerReleaseThresholdPercent
	}

	if c.DNCClientConfig.RequestTimeout == "" {
		c.DNCClientConfig.RequestTimeout = defaultConfigDNCClientRequestTimeout
	}

	if c.DNCClientConfig.PollingInterval == "" {
		c.DNCClientConfig.PollingInterval = defaultConfigDNCClientPollingInterval
	}

	if c.DNCClientConfig.DNCEndpoint == "" {
		c.DNCClientConfig.DNCEndpoint = defaultConfigDNCClientDNCEP
	}

	if c.LoggingConfig.ThresholdForLoggingDataStoreOpLatencyMs == 0 {
		c.LoggingConfig.ThresholdForLoggingDataStoreOpLatencyMs = 1000 // 1 second
	}
}

//SynchronousDNCClient is an idempotent, syncronous DNC client.
type SynchronousDNCClient interface {
	GetNode(ctx context.Context, nodeID string) (*api.NodeInfoResponse, error)
	UpsertNode(ctx context.Context, nodeID, ipAddress, orchestratorType, infraVnet string) error
	DeleteNode(ctx context.Context, nodeID string) error
	GetNC(ctx context.Context, networkID, ncID string) (*api.NetworkContainerResponse, error)
	UpsertNcV2(ctx context.Context, networkID, ncID string, req api.NetworkContainerRequestV2) error
	DeleteNC(ctx context.Context, networkID, ncID string) error
	RegisterVnet(ctx context.Context, networkID string) error
	RegisterSubnet(ctx context.Context, networkID, subnetID, authenticationToken string) error
}

//SubnetInfoProvider get vnet/subnet config for given node. A SubnetInfoProvider's GetSubnetInfo may be called by mutliple concurrent goroutines,
// so its implementation must be threadsafe
type SubnetInfoProvider interface {
	GetSubnetInfo(ctx context.Context, delegatedSubnetID, subnetName string) (SubnetInfo, error)
}
