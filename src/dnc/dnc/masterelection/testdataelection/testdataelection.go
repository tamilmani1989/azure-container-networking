package testdataelection

import (
	"dnc/configuration"
	"dnc/masterelection/common"
	"dnc/utils"
	"log"
)

// TestDataElection - struct to hold TestDataElection settings.
type TestDataElection struct {
	serverSettings                        configuration.ServerSettings
	replicaChangeNotificationCallback     callbacks.ReplicaChangeNotification
	primaryIPEndpointNotificationCallback callbacks.PrimaryIPEndpointNotification
	cancelEvent                           chan bool
	startEvent                            chan bool
}

// NewTestDataElection - Create new TestDataElection
func NewTestDataElection(
	cfgServerSettings configuration.ServerSettings,
	replicaStatusNotificationCallback callbacks.ReplicaChangeNotification,
	primaryIPNotificationCallback callbacks.PrimaryIPEndpointNotification,
	start chan bool,
	cancel chan bool) *TestDataElection {
	testDataElection := new(TestDataElection)
	testDataElection.serverSettings = cfgServerSettings
	testDataElection.replicaChangeNotificationCallback = replicaStatusNotificationCallback
	testDataElection.primaryIPEndpointNotificationCallback = primaryIPNotificationCallback
	testDataElection.cancelEvent = cancel
	testDataElection.startEvent = start

	return testDataElection
}

// Init - Initialize testdataelection for primary election.
func (t *TestDataElection) Init() {
}

// Cleanup - Cleanup of testdataelection.
func (t *TestDataElection) Cleanup() {
	t.cancelEvent <- true
}

// Start - Start the leader election process
func (t *TestDataElection) Start() {
	ipAddress, err := utils.GetLocalIP()

	if err != nil {
		log.Println("Failed to get local IP address due to error", err)
		utils.OnErrorFail(err, "Failed to get local host IP")
	}

	ipEndpoint := ipAddress + ":" + t.serverSettings.Port
	t.replicaChangeNotificationCallback(true)
	t.primaryIPEndpointNotificationCallback(ipEndpoint)
}
