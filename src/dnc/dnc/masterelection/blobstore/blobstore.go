package blobstore

import (
	"dnc/configuration"
	"dnc/masterelection/common"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"

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
	return blobStore
}

// Init - Initialize blob storage for primary election.
func (b *BlobStore) Init() {
	client, err := storage.NewBasicClient(b.blobStorageSettings.AccountName, b.blobStorageSettings.Key)
	onErrorFail(err, "Create client failed")
	b.blobClient = client.GetBlobService()
}

// Cleanup - Cleanup of blob store.
func (b *BlobStore) Cleanup() {
	b.cancelEvent <- true
	glog.Info("Try to release lease")
	b.blob.ReleaseLease(b.leaseID, nil)
}

// Start - Start the leader election process
func (b *BlobStore) Start() {
	b.TryAcquireLease()
}

func (b *BlobStore) createBlob() error {
	cnt := b.blobClient.GetContainerReference(b.blobStorageSettings.ContainerName)
	options := storage.CreateContainerOptions{
		Access: storage.ContainerAccessTypePrivate,
	}
	_, err := cnt.CreateIfNotExists(&options)
	if err != nil {
		onErrorFail(err, "Create container failed")
	}

	b.blob = cnt.GetBlobReference(b.blobStorageSettings.BlobName)
	blobExist, checkBlobErr := b.blob.Exists()
	if checkBlobErr != nil {
		glog.Error("Check if blob exist failed:", checkBlobErr)
		return fmt.Errorf("create block blob failed as blob exist check failed: %v", checkBlobErr)
	}

	if !blobExist {
		err = b.blob.CreateBlockBlob(nil)
		if err != nil {
			storagErr, ok := err.(storage.AzureStorageServiceError)
			if ok {
				if storagErr.StatusCode == 412 {
					log.Println("Blob exist and there exist as lease on the blob", storagErr.StatusCode)
				}
			} else {
				log.Println("create block blob failed:", err)
				return fmt.Errorf("create block blob failed: %v", err)
			}
		}
		log.Println("Created blob ", b.blobStorageSettings.BlobName)
	}
	return err
}

func (b *BlobStore) getIPAddress() (string, error) {
	ipAddress := b.serverSettings.IPAddress
	var err error
	if ipAddress == "" {
		ipAddress, err = getLocalIP()
		if err != nil {
			log.Println("Failed to retrieve IP Address")
			return "", err
		}
	}

	return ipAddress, nil
}

func (b *BlobStore) UpdatePrimaryIPEndpoint() error {
	blockIDEncoded := base64.StdEncoding.EncodeToString([]byte(blockID))
	ipAddress, err := getLocalIP()
	if err != nil {
		log.Println("Failed to get local IP address due to error", err)
		return err
	}
	ipEndpoint := ipAddress + ":" + b.serverSettings.Port

	putBlockOptions := &storage.PutBlockOptions{LeaseID: b.leaseID}
	err = b.blob.PutBlock(
		blockIDEncoded,
		[]byte(ipEndpoint),
		putBlockOptions)

	if err != nil {
		log.Println("PutBlock failed:", err)
		return err
	}

	blocks := []storage.Block{
		storage.Block{ID: blockIDEncoded, Status: storage.BlockStatusUncommitted}}

	putBlockListOptions := &storage.PutBlockListOptions{
		LeaseID: b.leaseID}

	err = b.blob.PutBlockList(blocks, putBlockListOptions)
	if err != nil {
		log.Println("PutBlockList failed:", err)
		return err
	}

	log.Println("Successfully updated the blob")
	b.replicaChangeNotificationCallback(true)
	b.primaryIPEndpointNotificationCallback(ipEndpoint)

	return nil
}

func (b *BlobStore) updateReplicaPrimaryEndpoint() {
	primaryIPEP, err := b.GetPrimaryIPEndpoint()
	if err != nil {
		glog.Errorf("Failed to read the blob due to error %s", err.Error())
	} else {
		glog.Errorf("Updating primary replica EP %s", primaryIPEP)
	}
	b.replicaChangeNotificationCallback(false)
	b.primaryIPEndpointNotificationCallback(primaryIPEP)

}

// TryAcquireLease - Function to acquire and renew lease.
func (b *BlobStore) TryAcquireLease() error {
	err := b.createBlob()
	if err != nil {
		log.Println("Create blob failed with error", err)
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
					log.Println("Acquiring lease failed:", leaseErr)
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
					log.Println("Acquiring lease failed:", leaseErr)
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
				log.Println("Received cancel event")
				return nil
			}
		}

	AcquiredLease:
		for {
			select {
			case <-leaseRenewRetryTimerChannel:
				renewLeaseErr := b.blob.RenewLease(b.leaseID, nil)
				if renewLeaseErr != nil {
					log.Println("Acquiring lease failed:", renewLeaseErr)
					b.replicaChangeNotificationCallback(false)
					goto AcquireLease
				} else {
					//log.Println("Successfully renewed the lease", leaseID)
				}
				primaryIPEndpoint, err := b.GetPrimaryIPEndpoint()
				if err != nil {
					b.primaryIPEndpointNotificationCallback(primaryIPEndpoint)
				}

				break
			case <-b.cancelEvent:
				log.Println("Received cancel event")
				return nil
			}
		}
	}
}

// GetPrimaryIPEndpoint - Method to get the primary replica IP.
func (b *BlobStore) GetPrimaryIPEndpoint() (string, error) {
	err := b.createBlob()
	if err != nil {
		log.Println("Create blob failed with error", err)
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
	log.Printf("GetPrimaryIPEndpoint: %s", primaryEP)
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
		fmt.Printf("Missing environment variable %s\n", varName)
		os.Exit(1)
	}

	return value
}

// onErrorFail prints a failure message and exits the program if err is not nil.
func onErrorFail(err error, message string) {
	if err != nil {
		fmt.Printf("%s: %s\n", message, err)
		os.Exit(1)
	}
}
