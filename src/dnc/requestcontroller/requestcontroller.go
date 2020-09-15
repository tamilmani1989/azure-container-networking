// Copyright 2020 Microsoft. All rights reserved.

package requestcontroller

import (
	"context"
	"sync"

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
	defaultConfigKubeconfigInfoMode            = "file"
	defaultConfigKubeloginBinaryPath           = "/usr/local/bin/kubelogin"
	defaultConfigClientCertificatePath         = "/tmp/client-cert.pfx"
	defaultAzureEnvironment                    = "AzurePublicCloud"
	KubeconfigFileMode                         = "file"
	KubeconfigDatastoreMode                    = "datastore"
)

/* Using a global lock since loggershim has a global var that needs to be initialized
   in a threadsafe way. This will be removed when we refactor loggershim to remove the
   global pointer
*/
var LoggershimMutex sync.Mutex

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
	KubeconfigInfo KubeconfigInfo `json:"kubeconfigInfo,omitempty"`
	Namespace      string         `json:"namespace,omitempty"`
	ClientTimeout  string         `json:"clientTimeout,omitempty"`
}

type KubeconfigInfo struct {
	// Mode is one of (file | datastore)
	Mode                  string `json:"mode"`
	File                  string `json:"file"`
	ManagedDNCGUID        string `json:"managedDNCGUID"`
	ClientAppID           string `json:"clientAppID"`
	ClientCertificatePath string `json:"clientCertificatePath"`
	KubeloginBinaryPath   string `json:"kubeloginBinaryPath"`
	AzureEnvironment      string `json:"azureEnvironmment"`
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
	if c.KubernetesConfig.KubeconfigInfo.Mode == "" {
		c.KubernetesConfig.KubeconfigInfo.Mode = defaultConfigKubeconfigInfoMode
	}

	if c.KubernetesConfig.KubeconfigInfo.File == "" {
		c.KubernetesConfig.KubeconfigInfo.File = defaultConfigKubeconfigPath
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

	if c.KubernetesConfig.KubeconfigInfo.ClientCertificatePath == "" {
		c.KubernetesConfig.KubeconfigInfo.ClientCertificatePath = defaultConfigClientCertificatePath
	}

	if c.KubernetesConfig.KubeconfigInfo.KubeloginBinaryPath == "" {
		c.KubernetesConfig.KubeconfigInfo.KubeloginBinaryPath = defaultConfigKubeloginBinaryPath
	}

	if c.KubernetesConfig.KubeconfigInfo.AzureEnvironment == "" {
		c.KubernetesConfig.KubeconfigInfo.AzureEnvironment = defaultAzureEnvironment
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

//KubeconfigProvider gets all kubeconfig files as bytes that this RequestController will use to watch the API server
type KubeconfigProvider interface {
	//GetBytesForAll returns a map of {controller guid -> kubeconfig bytes} for each cluster that this RC watches
	GetBytesForAll(ctx context.Context) (map[string][]byte, error)
}
