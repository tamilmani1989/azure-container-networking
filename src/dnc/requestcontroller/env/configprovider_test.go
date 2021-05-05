package env

import (
	"dnc/requestcontroller"
	"fmt"
	"os"
	"reflect"
	"testing"
)

func TestGetDNCEndpointFromEnv(t *testing.T) {
	// set the env vars
	newIP := "10.0.0.10"
	newPort := "9999"
	os.Setenv(DNCIPAddressVarName, "10.0.0.10")
	os.Setenv(DNCPortVarName, "9999")

	config := requestcontroller.DefaultConfig()
	envConfigProvider := &ConfigProvider{}
	if err := envConfigProvider.ApplyConfig(&config); err != nil {
		t.Fatalf("could not get config from environment: %v", err)
	}

	expectedConfig := requestcontroller.DefaultConfig()
	expectedConfig.DNCClientConfig.DNCEndpoint = fmt.Sprintf("http://%s:%s", newIP, newPort)

	if !reflect.DeepEqual(config, expectedConfig) {
		t.Fatalf("config doesn't match expected, expected %+v, got %+v", expectedConfig, config)
	}
}
