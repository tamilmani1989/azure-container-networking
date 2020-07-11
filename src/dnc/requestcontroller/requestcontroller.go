package requestcontroller

const (
	defaultConfigKubeconfigPath                = "/etc/kubernetes/config.yaml"
	defaultConfigNamespace                     = "default"
	defaultConfigK8sClientTimeout              = "10s"
	defaultConfigScalerBatchSize               = int64(16)
	defaultConfigScalerReleaseThresholdPercent = int64(150)
	defaultConfigScalerRequestThresholdPercent = int64(50)
)

// RequestController handles requests from CNS for IP allocations/deallocations and calls DNC functions
type RequestController interface {
	Start(chan struct{}) error
}

// Config config for RequestController
type Config struct {
	KubernetesConfig    KubernetesConfig    `json:"kubernetesConfig,omitempty"`
	RequestScalerConfig RequestScalerConfig `json:"requestScalerConfig,omitempty"`
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
}
