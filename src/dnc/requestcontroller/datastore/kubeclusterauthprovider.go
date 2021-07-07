package datastore

import (
	"context"
	"dnc/requestcontroller"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"net/http"
	"sync"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
)

const (
	OrchestratorInstancesConfigurationTableName = "OrchestratorInstancesConfiguration"
	OrchestratorPropertiesPropertyName          = "Properties"
	orchestratorTypePropertyName                = "OrchestratorType"
	kubernetesOrchestratorType                  = "Kubernetes"
	secretProviderFailureMetricName             = "DNCRCSecretProviderFailure"
	datastoreFailureMetricName                  = "DNCRCDatastoreFailure"
	datastoreErrorCodeDimension                 = "ErrorCode"
)

type OrchestratorProperties struct {
	OrchestratorAppID    string `json:"orchestratorAppID"`
	OrchestratorTenantID string `json:"orchestratorTenantID"`
	ClusterRootCA        string `json:"clusterRootCA"`
	APIServerEndpoint    string `json:"apiServerEndpoint"`
	ClientID             string `json:"clientID"`
	ClientSecretName     string `json:"clientSecretName"`
}

// Table:            OrchestratorInstancesConfiguration
// PartitionKey:     <ManagedDNCGuid>
// RowKey:           <OrchestratorGuid>
// Properties:       <json blob> (see OrchestratorProperties above)
// OrchestratorType: <OrchestratorType>

type KubeClusterAuthProvider struct {
	datastoreClient    datastoreClient
	secretProvider     requestcontroller.SecretProvider
	logger             logrus.FieldLogger
	managedDNCGUID     string
	azureEnvironment   string
	clusterAuthConfigs map[string]requestcontroller.ClusterAuthConfig
	tokenProvider      requestcontroller.TokenProvider
	metricsClient      requestcontroller.MetricsClient
}

type KubeClusterAuthProviderOptions struct {
	DataStoreClient  datastoreClient
	SecretProvider   requestcontroller.SecretProvider
	Logger           logrus.FieldLogger
	AzureEnvironment string
	ManagedDNCGUID   string
	TokenProvider    requestcontroller.TokenProvider
	MetricsClient    requestcontroller.MetricsClient
}

func NewKubeClusterAuthProvider(opts KubeClusterAuthProviderOptions) (*KubeClusterAuthProvider, error) {
	k := KubeClusterAuthProvider{
		managedDNCGUID:   opts.ManagedDNCGUID,
		datastoreClient:  opts.DataStoreClient,
		secretProvider:   opts.SecretProvider,
		azureEnvironment: opts.AzureEnvironment,
		logger: opts.Logger.WithFields(logrus.Fields{
			"managedDNCGUID":   opts.ManagedDNCGUID,
			"azureEnvironment": opts.AzureEnvironment,
		}),
		clusterAuthConfigs: make(map[string]requestcontroller.ClusterAuthConfig),
		tokenProvider:      opts.TokenProvider,
		metricsClient:      opts.MetricsClient,
	}

	return &k, nil
}

// GetAllClusterAuthConfigs returns a map of orchestratorID->ClusterAuthConfig. The configs are unititiziled, so the caller MUST Refresh them before use.
func (k *KubeClusterAuthProvider) GetAllClusterAuthConfigs(ctx context.Context) (map[string]requestcontroller.ClusterAuthConfig, error) {
	// Query the DB for all clusters
	rowKeys := make(map[string]struct{})

	// Results could be paginated, so loop until all have been read
	var pageToken interface{}
	hasMoreData := true
	for hasMoreData {
		result, moreData, nextPageToken, status, err := k.datastoreClient.QueryAllRowsInPartition(OrchestratorInstancesConfigurationTableName, k.managedDNCGUID, pageToken)
		hasMoreData = moreData
		if err != nil {
			k.metricsClient.SendMetric(requestcontroller.Metric{
				Name:  datastoreFailureMetricName,
				Value: 1,
				CustomDimensions: map[string]string{
					datastoreErrorCodeDimension: strconv.Itoa(status),
				},
			})
			return nil, fmt.Errorf("could not query datastore for table %s: %w", OrchestratorInstancesConfigurationTableName, err)
		}

		if status != http.StatusOK {
			k.metricsClient.SendMetric(requestcontroller.Metric{
				Name:  datastoreFailureMetricName,
				Value: 1,
				CustomDimensions: map[string]string{
					datastoreErrorCodeDimension: strconv.Itoa(status),
				},
			})
			return nil, fmt.Errorf("unexpected status %d returned by datastore", status)
		}

		for _, entity := range result {
			// Save the row key to compare against later to determine what has been removed
			rowKeys[entity.RowKey] = struct{}{}

			// create or update the ClusterAuthConfig
			var clusterAuthConfig *datastoreClusterAuthConfig
			// If a row has already been read, update instead of creating a new object
			if config, found := k.clusterAuthConfigs[entity.RowKey]; found {
				clusterAuthConfig = config.(*datastoreClusterAuthConfig)
			} else {
				// new row, so create a new clusterAuthConfig
				clusterAuthConfig = &datastoreClusterAuthConfig{
					logger:           k.logger.WithField("orchestratorID", entity.RowKey),
					metricsClient:    k.metricsClient,
					tokenProvider:    k.tokenProvider,
					secretProvider:   k.secretProvider,
					azureEnvironment: k.azureEnvironment,
					orchestratorID:   entity.RowKey,
				}
			}

			orchestratorType, exists := entity.Properties[orchestratorTypePropertyName]
			if !exists {
				k.logger.WithFields(logrus.Fields{
					"rowKey":        entity.RowKey,
					"missingColumn": orchestratorTypePropertyName,
				}).Errorf("row %s is missing column %s", entity.RowKey, orchestratorTypePropertyName)
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

			orchestratorPropertiesJSONBlob, exists := entity.Properties[OrchestratorPropertiesPropertyName]
			if !exists {
				k.logger.WithFields(logrus.Fields{
					"rowKey":        entity.RowKey,
					"missingColumn": OrchestratorPropertiesPropertyName,
				}).Errorf("row %s is missing column %s", entity.RowKey, OrchestratorPropertiesPropertyName)
				continue
			}

			// orchestratorProperties is json blob, so unmarshal it
			var orchestratorProperties OrchestratorProperties
			err := json.Unmarshal([]byte(orchestratorPropertiesJSONBlob.(string)), &orchestratorProperties)
			if err != nil {
				k.logger.WithFields(logrus.Fields{
					"rowKey": entity.RowKey,
					"error":  err.Error(),
				}).Errorf("could not unmarshal orchestratorProperties json blob in row %s with error %v", entity.RowKey, err)
				continue
			}

			clusterAuthConfig.orchestratorProperties = orchestratorProperties
			k.clusterAuthConfigs[entity.RowKey] = clusterAuthConfig
		}

		pageToken = nextPageToken
	}

	// Figure out which orchestrators were removed and remove them from result
	for orchestratorID := range k.clusterAuthConfigs {
		if _, found := rowKeys[orchestratorID]; !found {
			delete(k.clusterAuthConfigs, orchestratorID)
		}
	}

	return k.clusterAuthConfigs, nil
}

type datastoreClusterAuthConfig struct {
	restConfig             *rest.Config
	logger                 logrus.FieldLogger
	metricsClient          requestcontroller.MetricsClient
	mutex                  sync.RWMutex
	orchestratorID         string
	orchestratorProperties OrchestratorProperties
	azureEnvironment       string
	tokenProvider          requestcontroller.TokenProvider
	secretProvider         requestcontroller.SecretProvider
	tokenExpires           time.Time
}

// GetConfig returns the rest.Config to the caller. It does not refresh the config. For refresh, see RefreshForever.
func (d *datastoreClusterAuthConfig) GetConfig() *rest.Config {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.restConfig
}

// Refresh will refresh this ClusterAuthConfig's rest.Config
func (d *datastoreClusterAuthConfig) Refresh(ctx context.Context) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.restConfig == nil {
		d.restConfig = &rest.Config{}
	}

	d.restConfig.Host = d.orchestratorProperties.APIServerEndpoint
	d.restConfig.TLSClientConfig = rest.TLSClientConfig{Insecure: true}

	// Get certificate from secretProvider
	cert, err := d.secretProvider.GetLatestSecret(ctx, d.orchestratorProperties.ClientSecretName)
	if err != nil {
		// TODO: Wrap the keyvault client and move this to the wrapper instead of here
		d.metricsClient.SendMetric(requestcontroller.Metric{
			Name:  secretProviderFailureMetricName,
			Value: 1,
		})
		return fmt.Errorf("could not fetch secret from secret provider: %w", err)
	}

	//base64 decode cert since keyvault stores as base64 encoded string
	// TODO: move this decode to secret provider so caller doesn't have to know this detail
	decodedCert, err := base64.StdEncoding.DecodeString(cert)
	if err != nil {
		return fmt.Errorf("could not decode pfx cert from base64: %w", err)
	}

	var orchestratorIDKey requestcontroller.ContextKey = requestcontroller.OrchestratorIDContextKey
	getTokenCtx := context.WithValue(ctx, orchestratorIDKey, d.orchestratorID)
	token, err := d.tokenProvider.GetToken(getTokenCtx, requestcontroller.TokenOptions{
		ServerAppID:              d.orchestratorProperties.OrchestratorAppID,
		ServerTenantID:           d.orchestratorProperties.OrchestratorTenantID,
		ClientAppID:              d.orchestratorProperties.ClientID,
		ClientCertificateKeyPair: decodedCert,
		AzureEnvironment:         d.azureEnvironment,
	})
	if err != nil {
		return fmt.Errorf("unable to get token for orchestratorID %s: %w", d.orchestratorID, err)
	}

	d.tokenExpires = token.Expires()

	d.restConfig.BearerToken = token.AccessToken

	return nil
}

func (d *datastoreClusterAuthConfig) Expires() time.Time {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.tokenExpires
}
