// Copyright 2020 Microsoft. All rights reserved.

package datastore

import (
	"bufio"
	"bytes"
	"context"
	"dnc/requestcontroller/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

const (
	enableIntegrationTestEnvVar = "ENABLE_INTEGRATION_TESTS"
	configFileEnvVar            = "CONFIG_FILE"
	defaultConfigFile           = "testdata/config_test.json"
)

func TestGetBytesForAllWithCosmos(t *testing.T) {
	if _, set := os.LookupEnv(enableIntegrationTestEnvVar); !set {
		t.Skip(fmt.Sprintf("Skipping integration test, set env var %s to enable", enableIntegrationTestEnvVar))
	}

	rowKey := "0557d795-9553-4bb9-92a3-46d5052165e8"

	// Read the expected kubeconfig file, and drop the windows line endings if present
	expectedKubeconfigBytes, err := readFileIgnoreCarriageReturn("testdata/expected_kubeconfig.yaml")
	if err != nil {
		t.Fatalf("Could not read testdata file, %v", err)
	}

	configFile := os.Getenv(configFileEnvVar)
	if configFile == "" {
		configFile = defaultConfigFile
	}

	configProvider := json.ConfigProvider{ConfigFile: configFile}
	config, err := configProvider.GetConfig()
	if err != nil {
		t.Fatalf("Failed getting config with error: %v", err)
	}

	kubeconfigProvider := NewKubeconfigProvider(config.KubernetesConfig.KubeconfigInfo.ManagedDNCGUID, config.DataStoreSettings, config.LoggingConfig,
		mockSecretProvider{}, config.KubernetesConfig.KubeconfigInfo.ClientCertificatesDirectory, config.KubernetesConfig.KubeconfigInfo.KubeloginBinaryPath,
		config.KubernetesConfig.KubeconfigInfo.AzureEnvironment)

	kubeconfigs, err := kubeconfigProvider.GetBytesForAll(context.Background())
	if err != nil {
		t.Fatalf("Failed getting kubeconfigs, %v", err)
	}

	if bytes.Compare(kubeconfigs[rowKey], expectedKubeconfigBytes) != 0 {
		t.Fatalf("Generated kubeconfig doesn't match expected kubeconfig\nExpected: %s\nGenerated: %s", expectedKubeconfigBytes, kubeconfigs[rowKey])
	}
}

func readFileIgnoreCarriageReturn(filename string) ([]byte, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	var content []byte
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, err
		}

		hasNewLine := strings.HasSuffix(line, "\n")
		// strip the newline + carriage return
		line = strings.TrimRight(line, "\r\n")
		// put the newline back if it was stripped
		if hasNewLine {
			line = line + "\n"
		}

		content = append(content, []byte(line)...)
		if err != nil {
			break
		}
	}

	return content, nil
}

type mockSecretProvider struct{}

func (m mockSecretProvider) GetLatestSecret(ctx context.Context, secretName string) (string, error) {
	return "", nil
}
