package storagetablelease

import (
	"dnc/configuration"
	"dnc/datamodel"
	"dnc/datastoreshim"
	"dnc/logger/loggershim"
	callbacks "dnc/masterelection/common"
	"dnc/utils"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	// LeaseTableRowKey - row key for lease table
	leaseTableRowKey = "LeaseId"
	// LeaseTablePartitionKey - partition key for lease table
	leaseTablePartitionKey = "LeasePartition"
	// LeaseDurationProperty - duration of the lease
	leaseDurationProperty = "LeaseDuration"
	// primaryReplicaIPProperty - IP address of primary replica
	primaryReplicaEPProperty = "primaryReplicaEP"
	// DemoteAcquireLeaseIntervalPenaltyInSecs - penalty
	DemoteAcquireLeaseIntervalPenaltyInSecs = 5
	// retry count
	retryCount = 2
)

// StorageTableLease =
type StorageTableLease struct {
	serverSettings                        configuration.ServerSettings
	dataStoreSettings                     configuration.DataStoreSettings
	leaseSettings                         configuration.DataStoreLeaseSettings
	dncSettings                           configuration.DNCSettings
	replicaChangeNotificationCallback     callbacks.ReplicaChangeNotification
	primaryIPEndpointNotificationCallback callbacks.PrimaryIPEndpointNotification
	cancelEvent                           chan bool
	startEvent                            chan bool
	demoteEvent                           chan bool
	leaseID                               int64
	mutex                                 *sync.RWMutex
	dataStoreShim                         datastoreshim.DataStoreShim
	initialized                           bool
	identity                              string
	tableToColumnMap                      map[string]string
}

// NewStorageTableLease - Create StorageTableLease variable
func NewStorageTableLease(
	cfgServerSettings configuration.ServerSettings,
	dataStoreSettings configuration.DataStoreSettings,
	dataStoreLeaseSettings configuration.DataStoreLeaseSettings,
	dncSettings configuration.DNCSettings,
	tableToColumnMap map[string]string,
	replicaStatusNotificationCallback callbacks.ReplicaChangeNotification,
	primaryIPNotificationCallback callbacks.PrimaryIPEndpointNotification,
	start chan bool,
	cancel chan bool,
	demote chan bool) *StorageTableLease {

	tableLease := new(StorageTableLease)
	tableLease.serverSettings = cfgServerSettings
	tableLease.dataStoreSettings = dataStoreSettings
	tableLease.leaseSettings = dataStoreLeaseSettings
	tableLease.dncSettings = dncSettings
	tableLease.tableToColumnMap = tableToColumnMap
	tableLease.replicaChangeNotificationCallback = replicaStatusNotificationCallback
	tableLease.primaryIPEndpointNotificationCallback = primaryIPNotificationCallback
	tableLease.cancelEvent = cancel
	tableLease.startEvent = start
	tableLease.demoteEvent = demote
	tableLease.mutex = new(sync.RWMutex)

	tableLease.initialized = true
	return tableLease
}

// Init - Initialize storage table for primary election
func (st *StorageTableLease) Init() {
	defer utils.PanicRecoverAndExitWithStackTrace("StorageTableLease.Init")

	if st.dataStoreSettings == (configuration.DataStoreSettings{}) {
		panic("DataStore should not be empty for storagetable lease.")
	} else if st.dataStoreSettings.AzureStorageInfo == (configuration.AzureStorageInfo{}) &&
		st.dataStoreSettings.TestPersistenceStoreInfo == (configuration.TestPersistenceStoreInfo{}) {
		panic("AzureStorageInfo should not be empty for storagetable lease." +
			" If you are using a datastore other than azure storage, you still need azure storage account for dnc lease")
	}

	ds, err := datastoreshim.NewDataStoreShim(st.dataStoreSettings, st.dncSettings.ThresholdForLoggingDataStoreOpLatencyMs, st.tableToColumnMap)
	if err != nil {
		panic(err) // todo: we should bubble up the error instead of panicking
	}
	st.dataStoreShim = ds
	loggershim.LS.Debugf("StorageTableLease initialized")
}

// Cleanup - Cleanup resources
func (st *StorageTableLease) Cleanup() {
	st.cancelEvent <- true
	loggershim.LS.Debugf("Try to release lease")
	st.RevokeLease(st.leaseID)
}

// Start - Start the primary election process
func (st *StorageTableLease) Start() {
	defer utils.PanicRecoverAndExitWithStackTrace("StorageTableLease.Start")
	st.tryAcquireLease()
}

// SetDataStoreSettings - Function to set account name, key and partition key
func (st *StorageTableLease) SetDataStoreSettings(config *configuration.Configuration) {
	st.mutex.Lock()
	st.dataStoreSettings = config.DataStoreSettings
	st.mutex.Unlock()
}

/*********************************** Private methods  ******************************/

// GetPrimaryIPEndpoint - Method to get the primary replica endpoint.
func (st *StorageTableLease) getPrimaryIPEndpoint() (string, error) {
	properties, _, _, err := st.dataStoreShim.QueryInPartitionWithEtagProperties(
		datamodel.MasterElectionTableName,
		st.getPartitionKey(),
		leaseTableRowKey)

	if err != nil {
		return "", fmt.Errorf("%s: Failed to query MasterElectionTableName. %s", utils.TracePrefix(2), err.Error())
	}

	replicaEP, ok := st.getreplicaPrimaryEPProperty(properties)
	if !ok {
		return "", fmt.Errorf("%s: Error in getreplicaPrimaryEPProperty", utils.TracePrefix(2))
	}

	return replicaEP, nil
}

// updateReplicaPrimaryEndpoint - This method is called by replica that fails to
// become primary. It caches that it is not primary and primary replica's endpoint
func (st *StorageTableLease) updateReplicaPrimaryEndpoint() {
	primaryIPEP, err := st.getPrimaryIPEndpoint()
	if err != nil {
		loggershim.LS.Errorf("Failed to get primary endpoint due to error %s", err.Error())
	} else {
		loggershim.LS.Debugf("Updating primary replica EP %s [%s]", primaryIPEP, st.identity)
	}

	st.replicaChangeNotificationCallback(false)
	st.primaryIPEndpointNotificationCallback(primaryIPEP)
}

// UpdatePrimaryIPEndpoint - This method is called by replica that becomes primary.
// It notes that it is primary and then caches endpoint where is listening.
func (st *StorageTableLease) updatePrimaryIPEndpoint() error {
	// Fetching IP from getReplicaIPEndpoint is optimization. It could have been
	// read from lease table as well (saves us a call to storage), since this method
	// is called from primary replica
	// endpoint now includes http(s) scheme to support https migration
	endpoint, err := st.getReplicaIPEndpoint()
	if err != nil {
		return err
	}

	loggershim.LS.Debugf("Replica %v [%s] got elected as primary. ", endpoint, st.identity)
	st.replicaChangeNotificationCallback(true)
	st.primaryIPEndpointNotificationCallback(endpoint)

	return nil
}

// // tryAcquireLease - Function to acquire and renew lease.
func (st *StorageTableLease) tryAcquireLease() error {

	updateBlobRetryInterval :=
		(time.Second) * time.Duration(st.leaseSettings.AcquireRetryIntervalInSecs/2)
	leaseAcquireRetryTimer := time.NewTicker(
		time.Second * time.Duration(st.leaseSettings.AcquireRetryIntervalInSecs))
	leaseRenewRetryTimer := time.NewTicker(
		time.Second * time.Duration(st.leaseSettings.LeaseIntervalInSecs))
	updatedLeaseAcquireIntervalTimeInSecs := st.leaseSettings.AcquireRetryIntervalInSecs + DemoteAcquireLeaseIntervalPenaltyInSecs
	var leaseOperSuccess bool
	var resetLeaseAcquireTimer bool

	for {
	AcquireLease:
		st.replicaChangeNotificationCallback(false)
		for {
			select {
			case <-leaseAcquireRetryTimer.C:
				if resetLeaseAcquireTimer {
					leaseAcquireRetryTimer.Stop()
					leaseAcquireRetryTimer = time.NewTicker(
						time.Second * time.Duration(st.leaseSettings.AcquireRetryIntervalInSecs))
					resetLeaseAcquireTimer = false
				}

				updateRetryCount := 0
				st.leaseID, leaseOperSuccess = st.AcquireLease(st.leaseID)
				if !leaseOperSuccess {
					st.updateReplicaPrimaryEndpoint()
					break
				}

				loggershim.LS.Debugf("Lease acquired at retry with leaseID: %d [%s]", st.leaseID, st.identity)

				for updateRetryCount < retryCount {
					err := st.updatePrimaryIPEndpoint()
					if err != nil {
						loggershim.LS.Errorf("%s", err.Error())
						time.Sleep(updateBlobRetryInterval)
						updateRetryCount++
					} else {
						break
					}
				}

				goto AcquiredLease

			case <-st.startEvent:
				updateRetryCount := 0
				st.leaseID, leaseOperSuccess = st.AcquireLease(st.leaseID)

				if !leaseOperSuccess {
					st.updateReplicaPrimaryEndpoint()
					break
				}

				loggershim.LS.Debugf("Lease acquired at start with leaseID: %d [%s]", st.leaseID, st.identity)

				for updateRetryCount < retryCount {
					err := st.updatePrimaryIPEndpoint()
					if err != nil {
						loggershim.LS.Errorf("%s", err.Error())
						time.Sleep(updateBlobRetryInterval)
						updateRetryCount++
					} else {
						break
					}
				}

				goto AcquiredLease

			// On receiving demote event we do the following.
			// Stop the current lease acquire timer.
			// Create a new lease acquire timer with increased lease acquire interval.
			case <-st.demoteEvent:
				loggershim.LS.Debugf("Received Demote event on non replica primary")
				leaseOperSuccess := st.RevokeLease(st.leaseID)
				if !leaseOperSuccess {
					loggershim.LS.Errorf("Revoke lease failed: LeaseId:%s", st.leaseID)
				}

				leaseAcquireRetryTimer.Stop()

				leaseAcquireRetryTimer = time.NewTicker(time.Second * time.Duration(updatedLeaseAcquireIntervalTimeInSecs))
				resetLeaseAcquireTimer = true
				goto AcquireLease

			case <-st.cancelEvent:
				loggershim.LS.Debugf("Received cancel event")
				return nil
			}
		}

	AcquiredLease:
		for {
			select {
			case <-leaseRenewRetryTimer.C:
				st.leaseID, leaseOperSuccess = st.RenewLease(st.leaseID)
				if !leaseOperSuccess {
					loggershim.LS.Errorf("Renew lease failed with leaseID: %d [%s]", st.leaseID, st.identity)
					st.replicaChangeNotificationCallback(false)
					goto AcquireLease
				}

				loggershim.LS.Debugf("Renew lease succeeded with leaseID: %d [%s]", st.leaseID, st.identity)
				primaryIPEndpoint, err := st.getPrimaryIPEndpoint()
				if err != nil {
					st.primaryIPEndpointNotificationCallback(primaryIPEndpoint)
				}

				break

				// On receiving demote event we do the following.
				// Release the lease on the blob.
				// Stop the current lease acquire timer.
				// Create a new lease acquire timer with increased lease acquire interval.
			case <-st.demoteEvent:
				loggershim.LS.Debugf("Received Demote event on replica primary")
				leaseOperSuccess = st.RevokeLease(st.leaseID)
				loggershim.LS.Debugf("Release lease succeeded:%t, LeaseId:%d", leaseOperSuccess, st.leaseID)

				leaseAcquireRetryTimer = time.NewTicker(
					time.Second * time.Duration(updatedLeaseAcquireIntervalTimeInSecs))
				resetLeaseAcquireTimer = true
				goto AcquireLease

			case <-st.cancelEvent:
				loggershim.LS.Debugf("Received cancel event")
				return nil
			}
		}
	}
}

// AcquireLease - Function to acquire/renew lease.
// --- Parameters ---
// currentLeaseID - It is optional. A lease holder with valid lease can pass it in
//     and get lease renewed earlier than expiry time. Otherwise lease will only be
//     assigned if there is no active lease.
// --- Return values ---
// int64 - It is the lease ID. Once lease is acquired, a unique id is generated and returned to caller.
//           Caller can use the same unique ID to renew the lease
// bool - success or failure. If true, method succeeded otherwise failed. For failure case, error is logged
// --- Notes ---
// lease duration is stored as unix epoch in storage and same epoch is interchangebly used as leaseID.
func (st *StorageTableLease) AcquireLease(currentLeaseID int64) (int64, bool) {
	// check if row exists and if not, try creating it
	properties, etag, status, err := st.dataStoreShim.QueryInPartitionWithEtagProperties(
		datamodel.MasterElectionTableName,
		st.getPartitionKey(),
		leaseTableRowKey)

	if err != nil {
		if status != http.StatusNotFound {
			loggershim.LS.Errorf(
				"%s: Failed to query MasterElectionTableName. %s", utils.TracePrefix(2), err.Error())
			return 0, false
		}

		props, ok := st.getLeaseAcquireProperties()
		if !ok {
			return 0, false
		}

		status, insertErr := st.dataStoreShim.InsertInPartitionProperties(
			datamodel.MasterElectionTableName, st.getPartitionKey(), leaseTableRowKey, props)

		if insertErr != nil && status != http.StatusConflict {
			loggershim.LS.Errorf(
				"%s: Failed to insert row in MasterElectionTableName. %s", utils.TracePrefix(2), insertErr.Error())
		}

		// Get lease ID and return to caller. Caller can use this ID to renew lease.
		// As mentioned in comments of this method, lease ID is actually calcuated
		// from lease duration (stored as unix epoch) in CosmosDB table.
		leaseUntilEpoch, _ := st.getLeaseDurationProperty(props)
		return st.checkErrorAndGetLeaseID(leaseUntilEpoch, insertErr), insertErr == nil
	}

	// If row found, then read the lease time property and see if lease has expired.
	leaseEpoch, ok := st.getLeaseDurationProperty(properties)
	if !ok {
		return 0, false
	}

	leaseUntil := time.Unix(leaseEpoch, 0)
	if time.Now().Before(leaseUntil) && leaseEpoch != currentLeaseID {
		// if lease is active and caller doesn't hold this lease, skip it
		loggershim.LS.Errorf(
			"%s: Skip AcquireLease as current lease is active. Leaseuntil=%s [%s]",
			utils.TracePrefix(2),
			leaseUntil.String(),
			st.identity)
		return 0, false
	}

	// if current time is past lease time, try acquiring lease by updating row
	props, ok := st.getLeaseAcquireProperties()
	if !ok {
		return 0, false
	}

	updateErr := st.dataStoreShim.UpdateInPartitionWithEtag(
		datamodel.MasterElectionTableName, st.getPartitionKey(), leaseTableRowKey, etag, props)
	if updateErr != nil {
		loggershim.LS.Errorf(
			"%s: Leasetime update error. %s", utils.TracePrefix(2), updateErr.Error())
	}

	leaseUntilEpoch, _ := st.getLeaseDurationProperty(props)
	return st.checkErrorAndGetLeaseID(leaseUntilEpoch, updateErr), updateErr == nil
}

// RevokeLease - Function to revoke lease.
// currentLeaseID - leaseID of the current lease. If lease ID is invalid, function fails
func (st *StorageTableLease) RevokeLease(currentLeaseID int64) bool {
	// check if row exists for lease
	properties, etag, _, err := st.dataStoreShim.QueryInPartitionWithEtagProperties(
		datamodel.MasterElectionTableName, st.getPartitionKey(), leaseTableRowKey)
	if err != nil {
		loggershim.LS.Errorf(
			"%s: Failed to query MasterElectionTableName for RevokeLease. %s", utils.TracePrefix(2), err.Error())
		return false
	}

	// If row found, then read the lease time property and see if lease has expired.
	leaseEpoch, ok := st.getLeaseDurationProperty(properties)
	if !ok {
		return false
	}

	leaseUntil := time.Unix(leaseEpoch, 0)
	if !(leaseEpoch == currentLeaseID && time.Now().Before(leaseUntil)) {
		loggershim.LS.Errorf(
			"%s: Skip revoke lease as caller doesn't have active lease. Active leaseID=%d. Caller leaseID=%d",
			utils.TracePrefix(2),
			leaseEpoch,
			currentLeaseID)
		return false
	}

	// if current lease is valid, try revoking it by update row with default lease values
	// default lease properties are - leaseID = 0 and replica IP address with lease = ""
	props := st.getLeaseRevokeProperties()
	updateErr := st.dataStoreShim.UpdateInPartitionWithEtag(
		datamodel.MasterElectionTableName, st.getPartitionKey(), leaseTableRowKey, etag, props)
	if updateErr != nil {
		loggershim.LS.Errorf(
			"%s: Revoke lease update error. %s", utils.TracePrefix(2), updateErr.Error())
	}

	return updateErr == nil
}

// RenewLease - Renew lease.
func (st *StorageTableLease) RenewLease(leaseID int64) (int64, bool) {
	if leaseID > 0 {
		return st.AcquireLease(leaseID)
	}

	loggershim.LS.Errorf(
		"%s: Invalid lease ID for RenewLease. Skipping lease renewal", utils.TracePrefix(2))
	return 0, false
}

// checkErrorAndGetLeaseID - Check if error is nil or not and return the right
// value of leaseID. This method encapsulates if-else statement to reduce verbosity
func (st *StorageTableLease) checkErrorAndGetLeaseID(id int64, err error) int64 {
	if err == nil {
		return id
	}

	return int64(0)
}

// getLeaseTimeEpoch - Get lease duration (from current time) as epoch
func (st *StorageTableLease) getLeaseTimeEpoch() int64 {
	return time.Now().Add(time.Second * time.Duration(st.leaseSettings.LeaseIntervalInSecs)).Unix()
}

// getLeaseAcquireProperties - Create a map of lease properties for acquiring lease
// ---Return Values---
// It returns a map of properties. For acquiring lease, values of properties are
// LeaseID (property) = a valid leaseID (int64 number)
// PrimaryReplicaIPAddress (property) = IP address of replica that is trying to acquire lease
func (st *StorageTableLease) getLeaseAcquireProperties() (map[string]interface{}, bool) {
	replicaIP, err := st.getReplicaIPEndpoint()
	if err != nil {
		loggershim.LS.Errorf(
			"%s: getReplicaIPEndpoint error: %s", utils.TracePrefix(2), err.Error())
		return nil, false
	}

	properties := make(map[string]interface{})
	properties[leaseDurationProperty] = st.getLeaseTimeEpoch()
	properties[primaryReplicaEPProperty] = replicaIP

	return properties, true
}

// getLeaseRevokeProperties - Create a map of lease properties for revoking lease
// ---Return Values---
// It returns a map of properties. For revoking lease, values of properties are
// LeaseID (property) = 0
// PrimaryReplicaIPAddress (property) = ""
func (st *StorageTableLease) getLeaseRevokeProperties() map[string]interface{} {
	properties := make(map[string]interface{})
	properties[leaseDurationProperty] = int64(0)
	properties[primaryReplicaEPProperty] = ""

	return properties
}

// getLeaseDurationProperty - retrieve lease duration property from map of properties
func (st *StorageTableLease) getLeaseDurationProperty(properties map[string]interface{}) (int64, bool) {
	property, ok := properties[leaseDurationProperty]
	if !ok {
		loggershim.LS.Errorf("%s: Lease duration property not found in properties", utils.TracePrefix(2))
		return int64(0), false
	}

	leaseEpoch, ok := property.(int64)
	if !ok {
		loggershim.LS.Errorf("%s: Lease duration property is not of type int64", utils.TracePrefix(2))
		return int64(0), false
	}

	return leaseEpoch, true
}

// getreplicaPrimaryEPProperty - retrieve replica EP property from map of properties
func (st *StorageTableLease) getreplicaPrimaryEPProperty(properties map[string]interface{}) (string, bool) {
	property, ok := properties[primaryReplicaEPProperty]
	if !ok {
		loggershim.LS.Errorf("%s: replicaPrimaryEP property not found in properties", utils.TracePrefix(2))
		return "", false
	}

	replicaEP, ok := property.(string)
	if !ok {
		loggershim.LS.Errorf("%s: replicaPrimaryEP property is not of type string", utils.TracePrefix(2))
		return "", false
	}

	return replicaEP, true
}

// getDataStoreSettings - Function that return datastore settings (AccountName, Key, PartitionKey, Environment)
func (st *StorageTableLease) getDataStoreSettings() (string, string, string, string) {
	return st.dataStoreSettings.AzureStorageInfo.AccountName,
		st.dataStoreSettings.AzureStorageInfo.Key,
		st.dataStoreSettings.AzureStorageInfo.PartitionKey,
		st.dataStoreSettings.AzureStorageInfo.Environment
}

// getIPAddress - Get IP address from user config or determine it from interface
func (st *StorageTableLease) getIPAddress() (string, error) {
	ipAddress := st.serverSettings.IPAddress

	if ipAddress == "" {
		var err error
		ipAddress, err = getLocalIP()
		if err != nil {
			return "", fmt.Errorf("Failed to retrive ip address due to error: %s", err.Error())
		}
	}

	return ipAddress, nil
}

// getLocalIP - Get local IP of the machine
func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", fmt.Errorf("Failed to find IP Address")
}

// GetReplicaIPEndpoint - Get endpoint of current replica
func (st *StorageTableLease) getReplicaIPEndpoint() (string, error) {
	replicaIP, err := st.getIPAddress()
	if err != nil {
		return "", err
	}

	if st.serverSettings.UseHttps {
		return "https://" + replicaIP + ":" + st.serverSettings.TlsPort, nil
	}
	return "http://" + replicaIP + ":" + st.serverSettings.Port, nil
}

// getPartitionKey - Get endpoint of current replica
func (st *StorageTableLease) getPartitionKey() string {
	if st.dataStoreSettings.AzureStorageInfo.PartitionKey != "" {
		return st.dataStoreSettings.AzureStorageInfo.PartitionKey
	}

	return leaseTablePartitionKey
}

/************************* Test Initialization Methods *******************************/

// TestNewStorageTableLease - Create StorageTableLease variable for unit test.
// Purpose is to pass in a simulated datastorehim that can be shared between
// multiple replicas running on same machine.
func TestNewStorageTableLease(
	cfgServerSettings configuration.ServerSettings,
	dataStoreSettings configuration.DataStoreSettings,
	dataStoreLeaseSettings configuration.DataStoreLeaseSettings,
	dncSettings configuration.DNCSettings,
	dataStoreShim datastoreshim.DataStoreShim,
	tableToColumnMap map[string]string,
	replicaStatusNotificationCallback callbacks.ReplicaChangeNotification,
	primaryIPNotificationCallback callbacks.PrimaryIPEndpointNotification,
	identity string,
	start chan bool,
	cancel chan bool,
	demote chan bool) *StorageTableLease {

	tableLease := new(StorageTableLease)
	tableLease.serverSettings = cfgServerSettings
	tableLease.leaseSettings = dataStoreLeaseSettings
	tableLease.dncSettings = dncSettings
	tableLease.tableToColumnMap = tableToColumnMap
	tableLease.replicaChangeNotificationCallback = replicaStatusNotificationCallback
	tableLease.primaryIPEndpointNotificationCallback = primaryIPNotificationCallback
	tableLease.cancelEvent = cancel
	tableLease.startEvent = start
	tableLease.demoteEvent = demote
	tableLease.mutex = new(sync.RWMutex)
	tableLease.dataStoreShim = dataStoreShim
	tableLease.identity = identity
	tableLease.dataStoreSettings = dataStoreSettings

	tableLease.initialized = true
	return tableLease
}
