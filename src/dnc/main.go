// Copyright 2019 Microsoft. All rights reserved.

package main

import (
	"context"
	"crypto/tls"
	ad "dnc/activedirectory"
	"dnc/aiwrapper"
	"dnc/cns"
	"dnc/cnsinterface"
	"dnc/configuration"
	"dnc/controller"
	"dnc/datamodel"
	monitor "dnc/internal/resourcecleanupmonitor"
	"dnc/logger/loggershim"
	"dnc/masterelection/masterelectionshim"
	"dnc/pubsubproxy"
	"dnc/replica"
	"dnc/resourceupdatehandler/networkupdatehandler"
	"dnc/server"
	"dnc/utils"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Azure/azure-container-networking/aitelemetry"
	acncommon "github.com/Azure/azure-container-networking/common"
	"github.com/Azure/azure-container-networking/telemetry"
	"github.com/Azure/go-autorest/autorest/azure"
)

var (
	cancelProcessing        = make(chan bool)
	startChan               = make(chan bool)
	doneChan                = make(chan bool)
	demoteChan              = make(chan bool)
	refreshNCStopProcessing = make(chan bool)
	telemetryStopProcessing = make(chan bool)
	reports                 = make(chan telemetry.DNCReport)

	// todo: remove globals. inject as dependencies to types that need them.
	config             *configuration.Configuration
	masterElectionShim *masterelectionshim.MasterElectionShim
	cnsInterface       cnsinterface.CnsInterface
	logger             *loggershim.LoggerShim
	sessionManager     *utils.SessionManager
	ctrler             *controller.Controller
	dncServer          server.Server
	replicaInstance    *replica.Replica
	stderr             bool

	//  Base retry interval
	requestContextRetryInterval time.Duration

	//  Max request process backoff interval
	maxRequestBackoffInterval time.Duration

	// Flag to disable telemetry
	disableTelemetry bool

	// HTTP client object to be reused wherever needed
	httpClient *http.Client

	// HTTP client object to be used by to call cns
	cnsHttpClient *http.Client
)

const (
	maxLogFileSize            = 16 * 1024 * 1024 // Set 16MB as the max log file size
	maxLogFileCount           = 8                // Set 8 as max log file count.
	maxKeyvaultRetrievalTime  = 10
	tempPubSubCertFilePattern = "pubsubcert-*.pem"
)

func printVersion() {
	fmt.Printf("Azure Delegated Network Service\n")
	fmt.Printf("Version %v\n", utils.Version)
}

func main() {
	acncommon.ParseArgs(&args, printVersion)
	vers := acncommon.GetArg(OptVersion).(bool)
	if vers {
		printVersion()
		os.Exit(0)
	}

	stderr = acncommon.GetArg(OptLogTargetStderr).(bool)
	configpath, found := os.LookupEnv("DNC_CONFIGURATION_PATH")
	if !found {
		configpath = "config.json"
	}

	config = configuration.Load(configpath)
	configuration.OverrideConfigFromEnv(config)
	configuration.SetDncConfigDefaults(config)
	disableTelemetry = config.TelemetrySettings.DisableTelemetry

	var aiTelemetryErr error

	if disableTelemetry {
		reports = nil
	} else {
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

		aiTelemetryErr = aiwrapper.CreateTelemetryHandle(aiConfig)
	}

	logger = loggershim.NewLoggerShim(config)
	logger.Init(stderr, reports)
	logger.SetLogFileLimits(maxLogFileSize, maxLogFileCount)

	if aiTelemetryErr != nil {
		logger.Errorf("Error initializing AI telemetry handle. %v", aiTelemetryErr)
	} else {
		logger.Debugf("Successfully initialized AI telemetry.")
	}

	env, err := azure.EnvironmentFromName(config.AzureEnvironment.EnvironmentName)
	if err != nil {
		log.Fatalf("Invalid azure environment %s", config.AzureEnvironment.EnvironmentName)
	}

	tokenFetcher, err := getTokenFetcher()
	if err != nil {
		log.Fatalln("Error getting token fetcher. Error:", err)
	}

	ctx := context.Background()
	if config.DataStoreSettings.AzureStorageInfo.CosmosDBResourceID != "" {
		spt, err := tokenFetcher.GetServicePrincipalToken(env.ResourceManagerEndpoint)
		if err != nil {
			log.Fatalln("Error getting service principal token. Error:", err)
		}

		token, err := ad.GetFreshToken(ctx, spt)
		if err != nil {
			log.Fatalln("Error getting ARM token. Error:", err)
		}

		// todo: discuss with team: what should be our workflow for key rotation?
		keyFetcher := ad.CosmosKeyFetcher{ARMEndpoint: env.ResourceManagerEndpoint}
		keys, err := keyFetcher.GetReadWriteKeys(ctx, token, config.DataStoreSettings.AzureStorageInfo.CosmosDBResourceID)
		if err != nil {
			log.Fatalln("Error getting Cosmos RW Keys. Error:", err)
		}

		// todo: other attributes such as account name can be extracted from the resource id and azure environment
		config.DataStoreSettings.AzureStorageInfo.Key = keys.PrimaryMasterKey
	}

	replicaInstance = replica.NewReplica(
		config.ServerSettings,
		config.HttpClientSettings,
		networkupdatehandler.GetNetworkRefreshMonitorStatus)
	sessionManager = utils.NewSessionManager([]string{
		datamodel.NCRepositoryName,
		datamodel.NodeRepositoryName,
		datamodel.NetworkRepositoryName,
		datamodel.ReservationRepositoryName,
		datamodel.TagRepositoryName})
	masterElectionShim = masterelectionshim.NewMasterElectionShim(
		config,
		replicaInstance.UpdateReplicaStatus,
		replicaInstance.SetPrimaryIPEndpoint,
		startChan,
		doneChan,
		demoteChan)

	TlsClientConfig := &tls.Config{ServerName: config.CNSSettings.CnsTlsCertificateSubjectName}

	// Creating a cns client that uses the tls servername to avoid hostname mismatch
	cnsHttpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: TlsClientConfig,
			DialContext: (&net.Dialer{
				Timeout: time.Duration(config.HttpClientSettings.ConnectionTimeout) * time.Second,
			}).DialContext,
			ResponseHeaderTimeout: time.Duration(config.HttpClientSettings.ResponseHeaderTimeout) * time.Second,
		},
	}

	// Creating http client object to be reused instead of creating one every time.
	// This helps make use of the cached tcp connections.
	// Clients are safe for concurrent use by multiple goroutines.
	httpClient = &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: time.Duration(config.HttpClientSettings.ConnectionTimeout) * time.Second,
			}).DialContext,
			ResponseHeaderTimeout: time.Duration(config.HttpClientSettings.ResponseHeaderTimeout) * time.Second,
		},
	}

	switch strings.ToLower(config.CNSSettings.ChannelMode) {
	case cnsinterface.Direct:
		cnsInterface = &cnsinterface.DirectChannel{
			SessionManager:  sessionManager,
			EnforceNicLimit: config.DNCSettings.EnforceNICLimit,
			CNSClient: &cns.Client{
				Paths: config.CNSSettings.ConnectionStrings,
				HTTP:  cnsHttpClient,
			},
			Logger: logger,
		}
	case cnsinterface.Managed:
		cnsInterface = cnsinterface.NewManagedChannel(sessionManager, config.DNCSettings.EnforceNICLimit)
	case cnsinterface.Disabled:
		cnsInterface = &cnsinterface.DisabledChannel{NCLimit: 1}
	default:
		logger.Errorf("did not set a valid CNS channel: '%s'", config.CNSSettings.ChannelMode)
		os.Exit(1)
	}

	// Initialize the datamodel before initializing controller
	if config.DataStoreSettings != (configuration.DataStoreSettings{}) {
		masterElectionShim.Init()
		// TO DO : Pass config object itself.
		err := datamodel.Init(
			config.CNSSettings,
			config.DataStoreSettings,
			config.DNCSettings,
			config.HostAgentSettings,
			config.VlanSettings.Min,
			config.VlanSettings.Max,
			config.GreSettings.Min,
			config.GreSettings.Min,
			httpClient,
			cnsHttpClient,
			config.HostAgentSettings.WireServerIP)
		if err != nil {
			log.Fatalln("Error initializing datamodel. Error:", err)
		}
	}

	networkupdatehandler.Init(config, replicaInstance, sessionManager, httpClient, &ipAllocator, logger)
	initNetworkContainerStateMachine()

	ctrler = controller.NewController(replicaInstance, sessionManager, config.CNSSettings.ConnectionStrings.SetOrchestratorTypeURLFmt,
		masterElectionShim, config, httpClient, cnsHttpClient, &ipAllocator)
	dncServer = server.Server{
		ServerSettings:      config.ServerSettings,
		Controller:          ctrler,
		ReverseProxyEnabled: config.DNCSettings.ReverseProxyEnabled,
		ConnectionTimeout:   config.HttpClientSettings.ConnectionTimeout,
		Logger:              logger}

	if err := dncServer.Init(); err != nil {
		log.Fatalln("Error initializing dnc server. Error:", err)
	}

	requestContextRetryInterval = time.Duration(config.DNCSettings.RequestRetryBaseInterval) * time.Millisecond
	maxRequestBackoffInterval = time.Duration(config.DNCSettings.MaxRequestBackoffInterval) * time.Millisecond

	if config.DataStoreSettings != (configuration.DataStoreSettings{}) {
		ctrler.UpdateReplicaStatus(true)
		setOrchestratorForTelemetry()
	}

	pubSubCfg := config.PubSubProxySettings
	if pubSubCfg.EnablePubSubProxy {
		certPath := pubSubCfg.PubsubCertFilePath
		if certPath == "" {
			akv, err := ad.NewKeyvaultClient(tokenFetcher, env, config.PubSubProxySettings.PubsubCertKeyvaultName)
			if err != nil {
				log.Fatalln("Error initializing keyvault client. Error:", err)
			}

			path, err := getCertPath(ctx, akv, pubSubCfg.PubsubCertName)
			if err != nil {
				log.Fatalln("Error writing temporary cert file. Error:", err)
			}
			certPath = path
		}

		if err := pubsubproxy.Init(pubSubCfg, replicaInstance, certPath); err != nil {
			log.Fatalln("Error initializing PubsubProxy. Error:", err)
		}

		resourceCleanupMonitor := monitor.ResourceCleanupMonitor{
			IsPrimary:          true, // todo: make this dynamic
			IpAllocator:        &ipAllocator,
			Logger:             logger,
			PendingCleanupRepo: datamodel.PendingCleanupRepo,
		}

		go monitor.StartCleanupMonitor(&resourceCleanupMonitor, ctx)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go ProcessExitMonitor(sigs, processExitCallback)
	go startReplica(ctrler)
	dncServer.Start()
}

func getTokenFetcher() (ad.TokenFetcher, error) {
	switch {
	case usePodIdentityThroughNMI():
		return ad.NewPodIdentityManager()
	case config.IdentitySettings.MSIResourceID != "":
		return &ad.MSITokenFetcher{ResourceID: config.IdentitySettings.MSIResourceID}, nil
	default:
		return new(ad.MSITokenFetcher), nil
	}
}

func usePodIdentityThroughNMI() bool {
	podNameEnv := os.Getenv(ad.AADPodNameEnv)
	podNSEnv := os.Getenv(ad.AADPodNamespaceEnv)
	return podNameEnv != "" && podNSEnv != ""
}

// getCertPath fetches a certificate from keyvault, writes it to a temporary file, and returns its file path
func getCertPath(ctx context.Context, kvc *ad.KeyvaultClient, certName string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, maxKeyvaultRetrievalTime*time.Second)
	defer cancel()

	bs, err := kvc.GetLatestSecretPEM(ctx, certName)
	if err != nil {
		return "", fmt.Errorf("failed to fetch keyvault certificate: %v", err)
	}

	return utils.ToTempFile(tempPubSubCertFilePattern, bs)
}

func processExitCallback() {
	if ctrler.IsReplicaReady() {
		masterElectionShim.Cleanup()
	}

	// If using test persistence store, dump state
	if config.DataStoreSettings.TestPersistenceStoreInfo.Test != "" {
		datamodel.DumpState()
	}

	cancelProcessing <- true

	if config.PubSubProxySettings.EnablePubSubProxy {
		pubsubproxy.Deinit()
	}
}

func startReplica(c *controller.Controller) {
	ticker := time.NewTicker(time.Millisecond * 1000)
	// Wait for orchestrator to send azure storage information
	if !c.IsReplicaReady() {
		go masterElectionShim.Start()

		for {
			<-ticker.C
			if c.IsReplicaReady() {
				break
			}
		}
	}

	ticker.Stop()
	go masterElectionShim.Start()
	go monitorRequests()

	// Network refresh monitor is not needed if PubSubProxy is enabled since Pubsubproxy
	// handles the network updates using pubsub subscribe mechanism for the networks
	if !config.PubSubProxySettings.EnablePubSubProxy {
		go networkupdatehandler.MonitorNetworkUpdates()
	}

	// NC refresh is not required in managed instances
	if !config.DNCSettings.Managed {
		go monitorNetworkContainers()
	}

	if !disableTelemetry {
		go sendTelemetry()
	}
}
