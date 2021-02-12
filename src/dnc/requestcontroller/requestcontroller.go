// Copyright 2020 Microsoft. All rights reserved.

package requestcontroller

import (
	"context"
	"dnc/api"
	"dnc/configuration"
	"fmt"
	"os"
	"sync"
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
	defaultConfigDNCClientDNCIPAddress         = "127.0.0.1"
	defaultConfigDNCClientDNCPort              = "9000"
	defaultConfigKubeconfigInfoMode            = "file"
	defaultConfigKubeloginBinaryPath           = "/usr/local/bin/kubelogin"
	defaultConfigClientCertificatesDirectory   = "/etc/dnc/client-certificates"
	defaultAzureEnvironment                    = "AzurePublicCloud"
	defaultCleanupMode                         = "configmap"
	defaultCleanupPollingInterval              = "30s"
	defaultCleanupConfigMapName                = "dnc-cleanup-signal"
	defaultCleanupConfigMapNamespace           = "kube-system"
	KubeconfigFileMode                         = "file"
	KubeconfigDatastoreMode                    = "datastore"
	SubnetInfoProviderDatastoreMode            = "datastore"
	SubnetInfoProviderConfigMode               = "config"
)

/* Using a global lock since loggershim has a global var that needs to be initialized
   in a threadsafe way. This will be removed when we refactor loggershim to remove the
   global pointer
*/
var LoggershimMutex sync.Mutex

// RequestController handles requests from CNS for IP allocations/deallocations and calls DNC functions
type RequestController interface {
	Start()
}

// Config config for RequestController
type Config struct {
	KubernetesConfig         KubernetesConfig    `json:"kubernetesConfig,omitempty"`
	RequestScalerConfig      RequestScalerConfig `json:"requestScalerConfig,omitempty"`
	DNCClientConfig          DNCClientConfig     `json:"dncClientConfig,omitempty"`
	DataStoreSettings        configuration.DataStoreSettings
	LoggingConfig            LoggingConfig            `json:"loggingConfig,omitempty"`
	CosmosMSISettings        CosmosMSISettings        `json:"cosmosMSISettings,omitempty"`
	LeaderElectionSettings   LeaderElectionSettings   `json:"leaderElectionSettings,omitempty"`
	SubnetInfoProviderConfig SubnetInfoProviderConfig `json:"subnetInfoProviderConfig,omitempty"`
	CleanupConfig            CleanupConfig            `json:"cleanupConfig,omitempty"`
}

// KubernetesConfig config for interacting with kubernetes cluster
type KubernetesConfig struct {
	KubeconfigInfo KubeconfigInfo `json:"kubeconfigInfo,omitempty"`
	Namespace      string         `json:"namespace,omitempty"`
	ClientTimeout  string         `json:"clientTimeout,omitempty"`
}

type KubeconfigInfo struct {
	// Mode is one of (file | datastore)
	Mode                             string       `json:"mode"`
	File                             string       `json:"file"`
	ManagedDNCGUID                   string       `json:"managedDNCGUID"`
	ClientCertificatesDirectory      string       `json:"clientCertificatesDirectory"`
	KubeloginBinaryPath              string       `json:"kubeloginBinaryPath"`
	AzureEnvironment                 string       `json:"azureEnvironment"`
	ClientCertificatesKeyvaultConfig KeyvaultInfo `json:"clientCertificatesKeyvaultConfig"`
}

type KeyvaultInfo struct {
	KeyvaultName     string `json:"keyvaultName"`
	MSIResourceID    string `json:"msiResourceID"`
	AzureEnvironment string `json:"azureEnvironment"`
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

type CosmosMSISettings struct {
	AzureEnvironment string `json:"azureEnvironment,omitempty"`
	MSIResourceID    string `json:"msiResourceID,omitempty"`
	CosmosResourceID string `json:"cosmosResourceID,omitempty"`
}

type LeaderElectionSettings struct {
	Namespace string `json:"namespace,omitempty"`
	Enabled   bool   `json:"enabled,omitempty"`
}

type SubnetInfoProviderConfig struct {
	// Mode is one of (config|datastore)
	Mode             string `json:"mode,omitempty"`
	PrimaryTokenID   string `json:"primaryTokenID,omitempty"`
	SecondaryTokenID string `json:"secondaryTokenID,omitempty"`
	VnetID           string `json:"vnetID,omitempty"`
	SubnetName       string `json:"subnetName,omitempty"`
}

type CleanupConfig struct {
	// Mode must be (configmap), no other mode is currently supported
	Mode               string `json:"mode,omitempty"`
	ConfigMapName      string `json:"configMapName,omitempty"`
	ConfigMapNamespace string `json:"configMapNamespace,omitempty"`
	PollingInterval    string `json:"pollingInterval,omitempty"`
	SuccessSafetyValue string `json:"successSafetyValue,omitempty"`
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
		// Check to see if DNCIpAddress or DncPort is set as env variable during deployment
		dncIPAddress := defaultConfigDNCClientDNCIPAddress
		dncPort := defaultConfigDNCClientDNCPort

		if ipAddress := os.Getenv("DNCIpAddress"); len(ipAddress) != 0 {
			dncIPAddress = ipAddress
		}

		if port := os.Getenv("DNCPort"); len(port) != 0 {
			dncPort = port
		}
		c.DNCClientConfig.DNCEndpoint = "http://" + dncIPAddress + ":" + dncPort
	}

	if c.LoggingConfig.ThresholdForLoggingDataStoreOpLatencyMs == 0 {
		c.LoggingConfig.ThresholdForLoggingDataStoreOpLatencyMs = 1000 // 1 second
	}

	if c.KubernetesConfig.KubeconfigInfo.ClientCertificatesDirectory == "" {
		c.KubernetesConfig.KubeconfigInfo.ClientCertificatesDirectory = defaultConfigClientCertificatesDirectory
	}

	if c.KubernetesConfig.KubeconfigInfo.KubeloginBinaryPath == "" {
		c.KubernetesConfig.KubeconfigInfo.KubeloginBinaryPath = defaultConfigKubeloginBinaryPath
	}

	if c.KubernetesConfig.KubeconfigInfo.AzureEnvironment == "" {
		c.KubernetesConfig.KubeconfigInfo.AzureEnvironment = defaultAzureEnvironment
	}

	if c.SubnetInfoProviderConfig.Mode == "" {
		c.SubnetInfoProviderConfig.Mode = SubnetInfoProviderDatastoreMode
	}

	if c.CleanupConfig.Mode == "" {
		c.CleanupConfig.Mode = defaultCleanupMode
	}

	if c.CleanupConfig.ConfigMapName == "" {
		c.CleanupConfig.ConfigMapName = defaultCleanupConfigMapName
	}

	if c.CleanupConfig.ConfigMapNamespace == "" {
		c.CleanupConfig.ConfigMapNamespace = defaultCleanupConfigMapNamespace
	}

	if c.CleanupConfig.PollingInterval == "" {
		c.CleanupConfig.PollingInterval = defaultCleanupPollingInterval
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
