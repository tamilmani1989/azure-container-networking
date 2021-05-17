package requestcontroller

import (
	"dnc/configuration"
)

const (
	CleanupConfigmapMode             = "configmap"
	KubeconfigFileMode               = "file"
	KubeconfigDatastoreMode          = "datastore"
	SubnetInfoProviderDatastoreMode  = "datastore"
	SubnetInfoProviderConfigMode     = "config"
	ClientCertificatesKeyvaultMode   = "keyvault"
	ClientCertificatesFilesystemMode = "filesystem"
)

func DefaultConfig() Config {
	return Config{
		KubernetesConfig: KubernetesConfig{
			KubeconfigInfo: KubeconfigInfo{
				Mode:                         KubeconfigFileMode,
				File:                         "/etc/kubernetes/config.yaml",
				ManagedDNCGUID:               "",
				TokenCacheDirectory:          "/etc/requestcontroller/tokencache",
				TokenCacheFilename:           "tokenCache.json",
				AzureEnvironment:             "AzurePublicCloud",
				ClientCertificatesSecretMode: ClientCertificatesKeyvaultMode,
				ClientCertificatesKeyvaultConfig: KeyvaultInfo{
					KeyvaultName:     "",
					MSIResourceID:    "",
					AzureEnvironment: "AzurePublicCloud",
				},
			},
			Namespace:                 "default",
			ClientTimeout:             "10s",
			ConcurrentNodeReconcilers: 100,
			ConcurrentNNCReconcilers:  100,
		},
		RequestScalerConfig: RequestScalerConfig{
			BatchSize:               int64(16),
			ReleaseThresholdPercent: int64(150),
			RequestThresholdPercent: int64(50),
		},
		DNCClientConfig: DNCClientConfig{
			RequestTimeout:  "15s",
			PollingInterval: "3s",
			DNCEndpoint:     "http://127.0.0.1:9000",
			SkipSubnetJoin:  false,
		},
		DataStoreSettings: configuration.DataStoreSettings{
			AzureStorageInfo:         configuration.AzureStorageInfo{},
			TestPersistenceStoreInfo: configuration.TestPersistenceStoreInfo{},
			MaxBatchSize:             0,
		},
		LoggingConfig: LoggingConfig{
			ThresholdForLoggingDataStoreOpLatencyMs: 1000, // 1 second
		},
		CosmosMSISettings: CosmosMSISettings{
			AzureEnvironment: "AzurePublicCloud",
			MSIResourceID:    "",
			CosmosResourceID: "",
		},
		LeaderElectionSettings: LeaderElectionSettings{
			Namespace: "",
			Enabled:   false,
		},
		SubnetInfoProviderConfig: SubnetInfoProviderConfig{
			Mode:             SubnetInfoProviderDatastoreMode,
			PrimaryTokenID:   "",
			SecondaryTokenID: "",
			VnetID:           "",
			SubnetName:       "",
		},
		CleanupConfig: CleanupConfig{
			Mode:               CleanupConfigmapMode,
			ConfigMapName:      "dnc-cleanup-signal",
			ConfigMapNamespace: "kube-system",
			PollingInterval:    "30s",
			SuccessSafetyValue: "",
		},
		TelemetrySettings: configuration.TelemetrySettings{
			DisableTelemetry:             false,
			TelemetryBatchSizeBytes:      16384,
			TelemetryBatchIntervalInSecs: 7,
			DisableMetadataRefreshThread: false,
			RefreshIntervalInSecs:        15,
			DebugMode:                    false,
			DisableAiLogging:             true,
			GetEnvRetryWaitTimeInSecs:    3,
			GetEnvRetryCount:             5,
		},
		MetricsConfig: MetricsConfig{
			NNCReconcilerMetricPublishInterval: "5m",
		},
	}
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
	TelemetrySettings        configuration.TelemetrySettings
	MetricsConfig            MetricsConfig `json:"metricsConfig,omitempty"`
	FeatureFlags             FeatureFlags  `json:"featureFlags,omitempty"`
}

// KubernetesConfig config for interacting with kubernetes cluster
type KubernetesConfig struct {
	KubeconfigInfo            KubeconfigInfo `json:"kubeconfigInfo,omitempty"`
	Namespace                 string         `json:"namespace,omitempty"`
	ClientTimeout             string         `json:"clientTimeout,omitempty"`
	ConcurrentNodeReconcilers int            `json:"concurrentNodeReconcilers,omitempty"`
	ConcurrentNNCReconcilers  int            `json:"concurrentNNCReconcilers,omitempty"`
}

type KubeconfigInfo struct {
	// Mode is one of (file | datastore)
	Mode                             string            `json:"mode"`
	File                             string            `json:"file"`
	ManagedDNCGUID                   string            `json:"managedDNCGUID"`
	AzureEnvironment                 string            `json:"azureEnvironment"`
	ClientCertificatesSecretMode     string            `json:"clientCertificatesSecretMode"`
	ClientCertificatesKeyvaultConfig KeyvaultInfo      `json:"clientCertificatesKeyvaultConfig"`
	ClientCertificatesFromFileConfig map[string]string `json:"clientCertificatesFromFileConfig"`
	TokenCacheDirectory              string            `json:"tokenCacheDirectory,omitempty"`
	TokenCacheFilename               string            `json:"tokenCacheFilename,omitempty"`
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
	SkipSubnetJoin  bool   `json:"skipSubnetJoin,omitempty"`
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

type MetricsConfig struct {
	NNCReconcilerMetricPublishInterval string `json:"nncReconcilerMetricPublishInterval,omitempty"`
}

type FeatureFlags struct {
	EnableStartupNodeReconcile bool `json:"enableStartupNodeReconcile,omitempty"`
}
