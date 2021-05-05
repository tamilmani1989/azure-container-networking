// Copyright 2020 Microsoft. All rights reserved.

package env

import (
	"dnc/requestcontroller"
	"fmt"
	"os"
)

const (
	DNCIPAddressVarName = "DNCIpAddress"
	DNCPortVarName      = "DNCPort"
)

// ConfigProvider is a env variable implementation of a requestcontroller.ConfigProvider
type ConfigProvider struct{}

// ApplyConfig applies environment variables into the config object
func (c ConfigProvider) ApplyConfig(config *requestcontroller.Config) error {
	// If both environment variables are set, change the dnc endpoint
	if ipAddress := os.Getenv(DNCIPAddressVarName); len(ipAddress) != 0 {
		if port := os.Getenv(DNCPortVarName); len(port) != 0 {
			config.DNCClientConfig.DNCEndpoint = fmt.Sprintf("http://%s:%s", ipAddress, port)
		}
	}

	return nil
}
