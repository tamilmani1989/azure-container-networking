// Copyright 2019 Microsoft. All rights reserved.

package main

import (
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

func init() {
	flag.BoolVar(&stderr, "stderr", false, "Set to true to write logs to stdout.")
	flag.Parse()

	configpath, found := os.LookupEnv("DNC_CONFIGURATION_PATH")
	if !found {
		configpath = "config.json"
	}

	config = configuration.Load(configpath)
	setDncConfigDefaults()
	disableTelemetry = config.DNCSettings.DisableTelemetry
	logger = loggershim.NewLoggerShim(config)
	if disableTelemetry {
		reports = nil
	}
	logger.Init(stderr, reports)
	replicaInstance = replica.NewReplica(
		config.ServerSettings,
		GetNetworkRefreshMonitorStatus)
	sessionManager = utils.NewSessionManager([]string{datamodel.NodeRepositoryName, datamodel.NetworkRepositoryName, datamodel.TagRepositoryName})
	masterElectionShim = masterelectionshim.NewMasterElectionShim(
		config,
		replicaInstance.UpdateReplicaStatus,
		replicaInstance.SetPrimaryIPEndpoint,
		startChan,
		doneChan,
		demoteChan)

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
		ServerSettings: config.ServerSettings,
		Controller:     ctrler}
	dncServer.Init()
	requestContextRetryInterval = time.Duration(config.DNCSettings.RequestRetryBaseInterval) * time.Millisecond
	maxRequestBackoffInterval = time.Duration(config.DNCSettings.MaxRequestBackoffInterval) * time.Millisecond
	sleepIntervalBetweenNetworkBatches = time.Duration(config.DNCSettings.SleepIntervalBetweenNetworkBatches) * time.Millisecond
	sleepIntervalBetweenNetworkRefresh = time.Duration(config.DNCSettings.SleepIntervalBetweenNetworkRefresh) * time.Millisecond
	disableTelemetry = config.DNCSettings.DisableTelemetry

	if config.DataStoreSettings != (configuration.DataStoreSettings{}) {
		masterElectionShim.Init()
		datamodel.Init(config.CNSSettings, config.DataStoreSettings, config.DNCSettings.RequestRetryBaseInterval,
			config.VlanSettings.Min, config.VlanSettings.Max, httpClient)
		ctrler.UpdateReplicaStatus(true)
	}
}

func setDncConfigDefaults() {
	if config.DNCSettings.RequestRetryBaseInterval == 0 {
		// Set default to 1 sec.
		config.DNCSettings.RequestRetryBaseInterval = 1000
	}

	if config.DNCSettings.ConcurrentRequestsToProcess == 0 {
		// Set default to 5
		config.DNCSettings.ConcurrentRequestsToProcess = 5
	}

	if config.DNCSettings.MaxRequestBackoffInterval == 0 {
		// Set default to 16 secs
		config.DNCSettings.MaxRequestBackoffInterval = 16000
	}

	if config.DNCSettings.NetworkRefreshInterval == 0 {
		// Set default to 1 min.
		config.DNCSettings.NetworkRefreshInterval = 60000
	}

	if config.DNCSettings.TagRefreshInterval == 0 {
		// Set default to 1 min.
		config.DNCSettings.TagRefreshInterval = 60000
	}

	if config.DNCSettings.NetworkContainerRefreshInterval == 0 {
		// Set default to 1 min
		config.DNCSettings.NetworkContainerRefreshInterval = 60000
	}

	if config.DNCSettings.NetworkRefreshBatchSize == 0 {
		// Set default to 60
		config.DNCSettings.NetworkRefreshBatchSize = 60
	}

	if config.DNCSettings.SleepIntervalBetweenNetworkBatches == 0 {
		// Set default to 1 min
		config.DNCSettings.SleepIntervalBetweenNetworkBatches = 60000
	}

	if config.DNCSettings.ReportToHostInterval < 60000 {
		// Set default to 1 min
		config.DNCSettings.ReportToHostInterval = 60000
	}

	if config.DNCSettings.SleepIntervalBetweenNetworkRefresh == 0 {
		// Set default to 1 min
		config.DNCSettings.SleepIntervalBetweenNetworkRefresh = 300
	}

	if config.HttpClientSettings.ConnectionTimeout == 0 {
		// set the default connection timeout to 5 seconds
		config.HttpClientSettings.ConnectionTimeout = 5
	}

	if config.HttpClientSettings.ResponseHeaderTimeout == 0 {
		// set the default response header timeout to 120 seconds
		config.HttpClientSettings.ResponseHeaderTimeout = 120
	}
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
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go ProcessExitMonitor(sigs, processExitCallback)
	go startReplica(ctrler)
	dncServer.Start()
}
