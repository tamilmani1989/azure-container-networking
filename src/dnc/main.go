// Copyright 2019 Microsoft. All rights reserved.

package main

import (
	"dnc/aiwrapper"
	"dnc/configuration"
	"dnc/controller"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/masterelection/masterelectionshim"
	"dnc/replica"
	"dnc/server"
	"dnc/utils"
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Azure/azure-container-networking/aitelemetry"
	"github.com/Azure/azure-container-networking/telemetry"
)

var (
	isPrimary                     struct{ flag int32 }
	cancelProcessing              = make(chan bool)
	startProcessing               = make(chan bool)
	startChan                     = make(chan bool)
	doneChan                      = make(chan bool)
	demoteChan                    = make(chan bool)
	networkMonitorStartProcessing = make(chan bool)
	tagMonitorStartProcessing     = make(chan bool)
	networkMonitorStopProcessing  = make(chan bool)
	tagMonitorStopProcessing      = make(chan bool)
	refreshNCStartProcessing      = make(chan bool)
	refreshNCStopProcessing       = make(chan bool)
	telemetryStopProcessing       = make(chan bool)
	reports                       = make(chan telemetry.DNCReport)

	config             *configuration.Configuration
	masterElectionShim *masterelectionshim.MasterElectionShim
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

	// SleepIntervalBetweenBatches
	sleepIntervalBetweenNetworkBatches time.Duration

	// SleepIntervalBetweenNetworkRefresh
	sleepIntervalBetweenNetworkRefresh time.Duration

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

	if disableTelemetry {
		reports = nil
	} else {
		aiConfig := aitelemetry.AIConfig{
			BatchSize:                    config.TelemetrySettings.TelemetryBatchSizeBytes,
			BatchInterval:                config.TelemetrySettings.TelemetryBatchIntervalInSecs,
			RefreshTimeout:               config.TelemetrySettings.RefreshIntervalInSecs,
			DisableMetadataRefreshThread: config.TelemetrySettings.DisableMetadataRefreshThread,
			DebugMode:                    config.TelemetrySettings.DebugMode,
		}

		aiwrapper.CreateTelemetryHandle(aiConfig)
	}

	logger.Init(stderr, reports)
	logger.SetLogFileLimits(maxLogFileSize, maxLogFileCount)
	replicaInstance = replica.NewReplica(
		config.ServerSettings,
		GetNetworkRefreshMonitorStatus)
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
	sleepIntervalBetweenNetworkBatches = time.Duration(config.DNCSettings.SleepIntervalBetweenNetworkBatches) * time.Millisecond
	sleepIntervalBetweenNetworkRefresh = time.Duration(config.DNCSettings.SleepIntervalBetweenNetworkRefresh) * time.Millisecond

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
	}

	setOrchestratorForTelemetry()
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
	go monitorNetworkUpdates()
	go monitorTagUpdates()
	go monitorNetworkContainers()
	if !disableTelemetry {
		go sendTelemetry()
	}
}

func main() {
	// From go 1.13, we can't use flags api in package init as it conflicts with go test init.
	flag.BoolVar(&stderr, "stderr", false, "Set to true to write logs to stdout.")
	flag.Parse()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go ProcessExitMonitor(sigs, processExitCallback)
	go startReplica(ctrler)
	dncServer.Start()
}
