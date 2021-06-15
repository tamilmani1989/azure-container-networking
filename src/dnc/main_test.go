package main

import (
	"dnc/configuration"
	"os"
	"testing"
)

const (
	apiVersion = "?api-version=2018-03-01"
	apiVersion_2021_02_24 = "?api-version=2021-02-24"
	forceDeleteQueryParam = "force-delete=true"
)

// todo: not threadsafe. same with config
var dncEP string

const (
	testConfigFilePath = "config_test.json"
)

// TestMain :- main
func TestMain(m *testing.M) {
	if err := Setup(); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

func ReadConfig() *configuration.Configuration {
	config = mustReadConfig(testConfigFilePath)
	return config
}
