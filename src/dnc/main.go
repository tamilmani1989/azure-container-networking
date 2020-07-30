// Copyright 2019 Microsoft. All rights reserved.

package main

import (
	"dnc/aiwrapper"
	"dnc/cnsinterface"
	"dnc/configuration"
	"dnc/controller"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/masterelection/masterelectionshim"
	"dnc/pubsubproxy"
	"dnc/replica"
	"dnc/resourceupdatehandler/networkupdatehandler"
	"dnc/server"
	"dnc/utils"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Azure/azure-container-networking/aitelemetry"
	"github.com/Azure/azure-container-networking/cns"
	"github.com/Azure/azure-container-networking/telemetry"
)

var (
	isPrimary                struct{ flag int32 }
	cancelProcessing         = make(chan bool)
	startProcessing          = make(chan bool)
	startChan                = make(chan bool)
	doneChan                 = make(chan bool)
	demoteChan               = make(chan bool)
	refreshNCStartProcessing = make(chan bool)
	refreshNCStopProcessing  = make(chan bool)
	telemetryStopProcessing  = make(chan bool)
	reports                  = make(chan telemetry.DNCReport)

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
)

const (
	maxLogFileSize  = 16 * 1024 * 1024 // Set 16MB as the max log file size
	maxLogFileCount = 8                // Set 8 as max log file count.
)

func init() {
	configpath, found := os.LookupEnv("DNC_CONFIGURATION_PATH")
	if !found {
		configpath = "config.json"
	}

	config = configuration.Load(configpath)
	configuration.OverrideConfigFromEnv(config)
	configuration.SetDncConfigDefaults(config)
	disableTelemetry = config.TelemetrySettings.DisableTelemetry
	logger = loggershim.NewLoggerShim(config)

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

	logger.Init(stderr, reports)
	logger.SetLogFileLimits(maxLogFileSize, maxLogFileCount)

	if aiTelemetryErr != nil {
		logger.Errorf("Error initializing AI telemetry handle. %v", aiTelemetryErr)
	} else {
		logger.Debugf("Successfully initialized AI telemetry.")
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
	switch config.CNSSettings.ChannelMode {
	case cns.Direct:
		cnsInterface = cnsinterface.NewDirectClient(sessionManager, config.DNCSettings.EnforceNICLimit)
	case cns.Managed:
		cnsInterface = cnsinterface.NewLocalClient(sessionManager, config.DNCSettings.EnforceNICLimit)
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

	ctrler = controller.NewController(replicaInstance, sessionManager, config.CNSSettings.ConnectionStrings.SetOrchestratorTypeURLFmt,
		masterElectionShim, config, httpClient)
	dncServer = server.Server{
		ServerSettings:      config.ServerSettings,
		Controller:          ctrler,
		ReverseProxyEnabled: config.DNCSettings.ReverseProxyEnabled,
		ConnectionTimeout:   config.HttpClientSettings.ConnectionTimeout}
	dncServer.Init()
	requestContextRetryInterval = time.Duration(config.DNCSettings.RequestRetryBaseInterval) * time.Millisecond
	maxRequestBackoffInterval = time.Duration(config.DNCSettings.MaxRequestBackoffInterval) * time.Millisecond

	if config.DataStoreSettings != (configuration.DataStoreSettings{}) {
		masterElectionShim.Init()
		// TO DO : Pass config object itself.
		datamodel.Init(
			config.CNSSettings,
			config.DataStoreSettings,
			config.DNCSettings,
			config.VlanSettings.Min,
			config.VlanSettings.Max,
			httpClient,
			config.HostAgentSettings.WireServerIP)
		ctrler.UpdateReplicaStatus(true)
		setOrchestratorForTelemetry()
	}

	networkupdatehandler.Init(config, replicaInstance, sessionManager, httpClient, logger)

	if config.PubSubProxySettings.EnablePubSubProxy {
		if err := pubsubproxy.Init(config.PubSubProxySettings, replicaInstance); err != nil {
			log.Fatalln("Error initializing PubsubProxy. Error:", err)
		}
	}

	initNetworkContainerStateMachine()
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

func main() {
	// From go 1.13, we can't use flags api in package init as it conflicts with go test init.
	flag.BoolVar(&stderr, "stderr", false, "Set to true to write logs to stdout.")
	flag.Parse()
	logger.Init(stderr, reports)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go ProcessExitMonitor(sigs, processExitCallback)
	go startReplica(ctrler)
	dncServer.Start()
}
