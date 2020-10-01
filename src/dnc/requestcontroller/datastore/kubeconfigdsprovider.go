// Copyright 2020 Microsoft. All rights reserved.

package datastore

import (
	"bytes"
	"context"
	"dnc/configuration"
	"dnc/datastoreshim"
	"dnc/logger/loggershim"
	"dnc/requestcontroller"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"text/template"
)

const (
	orchestratorInstancesConfigurationTableName = "OrchestratorInstancesConfiguration"
	orchestratorPropertiesPropertyName          = "Properties"
	orchestratorTypePropertyName                = "OrchestratorType"
	kubernetesOrchestratorType                  = "Kubernetes"
)

type KubeconfigProvider struct {
	dataStoreShim               *datastoreshim.DataStoreShim
	kubeconfigTemplate          *template.Template
	secretProvider              requestcontroller.SecretProvider
	managedDNCGUID              string
	clientCertificatesDirectory string
	kubeloginBinaryPath         string
	azureEnvironment            string
}

type OrchestratorProperties struct {
	OrchestratorAppID    string `json:"orchestratorAppID"`
	OrchestratorTenantID string `json:"orchestratorTenantID"`
	ClusterRootCA        string `json:"clusterRootCA"`
	APIServerEndpoint    string `json:"apiServerEndpoint"`
	ClientID             string `json:"clientID"`
	ClientSecretName     string `json:"clientSecretName"`
}

/*
Table:            OrchestratorInstancesConfiguration
PartitionKey:     <ManagedDNCGuid>
RowKey:           <OrchestratorGuid>
Properties:       <json blob> (see OrchestratorProperties above)
OrchestratorType: <OrchestratorType>
*/

var _ requestcontroller.KubeconfigProvider = &KubeconfigProvider{}

func NewKubeconfigProvider(managedDNCGUID string, dataStoreSettings configuration.DataStoreSettings, loggingConfig requestcontroller.LoggingConfig,
	secretProvider requestcontroller.SecretProvider, clientCertificatesDirectory, kubeloginBinaryPath, azureEnvironment string) *KubeconfigProvider {
	//datastoreshim has a dependency on loggershim, and loggershim has a global that needs to be initialized with the NewLoggerShim function followed by Init
	requestcontroller.LoggershimMutex.Lock()
	if loggershim.LS == nil {
		// loggershim.LS hasn't been initialized elsewhere, so do it here
		_ = loggershim.NewLoggerShim(&configuration.Configuration{})
		loggershim.LS.Init(true, nil)
		loggershim.LS.SetLogFileLimits(16*1024*1024, 8) // Taken from dnc/main.go: Set 16MB as the max log file size
	}

	requestcontroller.LoggershimMutex.Unlock()

	k := KubeconfigProvider{
		managedDNCGUID: managedDNCGUID,
		dataStoreShim: datastoreshim.NewDataStoreShim(dataStoreSettings, configuration.DNCSettings{
			ThresholdForLoggingDataStoreOpLatencyMs: loggingConfig.ThresholdForLoggingDataStoreOpLatencyMs,
		}, map[string]string{orchestratorInstancesConfigurationTableName: orchestratorPropertiesPropertyName}),
		secretProvider:              secretProvider,
		clientCertificatesDirectory: clientCertificatesDirectory,
		kubeloginBinaryPath:         kubeloginBinaryPath,
		kubeconfigTemplate:          template.Must(template.New("kubeconfig").Parse(kubeconfigTemplateString)),
		azureEnvironment:            azureEnvironment,
	}

	return &k
}

func (k *KubeconfigProvider) GetBytesForAll(ctx context.Context) (map[string][]byte, error) {
	log.Printf("Getting kubeconfigs for all orchestrators")
	// Query the DB for all kubeconfig params
	allKubeconfigs := make(map[string][]byte)

	// Results could be paginated, so loop until all have been read
	var pageToken interface{}
	hasMoreData := true
	for hasMoreData {
		result, moreData, nextPageToken, status, err := k.dataStoreShim.QueryAllInPartition(orchestratorInstancesConfigurationTableName, k.managedDNCGUID, pageToken)
		hasMoreData = moreData
		if err != nil {
			return nil, fmt.Errorf("Error querying datastore for table %s: %w", orchestratorInstancesConfigurationTableName, err)
		}

		if status != http.StatusOK {
			return nil, fmt.Errorf("Unexpected status %d returned by datastore", status)
		}

		// Result is map of row key -> {map of column name -> value}, or map of row key -> value if there is only one column
		// We are expecting at least two columns, so assume result is map of row key -> {map of column name -> value}
		for rowKey, rowVal := range result {
			switch v := rowVal.(type) {
			case map[string]interface{}:
				orchestratorType, exists := v[orchestratorTypePropertyName]
				if !exists {
					log.Printf("Row %s is missing column %s", rowKey, orchestratorTypePropertyName)
					continue
				}

				if orchestratorType != kubernetesOrchestratorType {
					log.Printf("Skipping row %s because orchestrator type is %s, not %s", rowKey, orchestratorType, kubernetesOrchestratorType)
					continue
				}

				controllerPropertiesJSONBlob, exists := v[orchestratorPropertiesPropertyName]
				if !exists {
					log.Printf("Row %s is missing column %s", rowKey, orchestratorPropertiesPropertyName)
					continue
				}

				// orchestratorProperties is json blob, so unmarshall it
				var orchestratorProperties OrchestratorProperties
				err := json.Unmarshal([]byte(controllerPropertiesJSONBlob.(string)), &orchestratorProperties)
				if err != nil {
					log.Printf("Error unmarshalling controllerProperties json blob in row %s with error %v", rowKey, err)
					continue
				}

				kubeconfig, err := k.getKubeconfigForOrchestratorProperties(ctx, orchestratorProperties)
				if err != nil {
					log.Printf("Error building kubeconfig for controller %s: %v", rowKey, err)
					continue
				}

				allKubeconfigs[rowKey] = kubeconfig
			default:
				log.Printf("Unexpected result for row %s, expected type %s but received type %T", rowKey, "map[string]interface{}", v)
				continue
			}
		}

		pageToken = nextPageToken
	}

	log.Printf("Found %d orchestrators", len(allKubeconfigs))

	return allKubeconfigs, nil
}

func (k *KubeconfigProvider) getKubeconfigForOrchestratorProperties(ctx context.Context, orchestratorProperties OrchestratorProperties) ([]byte, error) {
	// Get certificate from secretProvider
	cert, err := k.secretProvider.GetLatestSecret(ctx, orchestratorProperties.ClientSecretName)
	if err != nil {
		return nil, fmt.Errorf("Error fetching secret from secret provider: %w", err)
	}

	//base64 decode cert and store as file for kubelogin to read
	decodedCert, err := base64.StdEncoding.DecodeString(cert)
	if err != nil {
		return nil, fmt.Errorf("Error decoding pfx cert from base64: %w", err)
	}

	certFilename := fmt.Sprintf("%s.pfx", orchestratorProperties.ClientID)
	absCertFilename := path.Join(k.clientCertificatesDirectory, certFilename)

	// Ensure directory exists
	err = os.MkdirAll(k.clientCertificatesDirectory, 0755) // directories need execute bits
	if err != nil {
		return nil, fmt.Errorf("Error ensuring certificate directory %s exists: %w", k.clientCertificatesDirectory, err)
	}

	err = ioutil.WriteFile(absCertFilename, decodedCert, 0644)
	if err != nil {
		return nil, fmt.Errorf("Error writing file %s: %w", absCertFilename, err)
	}

	templateInfo := TemplateInfo{
		CAData:                orchestratorProperties.ClusterRootCA,
		Endpoint:              orchestratorProperties.APIServerEndpoint,
		ServerAppID:           orchestratorProperties.OrchestratorAppID,
		ClientAppID:           orchestratorProperties.ClientID,
		ClientCertificatePath: absCertFilename,
		ServerTenantID:        orchestratorProperties.OrchestratorTenantID,
		KubeloginBinaryPath:   k.kubeloginBinaryPath,
		AzureEnvironment:      k.azureEnvironment,
	}

	var buffer bytes.Buffer
	err = k.kubeconfigTemplate.Execute(&buffer, templateInfo)
	if err != nil {
		return nil, fmt.Errorf("Error executing kubeconfig template: %w", err)
	}

	return buffer.Bytes(), nil
}
