package json

import (
	"dnc/requestcontroller"
	"reflect"
	"testing"
)

func TestGetConfigFromFile(t *testing.T) {
	configFile := "testdata/config.json"

	config := requestcontroller.DefaultConfig()
	jsonConfigProvider := ConfigProvider{ConfigFile: configFile}
	jsonConfigProvider.ApplyConfig(&config)

	expectedConfig := requestcontroller.DefaultConfig()
	expectedConfig.KubernetesConfig.KubeconfigInfo.File = "C:\\Users\\matlong\\.kube\\config"
	expectedConfig.KubernetesConfig.Namespace = "kube-system"
	expectedConfig.DNCClientConfig.RequestTimeout = "10s"
	expectedConfig.DNCClientConfig.PollingInterval = "1s"
	expectedConfig.DNCClientConfig.DNCEndpoint = "http://127.0.0.1:9000"
	expectedConfig.SubnetInfoProviderConfig.Mode = "config"
	expectedConfig.SubnetInfoProviderConfig.PrimaryTokenID = "cbabdad0-f88a-4791-b6d2-f18078953d23"
	expectedConfig.SubnetInfoProviderConfig.SecondaryTokenID = "92b714b6-0a70-4d65-830f-cbeab1826e01"
	expectedConfig.SubnetInfoProviderConfig.VnetID = "576d174c-57b8-4688-9d53-1b74caa642ca"
	expectedConfig.SubnetInfoProviderConfig.SubnetName = "subnet1"
	expectedConfig.MetricsConfig.NNCReconcilerMetricPublishInterval = "30s"

	if !reflect.DeepEqual(config, expectedConfig) {
		t.Fatalf("config does not match expected, expected %+v, got %+v", expectedConfig, config)
	}
}
