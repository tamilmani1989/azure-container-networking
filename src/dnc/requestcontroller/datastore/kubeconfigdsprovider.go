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
	"net/http"
	"os"
	"path"
	"text/template"

	"github.com/sirupsen/logrus"
)

const (
	orchestratorInstancesConfigurationTableName = "OrchestratorInstancesConfiguration"
	orchestratorPropertiesPropertyName          = "Properties"
	orchestratorTypePropertyName                = "OrchestratorType"
	kubernetesOrchestratorType                  = "Kubernetes"
	kubeconfigOutputDirectory                   = "/etc/kubeconfigs" // TODO: make this configurable
)

type KubeconfigProvider struct {
	dataStoreShim               datastoreshim.DataStoreShim
	kubeconfigTemplate          *template.Template
	secretProvider              requestcontroller.SecretProvider
	logger                      logrus.FieldLogger
	managedDNCGUID              string
	clientCertificatesDirectory string
	kubeloginBinaryPath         string
	azureEnvironment            string
}

type KubeconfigProviderOptions struct {
	DataStoreSettings           configuration.DataStoreSettings
	LoggingConfig               requestcontroller.LoggingConfig
	SecretProvider              requestcontroller.SecretProvider
	Logger                      logrus.FieldLogger
	ClientCertificatesDirectory string
	KubeloginBinaryPath         string
	AzureEnvironment            string
	ManagedDNCGUID              string
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

func NewKubeconfigProvider(opts KubeconfigProviderOptions) (*KubeconfigProvider, error) {
	//datastoreshim has a dependency on loggershim, and loggershim has a global that needs to be initialized with the NewLoggerShim function followed by Init
	requestcontroller.LoggershimMutex.Lock()
	if loggershim.LS == nil {
		// loggershim.LS hasn't been initialized elsewhere, so do it here
		_ = loggershim.NewLoggerShim(&configuration.Configuration{})
		loggershim.LS.Init(true, nil)
		loggershim.LS.SetLogFileLimits(16*1024*1024, 8) // Taken from dnc/main.go: Set 16MB as the max log file size
	}

	requestcontroller.LoggershimMutex.Unlock()

	ds, err := datastoreshim.NewDataStoreShim(opts.DataStoreSettings, opts.LoggingConfig.ThresholdForLoggingDataStoreOpLatencyMs, map[string]string{orchestratorInstancesConfigurationTableName: orchestratorPropertiesPropertyName})
	if err != nil {
		return nil, err
	}

	k := KubeconfigProvider{
		managedDNCGUID:              opts.ManagedDNCGUID,
		dataStoreShim:               ds,
		secretProvider:              opts.SecretProvider,
		clientCertificatesDirectory: opts.ClientCertificatesDirectory,
		kubeloginBinaryPath:         opts.KubeloginBinaryPath,
		kubeconfigTemplate:          template.Must(template.New("kubeconfig").Parse(kubeconfigTemplateString)),
		azureEnvironment:            opts.AzureEnvironment,
		logger: opts.Logger.WithFields(logrus.Fields{
			"managedDNCGUID":   opts.ManagedDNCGUID,
			"azureEnvironment": opts.AzureEnvironment,
		}),
	}

	// Ensure the output directory exists
	if _, err := os.Stat(kubeconfigOutputDirectory); os.IsNotExist(err) {
		if mkdirErr := os.Mkdir(kubeconfigOutputDirectory, os.ModeDir); mkdirErr != nil {
			return nil, mkdirErr
		}
	}

	return &k, nil
}

func (k *KubeconfigProvider) GetBytesForAll(ctx context.Context) (map[string][]byte, error) {
	// Query the DB for all kubeconfig params
	allKubeconfigs := make(map[string][]byte)

	// Results could be paginated, so loop until all have been read
	var pageToken interface{}
	hasMoreData := true
	for hasMoreData {
		result, moreData, nextPageToken, status, err := k.dataStoreShim.QueryAllRowsInPartition(orchestratorInstancesConfigurationTableName, k.managedDNCGUID, pageToken)
		hasMoreData = moreData
		if err != nil {
			return nil, fmt.Errorf("Error querying datastore for table %s: %w", orchestratorInstancesConfigurationTableName, err)
		}

		if status != http.StatusOK {
			return nil, fmt.Errorf("Unexpected status %d returned by datastore", status)
		}

		for _, entity := range result {
			orchestratorType, exists := entity.Properties[orchestratorTypePropertyName]
			if !exists {
				k.logger.WithFields(logrus.Fields{
					"rowKey":        entity.RowKey,
					"missingColumn": orchestratorTypePropertyName,
				}).Infof("Row %s is missing column %s", entity.RowKey, orchestratorTypePropertyName)
				continue
			}

			if orchestratorType != kubernetesOrchestratorType {
				k.logger.WithFields(logrus.Fields{
					"rowKey":                  entity.RowKey,
					"orchestratorType":        orchestratorType,
					"desiredOrchestratorType": kubernetesOrchestratorType,
				}).Infof("Skipping row %s because orchestrator type is %s, not %s", entity.RowKey, orchestratorType, kubernetesOrchestratorType)
				continue
			}

			controllerPropertiesJSONBlob, exists := entity.Properties[orchestratorPropertiesPropertyName]
			if !exists {
				k.logger.WithFields(logrus.Fields{
					"rowKey":        entity.RowKey,
					"missingColumn": orchestratorPropertiesPropertyName,
				}).Infof("Row %s is missing column %s", entity.RowKey, orchestratorPropertiesPropertyName)
				continue
			}

			// orchestratorProperties is json blob, so unmarshall it
			var orchestratorProperties OrchestratorProperties
			err := json.Unmarshal([]byte(controllerPropertiesJSONBlob.(string)), &orchestratorProperties)
			if err != nil {
				k.logger.WithFields(logrus.Fields{
					"rowKey": entity.RowKey,
					"error":  err.Error(),
				}).Infof("Error unmarshalling controllerProperties json blob in row %s with error %v", entity.RowKey, err)
				continue
			}

			kubeconfig, err := k.getKubeconfigForOrchestratorProperties(ctx, orchestratorProperties)
			if err != nil {
				k.logger.WithFields(logrus.Fields{
					"rowKey": entity.RowKey,
					"error":  err.Error(),
				}).Infof("Error building kubeconfig for controller %s: %v", entity.RowKey, err)
				continue
			}

			allKubeconfigs[entity.RowKey] = kubeconfig
		}

		pageToken = nextPageToken
	}

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

	kubeconfigBytes := buffer.Bytes()

	// Save to file for easier debugging
	filename := fmt.Sprintf("%s/%s.yaml", kubeconfigOutputDirectory, orchestratorProperties.OrchestratorAppID)
	if err := ioutil.WriteFile(filename, kubeconfigBytes, 0700); err != nil {
		k.logger.Errorf("Error saving kubeconfig file to disk: %v", err)
	}

	return kubeconfigBytes, nil
}
