package main

import (
	"dnc/configuration"
	"dnc/controller"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/masterelection/masterelectionshim"
	"dnc/replica"
	"dnc/server"
	"os"
	"os/signal"
	"syscall"
)

var isPrimary struct{ flag int32 }
var cancelProcessing = make(chan bool)
var startProcessing = make(chan bool)
var startChan = make(chan bool)
var doneChan = make(chan bool)
var config *configuration.Configuration
var masterElectionShim *masterelectionshim.MasterElectionShim
var logger *loggershim.LoggerShim

var ctrler *controller.Controller
var dncServer server.Server
var replicaInstance *replica.Replica

func init() {
	configpath, found := os.LookupEnv("DNC_CONFIGURATION_PATH")
	if !found {
		configpath = "config.json"
	}
	config = configuration.Load(configpath)
	logger = loggershim.NewLoggerShim(config)
	logger.Init()
	datamodel.Init(config.DataStoreSettings)
	replicaInstance = replica.NewReplica(config.ServerSettings)

	masterElectionShim = masterelectionshim.NewMasterElectionShim(
		config,
		replicaInstance.UpdateReplicaStatus,
		replicaInstance.SetPrimaryIPEndpoint,
		startChan,
		doneChan)
	masterElectionShim.Init()

	ctrler = controller.NewController(replicaInstance)
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
	go masterElectionShim.Start()
	dncServer.Start()
}
