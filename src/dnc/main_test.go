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
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/Azure/azure-container-networking/cns"
)

var (
	apiVersion = "?api-version=2018-03-01"
	dncEP      = "http://127.0.0.1:9000"
)

// TestMain :- main
func TestMain(m *testing.M) {
	if err := Setup(); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

// todo: this is the exact same logic as in main. extract to a single location
func StartDnc(managed bool) {
	configpath, found := os.LookupEnv("DNC_CONFIGURATION_PATH")
	if !found {
		configpath = "config_test.json"
	}

	config = configuration.Load(configpath)
	configuration.SetDncConfigDefaults(config)

	config.DNCSettings.Managed = managed
	if managed {
		config.ServerSettings.IPAddress = "127.0.0.1"
		config.ServerSettings.Port = "9002"
		config.CNSSettings.ChannelMode = cns.Managed
	}

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

	ctrler = controller.NewController(replicaInstance, sessionManager, config.CNSSettings.ConnectionStrings.SetOrchestratorTypeURLFmt,
		masterElectionShim, config, httpClient, cnsHttpClient)
	dncServer = server.Server{
		ServerSettings: config.ServerSettings,
		Controller:     ctrler}
	dncServer.Init()
	requestContextRetryInterval = time.Duration(config.DNCSettings.RequestRetryBaseInterval) * time.Millisecond
	maxRequestBackoffInterval = time.Duration(config.DNCSettings.MaxRequestBackoffInterval) * time.Millisecond

	if config.DataStoreSettings != (configuration.DataStoreSettings{}) {
		masterElectionShim.Init()
		datamodel.Init(
			config.CNSSettings,
			config.DataStoreSettings,
			config.DNCSettings,
			config.VlanSettings.Min,
			config.VlanSettings.Max,
			httpClient,
			cnsHttpClient,
			config.HostAgentSettings.WireServerIP)
		ctrler.UpdateReplicaStatus(true)
	}

	networkupdatehandler.Init(config, replicaInstance, sessionManager, httpClient, logger)
	initNetworkContainerStateMachine()

	go startReplica(ctrler)
	dncServer.Start()
}
