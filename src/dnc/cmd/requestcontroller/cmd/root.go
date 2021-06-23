package cmd

import (
	"context"
	ad "dnc/activedirectory"
	dncclient "dnc/client"
	"dnc/datastoreshim"
	"dnc/logger/loggershim"
	"dnc/logger/simplejson"
	"dnc/requestcontroller"
	"dnc/requestcontroller/aad"
	"dnc/requestcontroller/appinsights"
	"dnc/requestcontroller/datastore"
	syncdncclient "dnc/requestcontroller/dncclient"
	"dnc/requestcontroller/env"
	"dnc/requestcontroller/filesystem"
	"dnc/requestcontroller/json"
	"dnc/requestcontroller/kubernetes"
	"dnc/utils"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

const (
	shortDesc                     = "Watches K8s api-server for SWIFT-based CNI deployment"
	longDesc                      = "Watches K8s api-server for Nodes, creates/deletes NodeNetworkConfigs, and responds to NodeNetworkConfig changes"
	configFlag                    = "config"
	executable                    = "requestcontroller"
	DNCRCStartupFailureMetricName = "DNCRCStartupFailure"
	MDNCGUIDDimensionName         = "MDNCGUID"
	OrchestratorIDDimensionName   = "OrchestratorID"
)

type options struct {
	logger        logrus.FieldLogger
	metricsClient requestcontroller.MetricsClient
	config        requestcontroller.Config
}

var (
	/* Using a global lock since loggershim has a global var that needs to be initialized
	   in a threadsafe way. This will be removed when we refactor loggershim to remove the
	   global pointer
	*/
	loggershimMutex sync.Mutex

	rootCmd = &cobra.Command{
		Use:   executable,
		Short: shortDesc,
		Long:  longDesc,
		Run: func(cmd *cobra.Command, args []string) {
			// setup the signal handler
			exitChan := signals.SetupSignalHandler()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// read the command line flags
			configFile, err := cmd.Flags().GetString(configFlag)
			if err != nil {
				logrus.Fatal(err, "error getting config flag value")
			}

			// run the request controller
			errChan := make(chan error)
			defer close(errChan)
			go func(errChan chan error) {
				if err := runRequestController(ctx, configFile); err != nil {
					errChan <- err
				}
			}(errChan)

			// wait for an exit signal or error
			select {
			case <-exitChan:
				// return to cancel the context and exit
				return
			case err := <-errChan:
				logrus.Fatalf("error running request controller: %v", err)
			}
		},
	}

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(utils.Version)
		},
	}
)

func init() {
	rootCmd.PersistentFlags().StringP(configFlag, "c", "", "Configuration file")
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(pingDNCCommand)
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func runRequestController(ctx context.Context, configFile string) error {
	logger := logrus.StandardLogger()
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Get the config
	config, err := getConfig(configFile)
	if err != nil {
		return fmt.Errorf("error getting config: %w", err)
	}

	// Create the metrics client
	var persistentDimensions map[string]string
	// For managed dnc, add the mdnc guid to all metrics
	if config.KubernetesConfig.KubeconfigInfo.ManagedDNCGUID != "" {
		persistentDimensions = map[string]string{
			MDNCGUIDDimensionName: config.KubernetesConfig.KubeconfigInfo.ManagedDNCGUID,
		}
	}
	// For AKS, add the orchestrator ID to all metrics
	if config.KubernetesConfig.KubeconfigInfo.OrchestratorID != "" {
		persistentDimensions = map[string]string{
			OrchestratorIDDimensionName: config.KubernetesConfig.KubeconfigInfo.OrchestratorID,
		}
	}
	metricsClient := appinsights.NewMetricsClient(appinsights.MetricsClientOptions{
		Logger:               logger,
		Config:               config.TelemetrySettings,
		AppName:              executable,
		AppVersion:           utils.Version,
		PersistentDimensions: persistentDimensions,
	})

	if err := runRequestControllerWithOptions(ctx, options{
		logger:        logger,
		metricsClient: metricsClient,
		config:        config,
	}); err != nil {
		metricsClient.SendMetric(requestcontroller.Metric{
			Name:  DNCRCStartupFailureMetricName,
			Value: 1,
		})

		return err
	}

	return nil
}

func runRequestControllerWithOptions(ctx context.Context, opts options) error {
	config := opts.config
	metricsClient := opts.metricsClient
	logger := opts.logger

	// Get the Cosmos token via MSI if necessary
	if config.DataStoreSettings.AzureStorageInfo.IsUsingCosmosDb && config.DataStoreSettings.AzureStorageInfo.Key == "" {
		logger.Info("Getting key for CosmosDB with MSI")
		key, err := getCosmosKeyFromMSI(context.TODO(), config.CosmosMSISettings.MSIResourceID, config.CosmosMSISettings.CosmosResourceID, config.CosmosMSISettings.AzureEnvironment)
		if err != nil {
			return fmt.Errorf("error getting Cosmos key from MSI: %w", err)
		}

		// Saw a strange error where if the subscription was incorrect in the CosmosResourceID string, the getCosmosKeyFromMSI function would return an empty key and nil error.
		// The following check ensures that RC doesn't get an empty key and nil error. TODO: We should investigate why the adal functions were not returning errors and failing silently
		if key == "" {
			return fmt.Errorf("error getting Cosmos Key from MSI, key was empty")
		}

		config.DataStoreSettings.AzureStorageInfo.Key = key
	}

	// Choose a kubeconfig provider
	var kubeClusterAuthProvider requestcontroller.KubeClusterAuthProvider
	kubeconfigInfo := config.KubernetesConfig.KubeconfigInfo //later code references these properties a lot, so make future lines shorter
	logger.Infof("Creating kubeconfig provider for config mode %s", kubeconfigInfo.Mode)

	switch mode := kubeconfigInfo.Mode; mode {
	case requestcontroller.KubeconfigFileMode:
		kubeClusterAuthProvider = filesystem.KubeClusterAuthProvider{
			KubeconfigFile: kubeconfigInfo.File,
			OrchestratorID: config.KubernetesConfig.KubeconfigInfo.OrchestratorID,
		}
	case requestcontroller.KubeconfigDatastoreMode:
		// Datastore mode requires a secret provider
		var secretProvider requestcontroller.SecretProvider
		switch mode := kubeconfigInfo.ClientCertificatesSecretMode; mode {
		case requestcontroller.ClientCertificatesKeyvaultMode:
			var err error
			secretProvider, err = getKeyvaultClientFromMSI(context.TODO(), kubeconfigInfo.ClientCertificatesKeyvaultConfig.KeyvaultName,
				kubeconfigInfo.ClientCertificatesKeyvaultConfig.MSIResourceID, kubeconfigInfo.ClientCertificatesKeyvaultConfig.AzureEnvironment)
			if err != nil {
				return fmt.Errorf("error getting keyvault client from MSI: %w", err)
			}
		case requestcontroller.ClientCertificatesFilesystemMode:
			secretProvider = &filesystem.SecretProvider{
				SecretFilenames: config.KubernetesConfig.KubeconfigInfo.ClientCertificatesFromFileConfig,
			}
		default:
			return fmt.Errorf("invalid secret provider: %s", mode)
		}

		//datastoreshim has a dependency on loggershim, and loggershim has a global that needs to be initialized
		loggershimMutex.Lock()
		if loggershim.LS == nil {
			// loggershim.LS hasn't been initialized elsewhere, so do it here
			loggershim.LS = simplejson.NewLogger("datastoreshim")
		}
		loggershimMutex.Unlock()

		ds, err := datastoreshim.NewDataStoreShim(config.DataStoreSettings,
			config.LoggingConfig.ThresholdForLoggingDataStoreOpLatencyMs,
			map[string]string{
				datastore.OrchestratorInstancesConfigurationTableName: datastore.OrchestratorPropertiesPropertyName,
			},
		)
		if err != nil {
			return fmt.Errorf("could not build datastoreshim client: %w", err)
		}

		kubeClusterAuthProviderOpts := datastore.KubeClusterAuthProviderOptions{
			DataStoreClient:  ds,
			SecretProvider:   secretProvider,
			Logger:           logger,
			AzureEnvironment: kubeconfigInfo.AzureEnvironment,
			ManagedDNCGUID:   kubeconfigInfo.ManagedDNCGUID,
			TokenProvider: aad.NewTokenProvider(aad.TokenProviderOptions{
				Logger:        logger,
				MetricsClient: metricsClient,
			}),
			MetricsClient: metricsClient,
		}

		kcp, err := datastore.NewKubeClusterAuthProvider(kubeClusterAuthProviderOpts)
		if err != nil {
			return fmt.Errorf("error getting kubeconfig provider: %w", err)
		}
		kubeClusterAuthProvider = kcp
	default:
		return fmt.Errorf("error creating kubeconfig provider from config mode: %s is not a valid mode for config.kubernetesConfig.kubeconfigInfo.mode", mode)
	}

	// Create the DNC client
	dncClientPollingInterval, err := time.ParseDuration(config.DNCClientConfig.PollingInterval)
	if err != nil {
		return fmt.Errorf("could not parse config.DNCClientConfig.PollingInterval (%s) as duration: %w", config.DNCClientConfig.PollingInterval, err)
	}

	syncDncClient := syncdncclient.NewSynchronousDNCClient(dncclient.NewDncClient(config.DNCClientConfig.DNCEndpoint), dncClientPollingInterval, logger)

	// Create the subnet info provider
	var subnetInfoProvider requestcontroller.SubnetInfoProvider

	switch mode := config.SubnetInfoProviderConfig.Mode; mode {
	case requestcontroller.SubnetInfoProviderDatastoreMode:
		subnetInfoProvider, err = datastore.NewSubnetInfoProvider(datastore.SubnetInfoProviderOptions{
			DataStoreSettings: config.DataStoreSettings,
			LoggingConfig:     config.LoggingConfig,
			MetricsClient:     metricsClient,
		})
		if err != nil {
			return fmt.Errorf("could not create subnet info provider: %w", err)
		}
	case requestcontroller.SubnetInfoProviderConfigMode:
		subnetInfoProvider = &requestcontroller.ConfigSubnetInfoProvider{
			Config: config.SubnetInfoProviderConfig,
		}
	default:
		return fmt.Errorf("unknown SubnetInfoProvider mode: %s", mode)
	}

	logger.WithFields(logrus.Fields{
		"subnetInfoProviderMode": config.SubnetInfoProviderConfig.Mode,
	}).Infof("Using subnetInfoProviderMode %s", config.SubnetInfoProviderConfig.Mode)

	// Create the RequestController
	requestController := requestcontroller.NewRequestController(requestcontroller.RequestControllerOptions{
		KubeClusterAuthProvider: kubeClusterAuthProvider,
		ManagerBuilder: &kubernetes.ClusterManagerBuilder{
			Config:             config,
			SyncDNCClient:      syncDncClient,
			Logger:             logger,
			SubnetInfoProvider: subnetInfoProvider,
			MetricsClient:      metricsClient,
		},
		Logger:        logger,
		Config:        config,
		MetricsClient: metricsClient,
	})

	if err != nil {
		return fmt.Errorf("error creating new RequestController: %w", err)
	}

	// Start the RequestController - this function blocks
	if err := requestController.Start(ctx); err != nil {
		if !errors.Is(err, context.Canceled) {
			// return any non-cancellation errors
			return err
		}
	}

	return nil
}

func getCosmosKeyFromMSI(ctx context.Context, msiResourceID, cosmosResourceID, azureEnvironment string) (string, error) {
	tokenFetcher := &ad.MSITokenFetcher{ResourceID: msiResourceID}
	azureEnv, err := azure.EnvironmentFromName(azureEnvironment)
	if err != nil {
		return "", err
	}

	armEndpoint := azureEnv.ResourceManagerEndpoint

	spt, err := tokenFetcher.GetServicePrincipalToken(armEndpoint)
	if err != nil {
		return "", err
	}

	token, err := ad.GetFreshToken(ctx, spt)
	if err != nil {
		return "", err
	}

	keyFetcher := ad.CosmosKeyFetcher{ARMEndpoint: armEndpoint}
	keys, err := keyFetcher.GetReadWriteKeys(ctx, token, cosmosResourceID)
	if err != nil {
		return "", err
	}

	// todo: rotation still needs to be figured out
	return keys.PrimaryMasterKey, nil
}

func getKeyvaultClientFromMSI(ctx context.Context, keyvaultName, msiResourceID, azureEnvironmentName string) (*ad.KeyvaultClient, error) {
	tokenFetcher := &ad.MSITokenFetcher{ResourceID: msiResourceID}
	azureEnvironment, err := azure.EnvironmentFromName(azureEnvironmentName)
	if err != nil {
		return nil, fmt.Errorf("error getting AzureEnvironment from string %s: %w", azureEnvironment, err)
	}

	keyvaultClient, err := ad.NewKeyvaultClient(tokenFetcher, azureEnvironment, keyvaultName)
	if err != nil {
		return nil, fmt.Errorf("error getting keyvault client: %w", err)
	}

	return keyvaultClient, nil
}

func getConfig(configFile string) (requestcontroller.Config, error) {
	// Start with a default config
	config := requestcontroller.DefaultConfig()

	// Apply config from file if necessary
	if configFile != "" {
		jsonConfigProvider := json.ConfigProvider{
			ConfigFile: configFile,
		}

		if err := jsonConfigProvider.ApplyConfig(&config); err != nil {
			return requestcontroller.Config{}, fmt.Errorf("error reading the config file %s: %w", configFile, err)
		}
	}

	// Apply config from the environment
	envConfigProvider := env.ConfigProvider{}
	if err := envConfigProvider.ApplyConfig(&config); err != nil {
		return requestcontroller.Config{}, fmt.Errorf("error getting config from environment: %w", err)
	}

	return config, nil
}
