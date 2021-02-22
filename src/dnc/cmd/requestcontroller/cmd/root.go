package cmd

import (
	"context"
	ad "dnc/activedirectory"
	"dnc/aiwrapper"
	dncclient "dnc/client"
	"dnc/requestcontroller"
	"dnc/requestcontroller/buildinfo"
	"dnc/requestcontroller/datastore"
	syncdncclient "dnc/requestcontroller/dncclient"
	"dnc/requestcontroller/json"
	"dnc/requestcontroller/kubernetes"
	"dnc/utils"
	"fmt"
	"time"

	"github.com/Azure/azure-container-networking/aitelemetry"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	shortDesc  = "Watches K8s api-server for SWIFT-based CNI deployment"
	longDesc   = "Watches K8s api-server for Nodes, creates/deletes NodeNetworkConfigs, and responds to NodeNetworkConfig changes"
	configFlag = "config"
)

var (
	rootCmd = &cobra.Command{
		Use:   buildinfo.ExecutableName,
		Short: shortDesc,
		Long:  longDesc,
		Run: func(cmd *cobra.Command, args []string) {
			configFile, err := cmd.Flags().GetString(configFlag)
			if err != nil {
				logrus.Fatal(err, "Error getting config flag value")
			}
			err = runRequestController(configFile)
			if err != nil {
				logrus.Fatalf("Error starting requestcontroller: %v", err)
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
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func runRequestController(configFile string) error {
	logger := logrus.StandardLogger()
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Read the config
	jsonConfigProvider := json.ConfigProvider{
		ConfigFile: configFile,
	}
	config, err := jsonConfigProvider.GetConfig()
	if err != nil {
		return fmt.Errorf("Error reading config: %w", err)
	}

	// Get the Cosmos token via MSI if necessary
	if config.DataStoreSettings.AzureStorageInfo.IsUsingCosmosDb && config.DataStoreSettings.AzureStorageInfo.Key == "" {
		logger.Info("Getting key for CosmosDB with MSI")
		key, err := getCosmosKeyFromMSI(context.TODO(), config.CosmosMSISettings.MSIResourceID, config.CosmosMSISettings.CosmosResourceID, config.CosmosMSISettings.AzureEnvironment)
		if err != nil {
			return fmt.Errorf("Error getting Cosmos key from MSI: %w", err)
		}

		/* Saw a strange error where if the subscription was incorrect in the CosmosResourceID string, the getCosmosKeyFromMSI function would return an empty key and nil error.
		*  The following check ensures that RC doesn't get an empty key and nil error. TODO: We should investigate why the adal functions were not returning errors and failing silently
		 */
		if key == "" {
			return fmt.Errorf("Error getting Cosmos Key from MSI, key was empty")
		}

		config.DataStoreSettings.AzureStorageInfo.Key = key
	}

	// Choose a kubeconfig provider
	var kubeconfigProvider requestcontroller.KubeconfigProvider
	kubeconfigInfo := config.KubernetesConfig.KubeconfigInfo //later code references these properties a lot, so make future lines shorter
	logger.Infof("Creating kubeconfig provider for config mode %s", kubeconfigInfo.Mode)

	switch mode := kubeconfigInfo.Mode; mode {
	case requestcontroller.KubeconfigFileMode:
		kubeconfigProvider = requestcontroller.KubeconfigFromFileProvider{Filename: kubeconfigInfo.File}
	case requestcontroller.KubeconfigDatastoreMode:
		// Datastore mode requires secret from keyvault, so build keyvault client
		keyvaultClient, err := getKeyvaultClientFromMSI(context.TODO(), kubeconfigInfo.ClientCertificatesKeyvaultConfig.KeyvaultName,
			kubeconfigInfo.ClientCertificatesKeyvaultConfig.MSIResourceID, kubeconfigInfo.ClientCertificatesKeyvaultConfig.AzureEnvironment)
		if err != nil {
			return fmt.Errorf("Error getting keyvault client from MSI: %w", err)
		}

		kubeconfigProviderOpts := datastore.KubeconfigProviderOptions{
			DataStoreSettings:           config.DataStoreSettings,
			LoggingConfig:               config.LoggingConfig,
			SecretProvider:              keyvaultClient,
			Logger:                      logger,
			ClientCertificatesDirectory: kubeconfigInfo.ClientCertificatesDirectory,
			KubeloginBinaryPath:         kubeconfigInfo.KubeloginBinaryPath,
			AzureEnvironment:            kubeconfigInfo.AzureEnvironment,
			ManagedDNCGUID:              kubeconfigInfo.ManagedDNCGUID,
		}

		kcp, err := datastore.NewKubeconfigProvider(kubeconfigProviderOpts)
		if err != nil {
			return fmt.Errorf("Error getting kubeconfig provider: %w", err)
		}
		kubeconfigProvider = kcp
	default:
		return fmt.Errorf("Error creating kubeconfig provider from config mode: %s is not a valid mode for config.kubernetesConfig.kubeconfigInfo.mode", mode)
	}

	// Create the DNC client
	dncClientPollingInterval, err := time.ParseDuration(config.DNCClientConfig.PollingInterval)
	if err != nil {
		return fmt.Errorf("Could not parse config.DNCClientConfig.PollingInterval (%s) as duration: %w", config.DNCClientConfig.PollingInterval, err)
	}

	syncDncClient := syncdncclient.NewSynchronousDNCClient(dncclient.NewDncClient(config.DNCClientConfig.DNCEndpoint), dncClientPollingInterval, logger)

	// Create the subnet info provider
	var subnetInfoProvider requestcontroller.SubnetInfoProvider
	switch mode := config.SubnetInfoProviderConfig.Mode; mode {
	case requestcontroller.SubnetInfoProviderDatastoreMode:
		subnetInfoProvider, err = datastore.NewSubnetInfoProvider(config.DataStoreSettings, config.LoggingConfig)
		if err != nil {
			return fmt.Errorf("Could not create subnet info provider: %w", err)
		}
	case requestcontroller.SubnetInfoProviderConfigMode:
		subnetInfoProvider = &requestcontroller.ConfigSubnetInfoProvider{
			Config: config.SubnetInfoProviderConfig,
		}
	default:
		return fmt.Errorf("Unknown SubnetInfoProvider mode: %s", mode)
	}

	logger.WithFields(logrus.Fields{
		"subnetInfoProviderMode": config.SubnetInfoProviderConfig.Mode,
	}).Infof("Using subnetInfoProviderMode %s", config.SubnetInfoProviderConfig.Mode)

	// Create the RequestController
	requestController, err := kubernetes.NewRequestController(kubernetes.RequestControllerOptions{
		KubeconfigProvider: kubeconfigProvider,
		DNCClient:          syncDncClient,
		SubnetInfoProvider: subnetInfoProvider,
		Config:             config,
		Logger:             logger,
	})

	if err != nil {
		return fmt.Errorf("Error creating new RequestController: %w", err)
	}

	// Initialize the global AI Wrapper
	// TODO: Metrics should be an interface instead of global
	if !config.TelemetrySettings.DisableTelemetry {
		aiConfig := aitelemetry.AIConfig{
			BatchSize:                    config.TelemetrySettings.TelemetryBatchSizeBytes,
			BatchInterval:                config.TelemetrySettings.TelemetryBatchIntervalInSecs,
			RefreshTimeout:               config.TelemetrySettings.RefreshIntervalInSecs,
			DisableMetadataRefreshThread: config.TelemetrySettings.DisableMetadataRefreshThread,
			DebugMode:                    config.TelemetrySettings.DebugMode,
			AppVersion:                   utils.Version,
			AppName:                      aiwrapper.AppName,
			GetEnvRetryCount:             config.TelemetrySettings.GetEnvRetryCount,
			GetEnvRetryWaitTimeInSecs:    config.TelemetrySettings.GetEnvRetryWaitTimeInSecs,
		}

		err = aiwrapper.CreateTelemetryHandle(aiConfig)
		if err != nil {
			// Print message to be consistent with dnc, but might be worth exiting with error?
			logger.WithError(err).Error("Error creating AITelemetry handle")
		}
	}

	// Start the RequestController
	requestController.Start() // blocks

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
		return nil, fmt.Errorf("Error getting AzureEnvironment from string %s: %w", azureEnvironment, err)
	}

	keyvaultClient, err := ad.NewKeyvaultClient(tokenFetcher, azureEnvironment, keyvaultName)
	if err != nil {
		return nil, fmt.Errorf("Error getting keyvault client: %w", err)
	}

	return keyvaultClient, nil
}
