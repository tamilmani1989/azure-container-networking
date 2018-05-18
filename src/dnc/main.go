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
	"os"
	"os/signal"
	"syscall"
)

var isPrimary struct{ flag int32 }
var cancelProcessing = make(chan bool)
var startProcessing = make(chan bool)
var startChan = make(chan bool)
var doneChan = make(chan bool)
var networkMonitorStartProcessing = make(chan bool)
var tagMonitorStartProcessing = make(chan bool)
var networkMonitorStopProcessing = make(chan bool)
var tagMonitorStopProcessing = make(chan bool)

var config *configuration.Configuration
var masterElectionShim *masterelectionshim.MasterElectionShim
var logger *loggershim.LoggerShim
var sessionManager *utils.SessionManager
var ctrler *controller.Controller
var dncServer server.Server
var replicaInstance *replica.Replica
var debug bool
var stderr bool

func init() {
	flag.BoolVar(&debug, "debug", false, "Set to true to output debug statements.")
	flag.BoolVar(&stderr, "stderr", false, "Set to true to write logs to stdout.")
	flag.Parse()

	configpath, found := os.LookupEnv("DNC_CONFIGURATION_PATH")
	if !found {
		configpath = "config.json"
	}

	config = configuration.Load(configpath)
	logger = loggershim.NewLoggerShim(config)
	logger.Init(debug, stderr)
	datamodel.Init(config.DataStoreSettings, config.HostAgentSettings.RequestBaseInterval, config.VlanSettings.Min, config.VlanSettings.Max)
	replicaInstance = replica.NewReplica(config.ServerSettings)

	masterElectionShim = masterelectionshim.NewMasterElectionShim(
		config,
		replicaInstance.UpdateReplicaStatus,
		replicaInstance.SetPrimaryIPEndpoint,
		startChan,
		doneChan)
	masterElectionShim.Init()

	sessionManager = utils.NewSessionManager([]string{datamodel.NodeRepositoryName, datamodel.NetworkRepositoryName, datamodel.TagRepositoryName})
	ctrler = controller.NewController(replicaInstance, sessionManager, config.CNSSettings.ConnectionStrings.SetOrchestratorTypeURLFmt)

	dncServer = server.Server{
		ServerSettings: config.ServerSettings,
		Controller:     ctrler}
	dncServer.Init()
}

func processExitCallback() {
	masterElectionShim.Cleanup()
}

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go ProcessExitMonitor(sigs, processExitCallback)
	go monitorRequests()
	go monitorNetworkUpdates()
	go monitorTagUpdates()
	go masterElectionShim.Start()
	dncServer.Start()
}
