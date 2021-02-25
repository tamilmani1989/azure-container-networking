package main

import (
	internalcns "dnc/cns"
	"dnc/cnsinterface"
	"dnc/configuration"
	"dnc/controller"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/masterelection/masterelectionshim"
	"dnc/replica"
	"dnc/resourceupdatehandler/networkupdatehandler"
	"dnc/server"
	"dnc/utils"
	"log"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/Azure/azure-container-networking/cns"
)

var (
	apiVersion = "?api-version=2018-03-01"
	dncEP      string
)

// TestMain :- main
func TestMain(m *testing.M) {
	if err := Setup(); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

func ReadConfig() *configuration.Configuration {
	configpath, found := os.LookupEnv("DNC_CONFIGURATION_PATH")
	if !found {
		configpath = "config_test.json"
	}

	config = configuration.Load(configpath)
	configuration.SetDncConfigDefaults(config)

	return config
}

// todo: this is the exact same logic as in main. extract to a single location
func StartDnc(config *configuration.Configuration) {
	// Set dncEP as per config settings
	dncEP = "http://" + config.ServerSettings.IPAddress + ":" + config.ServerSettings.Port
	logger = loggershim.NewLoggerShim(config)
	logger.Init(stderr, reports)
	logger.SetLogFileLimits(maxLogFileSize, maxLogFileCount)
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

	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: time.Duration(config.HttpClientSettings.ConnectionTimeout) * time.Second,
			}).DialContext,
			ResponseHeaderTimeout: time.Duration(config.HttpClientSettings.ResponseHeaderTimeout) * time.Second,
		},
	}

	cnsHttpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: time.Duration(config.HttpClientSettings.ConnectionTimeout) * time.Second,
			}).DialContext,
			ResponseHeaderTimeout: time.Duration(config.HttpClientSettings.ResponseHeaderTimeout) * time.Second,
		},
	}
	switch config.CNSSettings.ChannelMode {
	case cns.Direct:
		cnsInterface = &cnsinterface.DirectChannel{
			SessionManager:  sessionManager,
			EnforceNicLimit: true,
			CNSClient: &internalcns.Client{
				Paths: config.CNSSettings.ConnectionStrings,
				HTTP:  httpClient,
			},
			Logger: logger,
		}
	case cns.Managed:
		cnsInterface = cnsinterface.NewManagedChannel(sessionManager, true)
	}

	if config.DataStoreSettings != (configuration.DataStoreSettings{}) {
		masterElectionShim.Init()

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
			log.Fatalln("Error initializing datastore. Error:", err)
		}
	}

	networkupdatehandler.Init(config, replicaInstance, sessionManager, httpClient, nil, logger)
	initNetworkContainerStateMachine()

	ctrler = controller.NewController(replicaInstance, sessionManager, config.CNSSettings.ConnectionStrings.SetOrchestratorTypeURLFmt,
		masterElectionShim, config, httpClient, cnsHttpClient, &ipAllocator)
	dncServer = server.Server{
		ServerSettings: config.ServerSettings,
		Controller:     ctrler,
		Logger:         logger}
	dncServer.Init()
	requestContextRetryInterval = time.Duration(config.DNCSettings.RequestRetryBaseInterval) * time.Millisecond
	maxRequestBackoffInterval = time.Duration(config.DNCSettings.MaxRequestBackoffInterval) * time.Millisecond

	if config.DataStoreSettings != (configuration.DataStoreSettings{}) {
		ctrler.UpdateReplicaStatus(true)
	}

	go startReplica(ctrler)
	dncServer.Start()
}
