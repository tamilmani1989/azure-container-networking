// Copyright 2020 Microsoft. All rights reserved.

package json

import (
	"dnc/requestcontroller"
	"encoding/json"
	"io/ioutil"
	"os"
)

// ConfigProvider is a JSON implementation of a requestcontroller.ConfigProvider
type ConfigProvider struct {
	ConfigFile string
}

// GetConfig returns a RequestControllerConfig
func (c ConfigProvider) GetConfig() (requestcontroller.Config, error) {
	config := requestcontroller.Config{}
	if c.ConfigFile != "" {
		if err := getConfigFromFile(&config, c.ConfigFile); err != nil {
			return requestcontroller.Config{}, err
		}
	}

	// Apply defaults to config if necessary
	(&config).ApplyDefaults()
	return config, nil
}

func getConfigFromFile(config *requestcontroller.Config, filename string) error {
	jsonFile, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer jsonFile.Close()

	bytes, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return err
	}

	return json.Unmarshal(bytes, config)
}
