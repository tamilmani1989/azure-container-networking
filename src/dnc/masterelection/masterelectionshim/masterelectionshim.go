package masterelectionshim

import (
	"dnc/configuration"
	"dnc/datamodel"
	"dnc/masterelection/blobstore"
	callbacks "dnc/masterelection/common"
	storagetablelease "dnc/masterelection/storagetable"
	"dnc/masterelection/testdataelection"
	"dnc/utils"
	"fmt"
)

// MasterElectionInterface - Interfaces undelrying leader election service should implement
type MasterElectionInterface interface {
	Init()
	Start()
	Cleanup()
	SetDataStoreSettings(*configuration.Configuration)
}

// MasterElectionShim - Struct to cache the LeaderElectionShim functions
type MasterElectionShim struct {
	masterElectionSettings                configuration.MasterElectionSettings
	serverSettings                        configuration.ServerSettings
	replicaChangeNotificationCallback     callbacks.ReplicaChangeNotification
	primaryIPEndpointNotificationCallback callbacks.PrimaryIPEndpointNotification
	startEvent                            chan bool
	cancelEvent                           chan bool
	demoteEvent                           chan bool
	masterElectionInterface               MasterElectionInterface
}

// NewMasterElectionShim - Create a NewMasterElectionShim
func NewMasterElectionShim(
	cfg *configuration.Configuration,
	replicaStatusNotificationCallback callbacks.ReplicaChangeNotification,
	primaryIPNotificationCallback callbacks.PrimaryIPEndpointNotification,
	start chan bool,
	cancel chan bool,
	demote chan bool) *MasterElectionShim {
	masterElectionShim := new(MasterElectionShim)
	masterElectionShim.serverSettings = cfg.ServerSettings
	masterElectionShim.masterElectionSettings = cfg.MasterElectionSettings
	masterElectionShim.replicaChangeNotificationCallback = replicaStatusNotificationCallback
	masterElectionShim.primaryIPEndpointNotificationCallback = primaryIPNotificationCallback
	masterElectionShim.cancelEvent = cancel
	masterElectionShim.startEvent = start
	masterElectionShim.demoteEvent = demote

	useDataStore := masterElectionShim.masterElectionSettings.UseDataStore
	if !useDataStore && masterElectionShim.masterElectionSettings.BlobStorageSettings != (configuration.BlobStorageSettings{}) {
		blobStore := blobstore.NewBlobStore(
			masterElectionShim.serverSettings,
			masterElectionShim.masterElectionSettings.BlobStorageSettings,
			masterElectionShim.replicaChangeNotificationCallback,
			masterElectionShim.primaryIPEndpointNotificationCallback,
			masterElectionShim.startEvent,
			masterElectionShim.cancelEvent,
			masterElectionShim.demoteEvent)
		masterElectionShim.masterElectionInterface = blobStore
	} else if useDataStore && masterElectionShim.masterElectionSettings.DataStoreLeaseSettings != (configuration.DataStoreLeaseSettings{}) {
		storageTableLease := storagetablelease.NewStorageTableLease(
			masterElectionShim.serverSettings,
			cfg.DataStoreSettings,
			masterElectionShim.masterElectionSettings.DataStoreLeaseSettings,
			cfg.DNCSettings,
			datamodel.TableToColumnMap,
			masterElectionShim.replicaChangeNotificationCallback,
			masterElectionShim.primaryIPEndpointNotificationCallback,
			masterElectionShim.startEvent,
			masterElectionShim.cancelEvent,
			masterElectionShim.demoteEvent)
		masterElectionShim.masterElectionInterface = storageTableLease
	} else if masterElectionShim.masterElectionSettings.TestDataElectionSettings != (configuration.TestDataElectionSettings{}) {
		testdataElection := testdataelection.NewTestDataElection(
			masterElectionShim.serverSettings,
			masterElectionShim.replicaChangeNotificationCallback,
			masterElectionShim.primaryIPEndpointNotificationCallback,
			masterElectionShim.startEvent,
			masterElectionShim.cancelEvent)
		masterElectionShim.masterElectionInterface = testdataElection
	} else {
		utils.OnErrorFail(fmt.Errorf("Invalid/UnSupported data store"), "")
	}

	return masterElectionShim
}

// Init - Function to initialize underlying leader election service.
func (masterElectionShim *MasterElectionShim) Init() {
	masterElectionShim.masterElectionInterface.Init()
}

// Start - Function to start underlying leader election service.
func (masterElectionShim *MasterElectionShim) Start() {
	go masterElectionShim.masterElectionInterface.Start()
	masterElectionShim.startEvent <- true
}

// Cleanup - Function to Cleanup underlying leader election service.
func (masterElectionShim *MasterElectionShim) Cleanup() {
	masterElectionShim.masterElectionInterface.Cleanup()
}

// SetDataStoreSettings - set datastore settings for masterelection shim
func (masterElectionShim *MasterElectionShim) SetDataStoreSettings(config *configuration.Configuration) {
	masterElectionShim.masterElectionInterface.SetDataStoreSettings(config)
}

// RelinquishPrimaryStatus - Send an event to lose primary status
func (masterElectionShim *MasterElectionShim) RelinquishPrimaryStatus() {
	masterElectionShim.demoteEvent <- true
}
