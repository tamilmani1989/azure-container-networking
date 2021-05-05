// Copyright 2020 Microsoft. All rights reserved.

package datastore

import (
	"bufio"
	"bytes"
	"context"
	"dnc/configuration"
	"dnc/requestcontroller"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/sirupsen/logrus"
)

const (
	expectedKubeconfigFile       = "testdata/expected_kubeconfig.yaml"
	expectedMGUID                = "01025494-6825-4200-af96-a69880326baf"
	expectedOrchestratorID       = "0557d795-9553-4bb9-92a3-46d5052165e8"
	expectedAPIServerEndpoint    = "https://kubernetes"
	expectedClusterRootCA        = "cluster-ca-data"
	expectedAzureEnvironment     = "AzurePublicCloud"
	expectedKubeloginBinaryPath  = "/usr/local/bin/kubelogin"
	expectedClientID             = "client-id"
	expectedClientSecretName     = "client-secret-name"
	expectedOrchestratorAppID    = "orchestrator-app-id"
	expectedOrchestratorTenantID = "d4aabfe3-34f6-4c52-b7f2-922d78f6e18f"
	testOutputDir                = "testdata/output"
)

var logger logrus.FieldLogger = logrus.New()

func TestGetBytesForAll(t *testing.T) {

	config := requestcontroller.DefaultConfig()

	kubeconfigOutputDirectory = testOutputDir

	kubeconfigProvider, err := NewKubeconfigProvider(KubeconfigProviderOptions{
		DataStoreSettings: configuration.DataStoreSettings{
			TestPersistenceStoreInfo: configuration.TestPersistenceStoreInfo{
				Test: "true",
			},
		},
		LoggingConfig:               config.LoggingConfig,
		SecretProvider:              mockSecretProvider{},
		Logger:                      logger,
		ClientCertificatesDirectory: testOutputDir,
		KubeloginBinaryPath:         config.KubernetesConfig.KubeconfigInfo.KubeloginBinaryPath,
		AzureEnvironment:            expectedAzureEnvironment,
		ManagedDNCGUID:              expectedMGUID,
	})
	if err != nil {
		t.Fatalf("could not build kubeconfig provider: %v", err)
	}

	orchestratorProps := OrchestratorProperties{
		OrchestratorAppID:    expectedOrchestratorAppID,
		OrchestratorTenantID: expectedOrchestratorTenantID,
		ClusterRootCA:        base64.StdEncoding.EncodeToString([]byte(expectedClusterRootCA)),
		APIServerEndpoint:    expectedAPIServerEndpoint,
		ClientID:             expectedClientID,
		ClientSecretName:     expectedClientSecretName,
	}

	jsonBlob, err := json.Marshal(orchestratorProps)
	if err != nil {
		t.Fatalf("could not marshal orchestratorProperties to json: %v", err)
	}

	// replace the datastoreClient with our mock
	kubeconfigProvider.datastoreClient = &mockDatastoreClient{
		rows: map[string]*storage.Entity{
			expectedOrchestratorID: {
				PartitionKey: expectedMGUID,
				RowKey:       expectedOrchestratorID,
				Properties: map[string]interface{}{
					orchestratorTypePropertyName:       kubernetesOrchestratorType,
					orchestratorPropertiesPropertyName: string(jsonBlob),
				},
			},
		},
	}

	kubeconfigs, err := kubeconfigProvider.GetBytesForAll(context.Background())
	if err != nil {
		t.Fatalf("failed getting kubeconfigs, %v", err)
	}

	// Read the expected kubeconfig file, and drop the windows line endings if present
	expectedKubeconfigBytes, err := readFileIgnoreCarriageReturn(expectedKubeconfigFile)
	if err != nil {
		t.Fatalf("could not read testdata file, %v", err)
	}

	// kubeconfig files should match
	if bytes.Compare(kubeconfigs[expectedOrchestratorID], expectedKubeconfigBytes) != 0 {
		t.Fatalf("Generated kubeconfig doesn't match expected kubeconfig\nExpected: %s\nGenerated: %s", expectedKubeconfigBytes, kubeconfigs[expectedOrchestratorID])
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
