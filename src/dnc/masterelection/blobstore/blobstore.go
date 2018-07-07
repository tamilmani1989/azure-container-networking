package blobstore

import (
	"dnc/configuration"
	"dnc/logger/loggershim"
	"dnc/masterelection/common"
	"dnc/utils"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
)

// BlobStore - strcut to hold blob store settings.
type BlobStore struct {
	blobStorageSettings                   configuration.BlobStorageSettings
	serverSettings                        configuration.ServerSettings
	blobClient                            storage.BlobStorageClient
	replicaChangeNotificationCallback     callbacks.ReplicaChangeNotification
	primaryIPEndpointNotificationCallback callbacks.PrimaryIPEndpointNotification
	cancelEvent                           chan bool
	startEvent                            chan bool
	blob                                  *storage.Blob
	leaseID                               string
	mutex                                 *sync.RWMutex
}

const (
	blockID = "7f3ed58e-3139-4f71-8247-652de5692fb2"
)

// NewBlobStore - Create a NewBlobStore
func NewBlobStore(
	cfgServerSettings configuration.ServerSettings,
	cfgBlobSettings configuration.BlobStorageSettings,
	replicaStatusNotificationCallback callbacks.ReplicaChangeNotification,
	primaryIPNotificationCallback callbacks.PrimaryIPEndpointNotification,
	start chan bool,
	cancel chan bool) *BlobStore {
	blobStore := new(BlobStore)
	blobStore.blobStorageSettings = cfgBlobSettings
	blobStore.serverSettings = cfgServerSettings
	blobStore.replicaChangeNotificationCallback = replicaStatusNotificationCallback
	blobStore.primaryIPEndpointNotificationCallback = primaryIPNotificationCallback
	blobStore.cancelEvent = cancel
	blobStore.startEvent = start
	blobStore.mutex = new(sync.RWMutex)
	return blobStore
}

// Init - Initialize blob storage for primary election.
func (b *BlobStore) Init() {
	accountName, key, _ := b.getDataStoreSettings()
	client, err := storage.NewBasicClient(accountName, key)
	onErrorFail(err, "Create client failed")
	b.blobClient = client.GetBlobService()
}

// Cleanup - Cleanup of blob store.
func (b *BlobStore) Cleanup() {
	b.cancelEvent <- true
	loggershim.LS.Debugf("Try to release lease\n")
	b.blob.ReleaseLease(b.leaseID, nil)
}

// Start - Start the leader election process
func (b *BlobStore) Start() {
	accountName, key, partitionKey := b.getDataStoreSettings()
	if accountName == "" ||
		key == "" ||
		partitionKey == "" {
		ipAddress, err := b.getIPAddress()
		utils.OnErrorFail(err, "Failed to get local host IP")
		ipEndpoint := ipAddress + ":" + b.serverSettings.Port
		b.replicaChangeNotificationCallback(true)
		b.primaryIPEndpointNotificationCallback(ipEndpoint)
	} else {
		b.TryAcquireLease()
	}
}

func (b *BlobStore) createBlob() error {
	cnt := b.blobClient.GetContainerReference(b.blobStorageSettings.ContainerName)
	options := storage.CreateContainerOptions{
		Access: storage.ContainerAccessTypePrivate,
	}
	_, err := cnt.CreateIfNotExists(&options)
	onErrorFail(err, "Create container failed")

	b.blob = cnt.GetBlobReference(b.blobStorageSettings.BlobName)
	blobExist, checkBlobErr := b.blob.Exists()
	if checkBlobErr != nil {
		loggershim.LS.Errorf("Check if blob exist failed: %s\n", checkBlobErr.Error())
		return fmt.Errorf("create block blob failed as blob exist check failed: %+v", checkBlobErr)
	}

	if !blobExist {
		err = b.blob.CreateBlockBlob(nil)
		if err != nil {
			storagErr, ok := err.(storage.AzureStorageServiceError)
			if ok {
				if storagErr.StatusCode == 412 {
					loggershim.LS.Debugf("Blob exist and there exist as lease on the blob\n", storagErr.StatusCode)
				}
			} else {
				loggershim.LS.Errorf("create block blob failed: %s\n", err.Error())
				return fmt.Errorf("create block blob failed: %v", err)
			}
		}
		loggershim.LS.Debugf("Created blob %s\n", b.blobStorageSettings.BlobName)
	}
	return err
}

func (b *BlobStore) getIPAddress() (string, error) {
	ipAddress := b.serverSettings.IPAddress
	var err error
	if ipAddress == "" {
		ipAddress, err = getLocalIP()
		if err != nil {
			loggershim.LS.Errorf("Failed to retrieve IP Address due to error: %s\n", err.Error())
			return "", err
		}
	}

	return ipAddress, nil
}

func (b *BlobStore) UpdatePrimaryIPEndpoint() error {
	blockIDEncoded := base64.StdEncoding.EncodeToString([]byte(blockID))

	ipEndpoint, err := b.GetReplicaIPEndpoint()
	if err != nil {
		loggershim.LS.Errorf("Failed to get local IP address due to error: %s\n", err.Error())
		return err
	}

	putBlockOptions := &storage.PutBlockOptions{LeaseID: b.leaseID}
	err = b.blob.PutBlock(
		blockIDEncoded,
		[]byte(ipEndpoint),
		putBlockOptions)

	if err != nil {
		loggershim.LS.Errorf("PutBlock failed: %s\n", err.Error())
		return err
	}

	blocks := []storage.Block{
		storage.Block{ID: blockIDEncoded, Status: storage.BlockStatusUncommitted}}

	putBlockListOptions := &storage.PutBlockListOptions{
		LeaseID: b.leaseID}

	err = b.blob.PutBlockList(blocks, putBlockListOptions)
	if err != nil {
		loggershim.LS.Errorf("PutBlockList failed: %s\n", err.Error())
		return err
	}

	loggershim.LS.Debugf("Successfully updated the blob\n")
	b.replicaChangeNotificationCallback(true)
	b.primaryIPEndpointNotificationCallback(ipEndpoint)

	return nil
}

func (b *BlobStore) updateReplicaPrimaryEndpoint() {
	primaryIPEP, err := b.GetPrimaryIPEndpoint()
	if err != nil {
		loggershim.LS.Errorf("Failed to read the blob due to error %s\n", err.Error())
	} else {
		loggershim.LS.Debugf("Updating primary replica EP %s\n", primaryIPEP)
	}

	b.replicaChangeNotificationCallback(false)
	b.primaryIPEndpointNotificationCallback(primaryIPEP)
}

// TryAcquireLease - Function to acquire and renew lease.
func (b *BlobStore) TryAcquireLease() error {
	err := b.createBlob()
	if err != nil {
		loggershim.LS.Errorf("Create blob failed with error %s\n", err.Error())
		return err
	}

	updateBlobRetryInterval :=
		(time.Second) * time.Duration(b.blobStorageSettings.AcquireRetryIntervalInSecs/2)
	leaseAcquireRetryTimerChannel := time.NewTicker(
		time.Second * time.Duration(b.blobStorageSettings.AcquireRetryIntervalInSecs)).C
	leaseRenewRetryTimerChannel := time.NewTicker(
		time.Second * time.Duration(b.blobStorageSettings.LeaseIntervalInSecs)).C
	var leaseErr error

	for {
	AcquireLease:
		b.replicaChangeNotificationCallback(false)
		for {
			select {
			case <-leaseAcquireRetryTimerChannel:
				updateRetryCount := 0
				b.leaseID, leaseErr = b.blob.AcquireLease(
					int(b.blobStorageSettings.LeaseIntervalInSecs), "", nil)

				if leaseErr != nil {
					loggershim.LS.Errorf("Acquiring lease failed: %s\n", leaseErr.Error())
					b.updateReplicaPrimaryEndpoint()
					break
				}

				for updateRetryCount < 2 {
					err := b.UpdatePrimaryIPEndpoint()
					if err != nil {
						time.Sleep(updateBlobRetryInterval)
						updateRetryCount++
					} else {
						break
					}
				}

				goto AcquiredLease

			case <-b.startEvent:
				updateRetryCount := 0
				b.leaseID, leaseErr = b.blob.AcquireLease(
					int(b.blobStorageSettings.LeaseIntervalInSecs), "", nil)

				if leaseErr != nil {
					loggershim.LS.Errorf("Acquiring lease failed: %s\n", leaseErr.Error())
					b.updateReplicaPrimaryEndpoint()
					break
				}

				for updateRetryCount < 2 {
					err := b.UpdatePrimaryIPEndpoint()
					if err != nil {
						time.Sleep(updateBlobRetryInterval)
						updateRetryCount++
					} else {
						break
					}
				}

				goto AcquiredLease

			case <-b.cancelEvent:
				loggershim.LS.Debugf("Received cancel event\n")
				return nil
			}
		}

	AcquiredLease:
		for {
			select {
			case <-leaseRenewRetryTimerChannel:
				renewLeaseErr := b.blob.RenewLease(b.leaseID, nil)
				if renewLeaseErr != nil {
					loggershim.LS.Errorf("Acquiring lease failed: %s\n", renewLeaseErr.Error())
					b.replicaChangeNotificationCallback(false)
					goto AcquireLease
				} else {
					//loggershim.LS.Debugf("Successfully renewed the lease", leaseID)
				}
				primaryIPEndpoint, err := b.GetPrimaryIPEndpoint()
				if err != nil {
					b.primaryIPEndpointNotificationCallback(primaryIPEndpoint)
				}

				break
			case <-b.cancelEvent:
				loggershim.LS.Debugf("Received cancel event\n")
				return nil
			}
		}
	}
}

// GetPrimaryIPEndpoint - Method to get the primary replica IP.
func (b *BlobStore) GetPrimaryIPEndpoint() (string, error) {
	err := b.createBlob()
	if err != nil {
		loggershim.LS.Errorf("Create blob failed with error %s\n", err.Error())
		return "", err
	}

	readCloser, err := b.blob.Get(nil)
	if err != nil {
		return "", err
	}
	defer readCloser.Close()

	bytesRead, err := ioutil.ReadAll(readCloser)
	if err != nil {
		return "", err
	}

	return string(bytesRead[:]), nil
}

// GetReplicaIPEndpoint - Method to check if the given replica is primary.
func (b *BlobStore) GetReplicaIPEndpoint() (string, error) {
	replicaIP, err := b.getIPAddress()
	if err != nil {
		return "", err
	}

	return replicaIP + ":" + b.serverSettings.Port, nil

}

// IsPrimary - Method to check if the given replica is primary.
func (b *BlobStore) IsPrimary() (bool, error) {
	primaryEP, err := b.GetPrimaryIPEndpoint()
	loggershim.LS.Debugf("GetPrimaryIPEndpoint: %s\n", primaryEP)
	if err != nil {
		return false, err
	}

	replicaIP, err := b.getIPAddress()
	if err != nil {
		return false, err
	}

	replicaEP := replicaIP + ":" + b.serverSettings.Port
	return strings.Compare(primaryEP, replicaEP) == 0, nil
}

// SetDataStoreSettings - Function to set account name, key and partition key
func (b *BlobStore) SetDataStoreSettings(config *configuration.Configuration) {
	b.mutex.Lock()
	b.blobStorageSettings.AccountName = config.DataStoreSettings.AzureStorageInfo.AccountName
	b.blobStorageSettings.Key = config.DataStoreSettings.AzureStorageInfo.Key
	b.blobStorageSettings.PartitionKey = config.DataStoreSettings.AzureStorageInfo.PartitionKey
	b.mutex.Unlock()
}

// getDataStoreSettings - Function that return datastore settings (AccountName, Key, PartitionKey)
func (b *BlobStore) getDataStoreSettings() (string, string, string) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.blobStorageSettings.AccountName, b.blobStorageSettings.Key, b.blobStorageSettings.PartitionKey
}

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

func getEnvVarOrExit(varName string) string {
	value := os.Getenv(varName)
	if value == "" {
		loggershim.LS.Errorf("Missing environment variable %s\n", varName)
		os.Exit(1)
	}

	return value
}

// onErrorFail prints a failure message and exits the program if err is not nil.
func onErrorFail(err error, message string) {
	if err != nil {
		loggershim.LS.Errorf("%s: %s\n", message, err.Error())
		os.Exit(1)
	}
}
