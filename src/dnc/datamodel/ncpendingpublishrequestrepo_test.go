package datamodel

import (
	"dnc/api"
	"dnc/datastore/testpersistencestore"
	"dnc/logger/loggermock"
	"encoding/json"
	"testing"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/stretchr/testify/assert"
)

const (
	testNCID1       = "3bddcadf-636e-4534-a0e5-07c878ec3d2c"
	testNetworkID1  = "a20e88f2-0200-43b3-8ccc-748b9ca8d6f8"
	testPubSubAZID1 = "1"
)

func TestNcPendingPublishRequestRepo_TryFilterNcPendingPublish(t *testing.T) {
	type fields struct {
		tableData                            []NcPendingPublishRequestContext
		wantedNcPendingPublishRequestContext *NcPendingPublishRequestContext
		expectedErr                          *api.JsonErr
	}
	type args struct {
		key string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Retrieve all in specified partition",
			fields: fields{
				wantedNcPendingPublishRequestContext: &NcPendingPublishRequestContext{
					Type:               NetworkContainerUnpublishRequest,
					NetworkContainerID: testNCID1,
					NetworkID:          testNetworkID1,
					PubsubAzId:         testPubSubAZID1,
				},
				tableData: []NcPendingPublishRequestContext{
					{
						Type:               NetworkContainerUnpublishRequest,
						NetworkContainerID: testNCID1,
						NetworkID:          testNetworkID1,
						PubsubAzId:         testPubSubAZID1,
					},
				},
				expectedErr: nil,
			},

			args: args{
				key: GetNcPendingPublishContextKey(testNetworkID1, testNCID1, testPubSubAZID1),
			},
		},
		{
			name: "Retrieve when there's none pending",
			fields: fields{
				tableData:   []NcPendingPublishRequestContext{},
				expectedErr: nil,
			},
			args: args{
				key: GetNcPendingPublishContextKey(testNetworkID1, testNCID1, testPubSubAZID1),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDataStore := getTestDataStore()
			hydrateTestNCPendingPublishRepo(testDataStore, tt.fields.tableData)
			n := &NcPendingPublishRequestRepo{
				data: testDataStore,
			}
			// TODO: testpersistencestore datastoreshim currently returns all on any QueryFilter,
			// need to complete that implementation before this test is complete
			pendingPublishContexts, err := n.TryGetNcPendingPublish(tt.args.key)
			assert.Equal(t, tt.fields.expectedErr, err)
			assert.Exactly(t, pendingPublishContexts, tt.fields.wantedNcPendingPublishRequestContext)
		})
	}
}

func getTestDataStore() *testpersistencestore.TestPersistenceStore {
	d := testpersistencestore.TestPersistenceStore{}
	d.Init(TableToColumnMap, loggermock.NewDummyLogger())
	return &d
}

func hydrateTestNCPendingPublishRepo(testDataStore *testpersistencestore.TestPersistenceStore, pendingPublishContexts []NcPendingPublishRequestContext) {
	var entities []storage.Entity
	for _, pendingPublish := range pendingPublishContexts {
		b, _ := json.Marshal(pendingPublish)
		entity := storage.Entity{
			RowKey: GetNcPendingPublishContextKey(pendingPublish.NetworkID, pendingPublish.NetworkContainerID, pendingPublish.PubsubAzId),
			Properties: map[string]interface{}{
				NcPendingPublishContextTag: string(b[:]),
			},
		}
		entities = append(entities, entity)
	}

	testDataStore.Seed(map[string][]storage.Entity{NcPendingPublishContextTableName: entities})
}
