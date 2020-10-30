// Copyright 2020 Microsoft. All rights reserved.

package kubernetes

import (
	"context"
	"fmt"
	"log"
	"time"

	ad "dnc/activedirectory"
	dncclient "dnc/client"
	"dnc/requestcontroller"
	"dnc/requestcontroller/datastore"
	syncdncclient "dnc/requestcontroller/dncclient"

	acnv1alpha "github.com/Azure/azure-container-networking/nodenetworkconfig/api/v1alpha"
	"github.com/Azure/go-autorest/autorest/azure"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

// NodeFinalizerName is the finalizer that is placed on Nodes
const (
	NNCFinalizerName                        = "finalizers.acn.azure.com/dnc-operations"
	DelegatedSubnetIDLabel                  = "kubernetes.azure.com/podnetwork-delegationguid"
	SubnetNameLabel                         = "kubernetes.azure.com/podnetwork-subnet"
	leaderElectionConfigmapSuffix           = ".resourcelock.acn.azure.com"
	kubeconfigRefreshIntervalSeconds        = 300
	kubeconfigRefreshOnErrorIntervalSeconds = 30
)

// RequestController implements requestcontroller.RequestController. RequestController watches Nodes and NetworkConfigs
// and calls DNC functions as appropriate.
type RequestController struct {
	managerPool        *ManagerPool
	kubeconfigProvider requestcontroller.KubeconfigProvider
}

var _ requestcontroller.RequestController = &RequestController{}

// K8sClient is an interface that wraps only the functions of controller-runtime/pkg/client.Client that the reconcilers use
// Unit tests can implement this interface to mock a K8s client
type K8sClient interface {
	Create(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error
	Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error
	Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error
	Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error
	Status() client.StatusWriter
}

// NewRequestControllerWithConfigProvider creates a new RequestController from the given ConfigProvider
func NewRequestControllerWithConfigProvider(cp requestcontroller.ConfigProvider) (*RequestController, error) {
	config, err := cp.GetConfig()
	if err != nil {
		return nil, err
	}

	// Add the CRD to the scheme
	scheme := runtime.NewScheme()
	err = clientgoscheme.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}

	err = acnv1alpha.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}

	// Get the Cosmos token via MSI if necessary
	if config.DataStoreSettings.AzureStorageInfo.IsUsingCosmosDb && config.DataStoreSettings.AzureStorageInfo.Key == "" {
		log.Printf("Getting key for CosmosDB with MSI")
		key, err := getCosmosKeyFromMSI(context.TODO(), config.CosmosMSISettings.MSIResourceID, config.CosmosMSISettings.CosmosResourceID, config.CosmosMSISettings.AzureEnvironment)
		if err != nil {
			return nil, err
		}

		config.DataStoreSettings.AzureStorageInfo.Key = key
	}

	// Choose a kubeconfig provider
	var kubeconfigProvider requestcontroller.KubeconfigProvider
	kubeconfigInfo := config.KubernetesConfig.KubeconfigInfo //later code references properties a lot, so make future line shorter
	log.Printf("Creating kubeconfig provider for config mode %s", kubeconfigInfo.Mode)

	switch mode := kubeconfigInfo.Mode; mode {
	case requestcontroller.KubeconfigFileMode:
		kubeconfigProvider = requestcontroller.KubeconfigFromFileProvider{kubeconfigInfo.File}
	case requestcontroller.KubeconfigDatastoreMode:
		// Datastore mode requires secret from keyvault, so build keyvault client
		keyvaultClient, err := getKeyvaultClientFromMSI(context.TODO(), kubeconfigInfo.ClientCertificatesKeyvaultConfig.KeyvaultName,
			kubeconfigInfo.ClientCertificatesKeyvaultConfig.MSIResourceID, kubeconfigInfo.ClientCertificatesKeyvaultConfig.AzureEnvironment)
		if err != nil {
			return nil, err
		}

		kubeconfigProvider = datastore.NewKubeconfigProvider(kubeconfigInfo.ManagedDNCGUID, config.DataStoreSettings, config.LoggingConfig,
			keyvaultClient, kubeconfigInfo.ClientCertificatesDirectory, kubeconfigInfo.KubeloginBinaryPath, kubeconfigInfo.AzureEnvironment)
	default:
		return nil, fmt.Errorf("Error creating kubeconfig provider from config mode: %s is not a valid mode for config.kubernetesConfig.kubeconfigInfo.mode", mode)
	}

	if err != nil {
		return nil, err
	}

	dncClientPollingInterval, err := time.ParseDuration(config.DNCClientConfig.PollingInterval)
	if err != nil {
		return nil, err
	}

	syncDncClient := syncdncclient.NewSynchronousDNCClient(dncclient.NewDncClient(config.DNCClientConfig.DNCEndpoint), dncClientPollingInterval)

	subnetInfoProvider := datastore.NewSubnetInfoProvider(config.DataStoreSettings, config.LoggingConfig)

	requestController := RequestController{
		managerPool: &ManagerPool{
			managers:           make(map[string]*manager),
			scheme:             scheme,
			config:             config,
			subnetInfoProvider: subnetInfoProvider,
			syncDNCClient:      syncDncClient,
		},
		kubeconfigProvider: kubeconfigProvider,
	}

	return &requestController, nil
}

// Start starts the RequestController and blocks until there is an error or exitChan is closed
func (r *RequestController) Start() {
	exitChan := signals.SetupSignalHandler()

	// Register all managers
	ctx, cancel := context.WithCancel(context.Background())
	go r.registerAndStartManagersForever(ctx)

	//block until exitChan is closed
	select {
	case <-exitChan:
		log.Printf("Stopping all managers")
		cancel()
		r.managerPool.UnregisterAll()
	}
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

func (r *RequestController) registerAndStartManagers(ctx context.Context) error {
	kubeconfigsBytes, err := r.kubeconfigProvider.GetBytesForAll(ctx)
	if err != nil {
		return fmt.Errorf("Error getting kubeconfigs: %w", err)
	}

	// Register the orchestrators with the manager pool
	r.managerPool.RegisterManagers(kubeconfigsBytes)

	// Start all the managers (in case there are new ones)
	r.managerPool.StartAll()

	return nil
}

func (r *RequestController) registerAndStartManagersForever(ctx context.Context) {
	err := r.registerAndStartManagers(ctx)
	if err != nil {
		log.Printf("Failed to get k8s clients: %v, will retry in 30s", err)
	}
	for {
		select {
		case <-ctx.Done():
			log.Printf("register managers loop was cancelled, no longer checking for new k8s clusters")
			return
		case <-time.After(30 * time.Second):
			log.Printf("Refreshing k8s clients from kubeconfig provider")
		}
		err := r.registerAndStartManagers(ctx)
		if err != nil {
			log.Printf("Failed to get k8s clients: %v, will retry in 30s", err)
		}
	}
}
