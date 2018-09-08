package main

import (
	"dnc/configuration"
	"dnc/controller"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/masterelection/masterelectionshim"
	"dnc/netagentcontracts"
	"dnc/replica"
	"dnc/server"
	"dnc/utils"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var isPrimary struct{ flag int32 }
var cancelProcessing = make(chan bool)
var startProcessing = make(chan bool)
var startChan = make(chan bool)
var doneChan = make(chan bool)
var demoteChan = make(chan bool)
var networkMonitorStartProcessing = make(chan bool)
var tagMonitorStartProcessing = make(chan bool)
var networkMonitorStopProcessing = make(chan bool)
var tagMonitorStopProcessing = make(chan bool)
var refreshNCStartProcessing = make(chan bool)
var refreshNCStopProcessing = make(chan bool)
var reports = make(chan netagentcontracts.DNCReport)

var config *configuration.Configuration
var masterElectionShim *masterelectionshim.MasterElectionShim
var logger *loggershim.LoggerShim
var sessionManager *utils.SessionManager
var ctrler *controller.Controller
var dncServer server.Server
var replicaInstance *replica.Replica
var stderr bool

//  Base retry interval
var requestContextRetryInterval time.Duration

func init() {
	flag.BoolVar(&stderr, "stderr", false, "Set to true to write logs to stdout.")
	flag.Parse()

	configpath, found := os.LookupEnv("DNC_CONFIGURATION_PATH")
	if !found {
		configpath = "config.json"
	}

	config = configuration.Load(configpath)
	logger = loggershim.NewLoggerShim(config)
	logger.Init(stderr, reports)
	replicaInstance = replica.NewReplica(config.ServerSettings)
	sessionManager = utils.NewSessionManager([]string{datamodel.NodeRepositoryName, datamodel.NetworkRepositoryName, datamodel.TagRepositoryName})
	masterElectionShim = masterelectionshim.NewMasterElectionShim(
		config,
		replicaInstance.UpdateReplicaStatus,
		replicaInstance.SetPrimaryIPEndpoint,
		startChan,
		doneChan,
		demoteChan)
	ctrler = controller.NewController(replicaInstance, sessionManager, config.CNSSettings.ConnectionStrings.SetOrchestratorTypeURLFmt, masterElectionShim, config)
	dncServer = server.Server{
		ServerSettings: config.ServerSettings,
		Controller:     ctrler}
	dncServer.Init()
	requestContextRetryInterval = time.Duration(config.DNCSettings.RequestRetryBaseInterval) * time.Millisecond

	if config.DataStoreSettings != (configuration.DataStoreSettings{}) {
		masterElectionShim.Init()
		datamodel.Init(config.DataStoreSettings, config.DNCSettings.RequestRetryBaseInterval, config.VlanSettings.Min, config.VlanSettings.Max)
		ctrler.UpdateReplicaStatus(true)
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
	go sendTelemetry()
}

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go ProcessExitMonitor(sigs, processExitCallback)
	go startReplica(ctrler)
	dncServer.Start()
}
