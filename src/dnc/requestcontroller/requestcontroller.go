package requestcontroller

const defaultConfigKubeconfigPath = "/etc/kubernetes/config.yaml"

// RequestController handles requests from CNS for IP allocations/deallocations and calls DNC functions
type RequestController interface {
	Start(chan struct{}) error
}

// Config config for RequestController
type Config struct {
	KubernetesConfig KubernetesConfig `json:"kubernetesConfig,omitempty"`
}

// KubernetesConfig config for interacting with kubernetes cluster
type KubernetesConfig struct {
	KubeconfigPath string `json:"kubeconfigPath,omitempty"`
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
}
