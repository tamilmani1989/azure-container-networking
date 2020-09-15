// Copyright 2020 Microsoft. All rights reserved.

package datastore

import (
	"bytes"
	"context"
	"dnc/configuration"
	"dnc/datastoreshim"
	"dnc/logger/loggershim"
	"dnc/requestcontroller"
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"
)

const (
	controllerConfigurationTableName = "ControllerConfiguration"
	controllerPropertiesPropertyName = "Properties"
)

type KubeconfigProvider struct {
	dataStoreShim         *datastoreshim.DataStoreShim
	kubeconfigTemplate    *template.Template
	managedDNCGUID        string
	clientCertificatePath string
	clientAppID           string
	kubeloginBinaryPath   string
	azureEnvironment      string
}

// ControllerPropertiesParent is a struct that holds a ControllerProperties struct. This is necesary to match json schema
type ControllerPropertiesParent struct {
	Properties ControllerProperties `json:"properties"`
}

type ControllerProperties struct {
	ServerAppID       string `json:"serverAppID"`
	ServerTenantID    string `json:"serverTenantID"`
	ClusterRootCA     string `json:"clusterRootCA"`
	APIServerEndpoint string `json:"apiServerEndpoint"`
}

/*
Table:          ControllerConfiguration
PartitionKey:   <ManagedDNCGuid>
RowKey:         <ControllerGuid>
Properties:     <json blob>
ControllerType: <ControllerType>
*/

var _ requestcontroller.KubeconfigProvider = &KubeconfigProvider{}

func NewKubeconfigProvider(managedDNCGUID string, dataStoreSettings configuration.DataStoreSettings, loggingConfig requestcontroller.LoggingConfig,
	clientCertificatePath, clientAppID, kubeloginBinaryPath, azureEnvironment string) *KubeconfigProvider {
	//datastoreshim has a dependency on loggershim, and loggershim has a global that needs to be initialized with the NewLoggerShim function followed by Init
	requestcontroller.LoggershimMutex.Lock()
	if loggershim.LS == nil {
		// loggershim.LS hasn't been initialized elsewhere, so do it here
		_ = loggershim.NewLoggerShim(&configuration.Configuration{})
		loggershim.LS.Init(true, nil)
	}

	requestcontroller.LoggershimMutex.Unlock()

	k := KubeconfigProvider{
		managedDNCGUID: managedDNCGUID,
		dataStoreShim: datastoreshim.NewDataStoreShim(dataStoreSettings, configuration.DNCSettings{
			ThresholdForLoggingDataStoreOpLatencyMs: loggingConfig.ThresholdForLoggingDataStoreOpLatencyMs,
		}, map[string]string{controllerConfigurationTableName: controllerPropertiesPropertyName}),
		clientCertificatePath: clientCertificatePath,
		clientAppID:           clientAppID,
		kubeloginBinaryPath:   kubeloginBinaryPath,
		kubeconfigTemplate:    template.Must(template.New("kubeconfig").Parse(kubeconfigTemplateString)),
		azureEnvironment:      azureEnvironment,
	}

	return &k
}

func (k *KubeconfigProvider) GetBytesForAll(context.Context) (map[string][]byte, error) {
	// Query the DB for all kubeconfig params
	allKubeconfigs := make(map[string][]byte)

	// Results could be paginated, so loop until all have been read
	var pageToken interface{}
	for {
		result, hasMoreData, nextPageToken, status, err := k.dataStoreShim.QueryAllInPartition(controllerConfigurationTableName, k.managedDNCGUID, pageToken)
		if err != nil {
			return nil, err
		}

		if status != http.StatusOK {
			return nil, fmt.Errorf("Unexpected status %d returned by datastore", status)
		}

		// Result is map of row key -> {map of column name -> value}, or map of row key -> value if there is only one column
		// We are expecting at least two columns, so assume result is map of row key -> {map of column name -> value}
		for rowKey, rowVal := range result {
			switch v := rowVal.(type) {
			case map[string]interface{}:
				controllerPropertiesJsonBlob, exists := v[controllerPropertiesPropertyName]
				if !exists {
					return nil, fmt.Errorf("Row %s is missing column %s", rowKey, controllerPropertiesPropertyName)
				}

				// controllerProperties is json blob, so unmarshall it
				var controllerPropertiesParent ControllerPropertiesParent
				err := json.Unmarshal([]byte(controllerPropertiesJsonBlob.(string)), &controllerPropertiesParent)
				if err != nil {
					return nil, fmt.Errorf("Error unmarshalling controllerProperties json blob in row %s with error %v", rowKey, err)
				}

				kubeconfig, err := k.getKubeconfigForControllerProperties(controllerPropertiesParent)
				if err != nil {
					return nil, err
				}

				allKubeconfigs[rowKey] = kubeconfig
			default:
				return nil, fmt.Errorf("Unexpected result for row %s, expected type %s but received type %T", rowKey, "map[string]interface{}", v)
			}
		}

		// Exit the loop if there are no more rows to read in Cosmos
		if !hasMoreData {
			break
		}

		pageToken = nextPageToken
	}

	return allKubeconfigs, nil
}

func (k *KubeconfigProvider) getKubeconfigForControllerProperties(controllerPropertiesParent ControllerPropertiesParent) ([]byte, error) {
	templateInfo := TemplateInfo{
		CAData:                controllerPropertiesParent.Properties.ClusterRootCA,
		Endpoint:              controllerPropertiesParent.Properties.APIServerEndpoint,
		ServerAppID:           controllerPropertiesParent.Properties.ServerAppID,
		ClientAppID:           k.clientAppID,
		ClientCertificatePath: k.clientCertificatePath,
		ServerTenantID:        controllerPropertiesParent.Properties.ServerTenantID,
		KubeloginBinaryPath:   k.kubeloginBinaryPath,
		AzureEnvironment:      k.azureEnvironment,
	}

	var buffer bytes.Buffer
	err := k.kubeconfigTemplate.Execute(&buffer, templateInfo)

	return buffer.Bytes(), err
}
