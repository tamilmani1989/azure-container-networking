package resourcecleanupmonitor

import (
	"dnc/datamodel"
	"dnc/datastore/testpersistencestore"
	"dnc/logger/loggermock"
	"dnc/networkcontainer"
	"dnc/networkcontainer/fakes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	"github.com/Azure/azure-sdk-for-go/storage"
)

const (
	testNCID         = "testncid1"
	testNetworkID    = "testnet1"
	testPubSubAZID   = "disabled"
	testSubnetName   = "subnet1"
	ipSubnet         = "172.16.0.0/24"
	testPartitionKey = "testpartitionkey"
	testEtag         = "testetag"
)

var (
	logger = loggermock.NewDummyLogger()
)

func getTestDataStore() *testpersistencestore.TestPersistenceStore {
	data := testpersistencestore.TestPersistenceStore{}
	data.Init(datamodel.TableToColumnMap, loggermock.NewDummyLogger())
	return &data
}

// TODO: finish integration test with ncpendingpublish and ipallocation tables
func TestResourceCleanupMonitor_ProcessResourceCleanup(t *testing.T) {
	type fields struct {
		IsPrimary                       bool
		ipAllocator                     *networkcontainer.IPAllocator
		ncPendingPublishRequestContexts []datamodel.NcPendingPublishRequestContext
		cleanupContexts                 []datamodel.PendingResourceCleanupEntity
		ncid                            string //todo: convert the above contexts to maps with rowkey to tune, multi-nc cleanupcontexts
		networkid                       string
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "Skip IP deallocation when there exists a job with NC ID",
			fields: fields{
				IsPrimary: true,
				ipAllocator: &networkcontainer.IPAllocator{
					Nodes: nil,
					NCs:   nil,
					Subnets: &fakes.SubnetRepo{
						Data: map[string]datamodel.SubnetConfiguration{
							datamodel.GetSubnetKey("network1", "subnet1"): {
								NetworkID:  testNCID,
								SubnetName: testSubnetName,
								IPSubnet:   ipSubnet,
							},
						},
						QueryErr: nil,
					},
					Reservations: nil,
					Allocations: &fakes.AllocationRepo{
						Data: []datamodel.IPAllocationEntity{
							{RowKey: "192.168.0.1_32", IPAllocation: "", AllocationID: "uuid0", ToBeDeletedOwner: datamodel.GetNCKey(testNetworkID, testNCID)},
							{RowKey: "192.168.0.2_32", IPAllocation: "", AllocationID: "uuid1", ToBeDeletedOwner: datamodel.GetNCKey(testNetworkID, testNCID)},
							{RowKey: "172.16.0.20_32", IPAllocation: "Network_network1_Reservation_res2", AllocationID: "uuid2"},
							{RowKey: "172.16.0.21_32", IPAllocation: "Network_network1_Reservation_res2", AllocationID: "uuid3"},
						},
						MaxBatchSize: 100,
					},
					Logger:       logger,
					MaxBatchSize: 0,
				},
				ncPendingPublishRequestContexts: []datamodel.NcPendingPublishRequestContext{
					{
						Type:               datamodel.NetworkContainerUnpublishRequest,
						NetworkContainerID: testNCID,
						NetworkID:          testNetworkID,
						PubsubAzId:         testPubSubAZID,
					},
				},

				cleanupContexts: []datamodel.PendingResourceCleanupEntity{
					{
						RowKey: datamodel.GetPendingResourceCleanupKey(datamodel.IPCleanup, testNetworkID, testNCID),
						CleanupContext: datamodel.CleanupContext{
							Type: datamodel.IPCleanup,
							ToBeDeletedIpAllocations: map[string][]string{
								datamodel.GetSubnetKey(testNetworkID, testSubnetName): {"192.168.0.1_32", "192.168.0.2_32"},
							},
						},
						Etag: testEtag,
					},
				},

				ncid:      testNCID,
				networkid: testNetworkID,
			},
		},
		{
			name: "Skip IP deallocation when there IS NO pending NC job",
			fields: fields{
				IsPrimary: true,
				ipAllocator: &networkcontainer.IPAllocator{
					Nodes: nil,
					NCs:   nil,
					Subnets: &fakes.SubnetRepo{
						Data: map[string]datamodel.SubnetConfiguration{
							datamodel.GetSubnetKey(testNetworkID, testSubnetName): {
								NetworkID:  testNCID,
								SubnetName: testSubnetName,
								IPSubnet:   ipSubnet,
							},
						},
						QueryErr: nil,
					},
					Reservations: nil,
					Allocations: &fakes.AllocationRepo{
						Data: []datamodel.IPAllocationEntity{
							{RowKey: "192.168.0.1_32", IPAllocation: "", AllocationID: "uuid0", ToBeDeletedOwner: datamodel.GetNCKey(testNetworkID, testNCID)},
							{RowKey: "192.168.0.2_32", IPAllocation: "", AllocationID: "uuid1", ToBeDeletedOwner: datamodel.GetNCKey(testNetworkID, testNCID)},
							{RowKey: "172.16.0.20_32", IPAllocation: "Network_network1_Reservation_res2", AllocationID: "uuid2"},
							{RowKey: "172.16.0.21_32", IPAllocation: "Network_network1_Reservation_res2", AllocationID: "uuid3"}},
						MaxBatchSize: 100},
					Logger:       logger,
					MaxBatchSize: 0,
				},
				ncPendingPublishRequestContexts: []datamodel.NcPendingPublishRequestContext{},
				cleanupContexts: []datamodel.PendingResourceCleanupEntity{
					{
						RowKey: datamodel.GetPendingResourceCleanupKey(datamodel.IPCleanup, testNetworkID, testNCID),
						CleanupContext: datamodel.CleanupContext{
							Type: datamodel.IPCleanup,
							ToBeDeletedIpAllocations: map[string][]string{
								datamodel.GetSubnetKey(testNetworkID, testSubnetName): {"192.168.0.1_32", "192.168.0.2_32"},
							},
						},
						Etag: testEtag,
					},
				},
				ncid:      testNCID,
				networkid: testNetworkID,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDataStore := getTestDataStore()
			hydrateTestNCPendingPublishRepo(testDataStore, tt.fields.ncid, tt.fields.networkid, tt.fields.ncPendingPublishRequestContexts, tt.fields.cleanupContexts)
			datamodel.NcPendingPublishRepo = datamodel.NewNcPendingPublishRequestRepo(testDataStore)
			c := &ResourceCleanupMonitor{
				PendingCleanupRepo: datamodel.NewPendingResourceCleanupRepo(testDataStore, loggermock.NewDummyLogger()),
				Logger:             logger,
				IsPrimary:          tt.fields.IsPrimary,
				IpAllocator:        tt.fields.ipAllocator,
			}
			for _, cleanupRow := range tt.fields.cleanupContexts {
				entry, err := c.PendingCleanupRepo.TryGetPendingCleanupRequest(cleanupRow.RowKey)
				require.Nil(t, err)

				err = c.processCleanupResource(*entry)
				assert.Nil(t, err)
			}
			//Todo: verify ipcleanup has been called when NC's are unpublished`
		})
	}
}

func hydrateTestNCPendingPublishRepo(testDataStore *testpersistencestore.TestPersistenceStore, ncid string, networkid string, pendingPublishContexts []datamodel.NcPendingPublishRequestContext, pendingResourceCleanupEntities []datamodel.PendingResourceCleanupEntity) {
	// seed NC pending publish
	pubsubEntities := []storage.Entity{}
	for _, pendingPublish := range pendingPublishContexts {
		b, _ := json.Marshal(pendingPublish)
		entity := storage.Entity{
			RowKey: datamodel.GetNcPendingPublishContextKey(pendingPublish.NetworkID, pendingPublish.NetworkContainerID, pendingPublish.PubsubAzId),
			Properties: map[string]interface{}{
				datamodel.NcPendingPublishContextTag: string(b[:]),
			},
		}
		pubsubEntities = append(pubsubEntities, entity)
	}
	testDataStore.Seed(map[string][]storage.Entity{datamodel.NcPendingPublishContextTableName: pubsubEntities})

	// seed preparecleanup
	cleanupEntities := []storage.Entity{}
	for _, pendingResourceEntity := range pendingResourceCleanupEntities {
		b, _ := json.Marshal(pendingResourceEntity.CleanupContext)
		entity := storage.Entity{
			RowKey: pendingResourceEntity.RowKey,
			Properties: map[string]interface{}{
				datamodel.PendingResourceCleanupTag: string(b[:]),
			},
			OdataEtag: testEtag,
		}
		cleanupEntities = append(cleanupEntities, entity)
	}
	testDataStore.Seed(map[string][]storage.Entity{datamodel.PendingResourceCleanupTableName: cleanupEntities})
}
