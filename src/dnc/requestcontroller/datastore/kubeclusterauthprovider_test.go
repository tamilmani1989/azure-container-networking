package datastore

import (
	"context"
	"dnc/requestcontroller"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/sirupsen/logrus"
)

const (
	expectedOrchestratorAppID    = "orchestrator-app-id"
	expectedOrchestratorTenantID = "d4aabfe3-34f6-4c52-b7f2-922d78f6e18f"
	expectedAPIServerEndpoint    = "https://kubernetes"
	expectedClusterRootCA        = "cluster-ca-data"
	expectedClientID             = "client-id"
	expectedClientSecretName     = "client-secret-name"
	expectedOrchestratorID       = "0557d795-9553-4bb9-92a3-46d5052165e8"
	expectedMDNCGUID             = "01025494-6825-4200-af96-a69880326baf"
)

func TestGetAllClusterAuthConfigs(t *testing.T) {
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

	authProvider, err := NewKubeClusterAuthProvider(KubeClusterAuthProviderOptions{
		DataStoreClient: &mockDatastoreClient{
			rows: map[string]*storage.Entity{
				expectedOrchestratorID: {
					PartitionKey: expectedMDNCGUID,
					RowKey:       expectedOrchestratorID,
					Properties: map[string]interface{}{
						orchestratorTypePropertyName:       kubernetesOrchestratorType,
						OrchestratorPropertiesPropertyName: string(jsonBlob),
					},
				},
			},
		},
		SecretProvider:   mockSecretProvider{},
		Logger:           logrus.StandardLogger(),
		AzureEnvironment: "AzurePublicCloud",
		ManagedDNCGUID:   "045da9f8-4201-4f3a-8d88-4f5709a683b0",
		TokenProvider:    &mockTokenProvider{},
	})

	if err != nil {
		t.Fatalf("failed to build auth provider with error: %v", err)
	}

	clusterConfigs, err := authProvider.GetAllClusterAuthConfigs(context.Background())

	if err != nil {
		t.Fatalf("failed to get all cluster configs with error: %v", err)
	}

	if len(clusterConfigs) != 1 {
		t.Fatalf("expected %d cluster auth configs, got %d", 1, len(clusterConfigs))
	}

	clusterConfig, found := clusterConfigs[expectedOrchestratorID]
	// config is uninitilized, so we must refresh it
	if err := clusterConfig.Refresh(context.Background()); err != nil {
		t.Fatalf("could not refresh clusterAuthConfig due to error: %v", err)
	}

	if !found {
		t.Fatalf("expected to find cluster auth config for orchestrator %s, but did not find it", expectedOrchestratorID)
	}

	if clusterConfig.GetConfig().Host != expectedAPIServerEndpoint {
		t.Fatalf("expected API server endpoint %s, but got %s", expectedAPIServerEndpoint, clusterConfig.GetConfig().Host)
	}
}

func TestGetAllClusterAuthConfigsAfterRemoval(t *testing.T) {
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

	authProvider, err := NewKubeClusterAuthProvider(KubeClusterAuthProviderOptions{
		DataStoreClient: &mockDatastoreClient{
			rows: map[string]*storage.Entity{
				expectedOrchestratorID: {
					PartitionKey: expectedMDNCGUID,
					RowKey:       expectedOrchestratorID,
					Properties: map[string]interface{}{
						orchestratorTypePropertyName:       kubernetesOrchestratorType,
						OrchestratorPropertiesPropertyName: string(jsonBlob),
					},
				},
			},
		},
		SecretProvider:   mockSecretProvider{},
		Logger:           logrus.StandardLogger(),
		AzureEnvironment: "AzurePublicCloud",
		ManagedDNCGUID:   "045da9f8-4201-4f3a-8d88-4f5709a683b0",
		TokenProvider:    &mockTokenProvider{},
	})

	if err != nil {
		t.Fatalf("failed to build auth provider with error: %v", err)
	}

	clusterConfigs, err := authProvider.GetAllClusterAuthConfigs(context.Background())

	if err != nil {
		t.Fatalf("failed to get all cluster configs with error: %v", err)
	}

	if len(clusterConfigs) != 1 {
		t.Fatalf("expected %d cluster auth configs, got %d", 1, len(clusterConfigs))
	}

	// remove the cluster from the datastore, then refresh configs
	authProvider.datastoreClient = &mockDatastoreClient{rows: nil}

	clusterConfigs, err = authProvider.GetAllClusterAuthConfigs(context.Background())

	if err != nil {
		t.Fatalf("failed to get all cluster configs with error: %v", err)
	}

	if len(clusterConfigs) != 0 {
		t.Fatalf("expected %d cluster auth configs, got %d", 0, len(clusterConfigs))
	}
}

func TestGetAllClusterAuthConfigsTwice(t *testing.T) {
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

	authProvider, err := NewKubeClusterAuthProvider(KubeClusterAuthProviderOptions{
		DataStoreClient: &mockDatastoreClient{
			rows: map[string]*storage.Entity{
				expectedOrchestratorID: {
					PartitionKey: expectedMDNCGUID,
					RowKey:       expectedOrchestratorID,
					Properties: map[string]interface{}{
						orchestratorTypePropertyName:       kubernetesOrchestratorType,
						OrchestratorPropertiesPropertyName: string(jsonBlob),
					},
				},
			},
		},
		SecretProvider:   mockSecretProvider{},
		Logger:           logrus.StandardLogger(),
		AzureEnvironment: "AzurePublicCloud",
		ManagedDNCGUID:   "045da9f8-4201-4f3a-8d88-4f5709a683b0",
		TokenProvider:    &mockTokenProvider{},
	})

	if err != nil {
		t.Fatalf("failed to build auth provider with error: %v", err)
	}

	clusterConfigs, err := authProvider.GetAllClusterAuthConfigs(context.Background())

	if err != nil {
		t.Fatalf("failed to get all cluster configs with error: %v", err)
	}

	if len(clusterConfigs) != 1 {
		t.Fatalf("expected %d cluster auth configs, got %d", 1, len(clusterConfigs))
	}

	clusterConfig, found := clusterConfigs[expectedOrchestratorID]

	if !found {
		t.Fatalf("expected to find cluster auth config for orchestrator %s, but did not find it", expectedOrchestratorID)
	}

	// refresh the clusterAuthConfig before use
	if err := clusterConfig.Refresh(context.Background()); err != nil {
		t.Fatalf("could not refresh cluster auth config with error: %v", err)
	}

	if clusterConfig.GetConfig().Host != expectedAPIServerEndpoint {
		t.Fatalf("expected API server endpoint %s, but got %s", expectedAPIServerEndpoint, clusterConfig.GetConfig().Host)
	}

	originalRESTConfig := clusterConfig.GetConfig()

	// Get the cluster auth configs again, the *rest.Config should not change
	clusterConfigs, err = authProvider.GetAllClusterAuthConfigs(context.Background())

	if err != nil {
		t.Fatalf("failed to get all cluster configs with error: %v", err)
	}

	if len(clusterConfigs) != 1 {
		t.Fatalf("expected %d cluster auth configs, got %d", 1, len(clusterConfigs))
	}

	clusterConfig, found = clusterConfigs[expectedOrchestratorID]

	if !found {
		t.Fatalf("expected to find cluster auth config for orchestrator %s, but did not find it", expectedOrchestratorID)
	}

	// verify *rest.Configs haven't changed
	secondRESTConfig := clusterConfig.GetConfig()

	// compare pointers, since it is important that the actual pointers don't change, not just values
	if originalRESTConfig != secondRESTConfig {
		t.Fatal("expected *rest.Configs to not change")
	}
}

func TestRefreshConfigForever(t *testing.T) {
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

	authProvider, err := NewKubeClusterAuthProvider(KubeClusterAuthProviderOptions{
		DataStoreClient: &mockDatastoreClient{
			rows: map[string]*storage.Entity{
				expectedOrchestratorID: {
					PartitionKey: expectedMDNCGUID,
					RowKey:       expectedOrchestratorID,
					Properties: map[string]interface{}{
						orchestratorTypePropertyName:       kubernetesOrchestratorType,
						OrchestratorPropertiesPropertyName: string(jsonBlob),
					},
				},
			},
		},
		SecretProvider:   mockSecretProvider{},
		Logger:           logrus.StandardLogger(),
		AzureEnvironment: "AzurePublicCloud",
		ManagedDNCGUID:   "045da9f8-4201-4f3a-8d88-4f5709a683b0",
		TokenProvider:    &mockTokenProvider{},
	})

	if err != nil {
		t.Fatalf("failed to build auth provider with error: %v", err)
	}

	clusterConfigs, err := authProvider.GetAllClusterAuthConfigs(context.Background())

	if err != nil {
		t.Fatalf("failed to get all cluster configs with error: %v", err)
	}

	if len(clusterConfigs) != 1 {
		t.Fatalf("expected %d cluster auth configs, got %d", 1, len(clusterConfigs))
	}

	clusterConfig, found := clusterConfigs[expectedOrchestratorID]

	if !found {
		t.Fatalf("expected to find cluster auth config for orchestrator %s, but did not find it", expectedOrchestratorID)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go clusterConfig.RefreshForever(ctx)

	// sleep past the token expiration and see if it was updated
	time.Sleep(time.Second * 2)
	cancel()

	mockTokenProvider := authProvider.tokenProvider.(*mockTokenProvider)
	refreshCount, err := mockTokenProvider.getRefreshCount(requestcontroller.TokenOptions{
		ServerAppID:              expectedOrchestratorAppID,
		ServerTenantID:           expectedOrchestratorTenantID,
		ClientAppID:              expectedClientID,
		ClientCertificateKeyPair: []byte(expectedClusterRootCA),
		AzureEnvironment:         "AzurePublicCloud",
	})

	if err != nil {
		t.Fatalf("unable to get refresh count with error: %v", err)
	}

	if refreshCount == 0 {
		t.Fatal("expected refresh count to be greater than 0")
	}
}

type mockTokenProvider struct {
	refreshCount map[string]int
}

func (m *mockTokenProvider) GetToken(_ context.Context, opts requestcontroller.TokenOptions) (adal.Token, error) {
	if m.refreshCount == nil {
		m.refreshCount = make(map[string]int)
	}

	_, found := m.refreshCount[opts.ServerAppID]

	if !found {
		m.refreshCount[opts.ServerAppID] = 0
	} else {
		m.refreshCount[opts.ServerAppID] = m.refreshCount[opts.ServerAppID] + 1
	}

	// give the token a 1 second expiry to test the refresh logic
	expires := time.Now().Add(time.Second)
	return adal.Token{
		ExpiresOn: json.Number(fmt.Sprintf("%v", expires.Unix())),
	}, nil
}

func (m *mockTokenProvider) getRefreshCount(opts requestcontroller.TokenOptions) (int, error) {
	refreshCount, found := m.refreshCount[opts.ServerAppID]
	if !found {
		return 0, errors.New("token not found")
	}

	return refreshCount, nil
}

type mockSecretProvider struct{}

func (m mockSecretProvider) GetLatestSecret(ctx context.Context, secretName string) (string, error) {
	return "", nil
}
