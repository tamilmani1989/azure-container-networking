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

// Apply config applies the config from the json file into the config object
func (c ConfigProvider) ApplyConfig(config *requestcontroller.Config) error {
	return getConfigFromFile(config, c.ConfigFile)
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
