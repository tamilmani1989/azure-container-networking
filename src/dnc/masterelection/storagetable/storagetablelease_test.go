package storagetablelease

import (
	"dnc/api"
	"dnc/configuration"
	"dnc/datamodel"
	"dnc/datastoreshim"
	"dnc/logger/loggershim"
	"dnc/replica"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Azure/azure-container-networking/telemetry"
)

var (
	startChanGlobal  = make(chan bool)
	cancelChanGlobal = make(chan bool)
	demoteChanGlobal = make(chan bool)
	reports          = make(chan telemetry.DNCReport)
	config           *configuration.Configuration
	logger           *loggershim.LoggerShim
	stderr           bool
	initialized      bool
	dataStoreShim    *datastoreshim.DataStoreShim
)

const (
	maxLogFileSize  = 16 * 1024 * 1024 // Set 16MB as the max log file size
	maxLogFileCount = 8                // Set 8 as max log file count.
)

type TestPrimary struct {
	st          *StorageTableLease
	replica     *replica.Replica
	startEvent  chan bool
	cancelEvent chan bool
	demoteEvent chan bool
	replicaId   string
}

// TestPrimaryElection1 :- Test primary election by calling
// direct methods like AcquireLease, RenewLease.
func TestPrimaryElection1(t *testing.T) {
	if err := setup(); err != nil {
		panic(err)
	}

	c1 := make(chan bool)
	c2 := make(chan bool)
	c3 := make(chan bool)
	r := make(chan bool)
	var leaseStatus [3]bool
	var leaseAcquireAttempted bool
	var replicaExited int

	st1 := getStorageTableLease()
	st2 := getStorageTableLease()
	st3 := getStorageTableLease()

	//timeout := time.After(3 * time.Minute)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	go acquireLease("r1", st1, c1, r)
	go acquireLease("r2", st2, c2, r)
	go acquireLease("r3", st3, c3, r)

	// Keep trying until we're timed out or got a result or got an error
	for {
		select {
		case l1 := <-c1:
			leaseStatus[0] = l1
			leaseAcquireAttempted = true
		case l2 := <-c2:
			leaseStatus[1] = l2
			leaseAcquireAttempted = true
		case l3 := <-c3:
			leaseStatus[2] = l3
			leaseAcquireAttempted = true
		case <-r:
			fmt.Println("replicaExited received in main loop. Value=", replicaExited)
			replicaExited++
			if replicaExited >= 3 {
				fmt.Println("Exiting main as all replicas have exited", replicaExited)
				return
			}

		// Got a tick, we should check on doSomething()
		case <-ticker.C:
			found := false
			for _, value := range leaseStatus {
				if value {
					if found {
						t.Error("More than one replica is primary")
						return
					}

					found = true
				}
			}

			if leaseAcquireAttempted && !found {
				t.Error("NO replica is primary")
				return
			}
		}
	}
}

// TestPrimaryElection2 :- Test primary election by calling
// Start method which is also invoked by DNC
func TestPrimaryElection2(t *testing.T) {
	if err := setup(); err != nil {
		panic(err)
	}

	r1 := replica.NewReplica(config.ServerSettings, refreshMonitor)
	r2 := replica.NewReplica(config.ServerSettings, refreshMonitor)
	r3 := replica.NewReplica(config.ServerSettings, refreshMonitor)

	start1 := make(chan bool)
	start2 := make(chan bool)
	start3 := make(chan bool)

	cancel1 := make(chan bool)
	cancel2 := make(chan bool)
	cancel3 := make(chan bool)

	demote1 := make(chan bool)
	demote2 := make(chan bool)
	demote3 := make(chan bool)

	st1 := getStorageTableLeaseCustom(r1, "r1", start1, cancel1, demote1)
	st2 := getStorageTableLeaseCustom(r2, "r2", start2, cancel2, demote2)
	st3 := getStorageTableLeaseCustom(r3, "r3", start3, cancel3, demote3)
	replicas := [3]TestPrimary{
		{st1, r1, start1, cancel1, demote1, "r1"},
		{st2, r2, start2, cancel2, demote2, "r2"},
		{st3, r3, start3, cancel3, demote3, "r3"}}

	timeout := time.After(3 * time.Minute)
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	go st1.Start()
	st1.startEvent <- true
	go st2.Start()
	st2.startEvent <- true
	go st3.Start()
	st3.startEvent <- true

	// Keep trying until we're timed out or got a result or got an error
	var primaryElectedOnce bool
	for {
		select {
		case <-timeout:
			if !primaryElectedOnce {
				t.Error("NO replica was elected primary ever")
				return
			}
			for _, value := range replicas {
				value.st.Cleanup()
			}
			fmt.Println("Exiting main for TestPrimaryElection2")
			return
		case <-ticker.C:
			// With current implementation, two replicas can be primary
			// for a brief moment. Reason being it is not a consensus based
			// algorithm rather it is based on a distributed locking and timers.
			// So test at least 3 times and we make sure we consistently see more than
			// one primary before failing the test.
			count := 0
			for i := 0; i < 3; i++ {

				found := false
				for _, value := range replicas {

					if value.replica.IsReplicaPrimary() {
						fmt.Println(time.Now().String(), ":--> Primary = ", value.replicaId)
						if found {
							count++
							break
						}

						found = true
						primaryElectedOnce = true
					}
				}

				time.Sleep(time.Second * time.Duration(3))
			}

			if count >= 3 {
				t.Error("More than one replica is primary")
			}
		}
	}
}

// Acquire a lease, revoke it with invalid leaseID
// and then correct lease ID
func TestRevokeLease(t *testing.T) {
	if err := setup(); err != nil {
		panic(err)
	}

	// Acquire lease
	st := getStorageTableLease()
	leaseID1, success := st.AcquireLease(0)
	if !success {
		t.Error("Lease not acquired")
		return
	}
	fmt.Println("Lease acquired")

	// revoke lease with wrong lease id
	success = st.RevokeLease(int64(12345))
	if success {
		t.Error("Lease revoked with wrong lease ID")
		return
	}
	fmt.Println("Lease revoke failed with wrong leaseID")

	// revoke lease with wrong lease id
	success = st.RevokeLease(leaseID1)
	if !success {
		t.Error("Lease NOT revoked")
		return
	}
	fmt.Println("Lease revoke succeeded with correct leaseID")
}

// Acquire a lease, renew it with invalid leaseID
// and then correct lease ID
func TestRenewLease(t *testing.T) {
	if err := setup(); err != nil {
		panic(err)
	}

	// Acquire lease
	st := getStorageTableLease()
	leaseID1, success1 := st.AcquireLease(0)
	if !success1 {
		t.Error("Lease not acquired")
		return
	}
	fmt.Println("Lease acquired. leaseID1=", leaseID1)

	// revoke lease with wrong lease id
	_, success2 := st.RenewLease(int64(12345))
	if success2 {
		t.Error("Lease renewed with wrong lease ID")
		return
	}
	fmt.Println("Lease renewal failed with wrong leaseID")

	// Wait few seconds before renewal
	fmt.Println("Wait for about 10 seconds before lease renewal")
	time.Sleep(time.Second * 10)

	// revoke lease with wrong lease id
	leaseID2, success3 := st.RenewLease(leaseID1)
	if !success3 {
		t.Error("Lease NOT renewed with correct leaseID")
		return
	}
	fmt.Println("Lease renewed with correct leaseID. New leaseID2=", leaseID2)

	if leaseID1 != leaseID2 {
		fmt.Println("leaseID before renewal=", leaseID1, ". after renewal=", leaseID2)
	} else {
		t.Error(
			"LeaseId after renewal should be different. leaseID before renewal=", leaseID1,
			". after renewal=", leaseID2)
	}

	// Revoke lease as cleanup
	st.RevokeLease(leaseID2)
}

// TestInitialization :- Test Init method
func TestInitialization(t *testing.T) {
	if err := setup(); err != nil {
		panic(err)
	}

	replicaInstance := replica.NewReplica(
		config.ServerSettings,
		refreshMonitor)
	st1 := NewStorageTableLease(
		config.ServerSettings,
		configuration.DataStoreSettings{},
		config.MasterElectionSettings.DataStoreLeaseSettings,
		config.DNCSettings,
		datamodel.TableToColumnMap,
		replicaInstance.UpdateReplicaStatus,
		replicaInstance.SetPrimaryIPEndpoint,
		startChanGlobal,
		cancelChanGlobal,
		demoteChanGlobal)

	st1.SetDataStoreSettings(config)
	st1.Init()
	go st1.Start()

	// Wait few seconds to check that Init succeeded and there is no crash in Start
	time.Sleep(time.Second * time.Duration(60))
}

// Setup :- Set up the environment (if necessary).
func setup() error {

	if initialized {
		return nil
	}

	configpath, found := os.LookupEnv("DNC_CONFIGURATION_PATH")
	if !found {
		configpath = "config_test.json"
	}

	config = configuration.Load(configpath)
	configuration.SetDncConfigDefaults(config)
	logger = loggershim.NewLoggerShim(config)
	logger.Init(stderr, reports)
	logger.SetLogFileLimits(maxLogFileSize, maxLogFileCount)
	dataStoreShim = datastoreshim.NewDataStoreShim(config.DataStoreSettings, config.DNCSettings, datamodel.TableToColumnMap)

	initialized = true
	return nil
}

// getStorageTableLease :- Get an instace of StorageTableLease
func getStorageTableLease() *StorageTableLease {
	replicaInstance := replica.NewReplica(
		config.ServerSettings,
		refreshMonitor)
	return TestNewStorageTableLease(
		config.ServerSettings,
		config.DataStoreSettings,
		config.MasterElectionSettings.DataStoreLeaseSettings,
		config.DNCSettings,
		dataStoreShim,
		datamodel.TableToColumnMap,
		replicaInstance.UpdateReplicaStatus,
		replicaInstance.SetPrimaryIPEndpoint,
		"",
		startChanGlobal,
		cancelChanGlobal,
		demoteChanGlobal)
}

// getStorageTableLease :- Get an instace of StorageTableLease
func getStorageTableLeaseCustom(
	replicaInstance *replica.Replica, replicaID string, start, cancel, demote chan bool) *StorageTableLease {
	return TestNewStorageTableLease(
		config.ServerSettings,
		config.DataStoreSettings,
		config.MasterElectionSettings.DataStoreLeaseSettings,
		config.DNCSettings,
		dataStoreShim,
		datamodel.TableToColumnMap,
		replicaInstance.UpdateReplicaStatus,
		replicaInstance.SetPrimaryIPEndpoint,
		replicaID,
		start,
		cancel,
		demote)
}

func refreshMonitor() api.RefreshMonitorStatus {
	return api.RefreshMonitorStatus{}
}

// Acquire/renew lease wrapper
func acquireLease(replicaID string, st *StorageTableLease, leased chan bool, replicaExited chan bool) {
	var leaseID int64
	var leaseAcquired bool
	timeout := time.After(3 * time.Minute)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Keep trying until we're timed out or got a result or got an error
	for {
		select {
		case <-timeout:
			if leaseID > 0 {
				fmt.Println(time.Now().String(), ":Revoking lease. ReplicaID = ", replicaID)
				revoked := st.RevokeLease(leaseID)
				fmt.Println(time.Now().String(), ":Revoked lease. ReplicaID = ", replicaID, ". Result=", revoked)
			}

			replicaExited <- true
			fmt.Println(time.Now().String(), ": ReplicaID exiting = ", replicaID)
			return
		// Got a tick, we should check on doSomething()
		case <-ticker.C:
			if leaseID > 0 {
				fmt.Println(time.Now().String(), ":Attempting to renew lease. ReplicaID = ", replicaID)
				leaseID, leaseAcquired = st.RenewLease(leaseID)
			} else {
				fmt.Println(time.Now().String(), ":Attempting to acquire lease. ReplicaID = ", replicaID)
				leaseID, leaseAcquired = st.AcquireLease(leaseID)
			}

			if leaseAcquired {
				fmt.Println(time.Now().String(), ":--> Lease acquired by = ", replicaID)
			} else {
				fmt.Println(time.Now().String(), ":*** Failed to acquire lease by = ", replicaID)
			}
			leased <- leaseAcquired
		}
	}
}
