// Copyright 2020 Microsoft. All rights reserved.

package requestcontroller

import "dnc/api"

const (
	defaultConfigKubeconfigPath                 = "/etc/kubernetes/config.yaml"
	defaultConfigNamespace                      = "default"
	defaultConfigK8sClientTimeout               = "10s"
	defaultConfigScalerBatchSize                = int64(16)
	defaultConfigScalerReleaseThresholdPercent  = int64(150)
	defaultConfigScalerRequestThresholdPercent  = int64(50)
	defaultConfigDNCClientRequestTimeout        = "15s"
	defaultConfigDNCClientWaitBetweenAsyncCalls = "3s"
	defaultConfigDNCClientDNCEP                 = "http://127.0.0.1:9000"
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
	RequestTimeout        string `json:"requestTimeout,omitempty"`
	WaitBetweenAsyncCalls string `json:"waitBetweenAsyncCalls,omitempty"`
	DNCEndpoint           string `json:"dncEndpoint,omitempty"`
}

// ConfigProvider provides a Config via GetConfig
type ConfigProvider interface {
	GetConfig() (Config, error)
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

	if c.DNCClientConfig.WaitBetweenAsyncCalls == "" {
		c.DNCClientConfig.WaitBetweenAsyncCalls = defaultConfigDNCClientWaitBetweenAsyncCalls
	}

	if c.DNCClientConfig.DNCEndpoint == "" {
		c.DNCClientConfig.DNCEndpoint = defaultConfigDNCClientDNCEP
	}
}

//DNCClient is an interface that wraps the dnc/client.DncClient functions.
type DNCClient interface {
	GetNode(nodeId string) (*api.NodeInfoResponse, *api.JsonErr)
	AddNode(nodeId, ipaddr, orchestratorType, infraVnet string) *api.JsonErr
	DeleteNode(nodeId string) *api.JsonErr
	GetNCStatus(networkId, ncId string) (*api.RequestStatus, *api.JsonErr)
	GetNC(networkId, ncId string) (*api.NetworkContainerResponse, *api.JsonErr)
	DeleteNC(networkId, ncId string) *api.JsonErr
	UpsertNcV2(networkId, ncId string, req api.NetworkContainerRequestV2) *api.JsonErr
}
