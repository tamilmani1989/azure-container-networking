//+build dbintegration

package datamodel

import (
	"dnc/api"
	"dnc/configuration"
	"dnc/datastore/azurestorage"
	"dnc/logger/loggermock"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNodeSetRepo(t *testing.T) {
	partitionKey := "integrationtest"

	cfg := configuration.AzureStorageInfo{
		AccountName:     "",
		Key:             "",
		PartitionKey:    partitionKey,
		IsUsingCosmosDb: true,
	}

	tablesToEnsure := map[string]string{NodeSetsTableName: ""}

	ds, err := azurestorage.NewAzureStorage(cfg, loggermock.NewDummyLogger(), tablesToEnsure, 5000, 1000)
	require.NoError(t, err)

	repo := NodeSetRepo{Datastore: ds, PartitionKey: partitionKey}

	nodeSetName := fmt.Sprintf("nodeset_%s", time.Now().Format("2006-01-02-150405"))
	entity := NodeSetEntity{
		PartitionKey: partitionKey,
		RowKey:       EncodeNodeSetRowKey(nodeSetName),
		NodeSetName:  nodeSetName,
		State:        NodeSetDelete,
		Status:       Pending,
		FailureDetail: api.JsonErr{
			Code:       10,
			Text:       "some error",
			HTTPStatus: http.StatusInternalServerError,
		},
		MarkedForDelete: true,
	}

	t.Run("node set insertion", func(t *testing.T) {
		apiErr := repo.InsertNodeSet(entity)
		assert.Nil(t, apiErr)

		t.Run("409 conflict when entity exists", func(t *testing.T) {
			apiErr := repo.InsertNodeSet(entity)
			require.NotNil(t, apiErr)
			assert.Equal(t, http.StatusConflict, apiErr.HTTPStatus)
		})
	})

	t.Run("node set query", func(t *testing.T) {
		t.Run("exists", func(t *testing.T) {
			nse, apiErr := repo.TryQueryNodeSet(nodeSetName)
			require.Nil(t, apiErr)
			require.NotNil(t, nse)

			assert.Equal(t, entity.PartitionKey, nse.PartitionKey)
			assert.Equal(t, entity.RowKey, nse.RowKey)
			assert.Equal(t, entity.NodeSetName, nse.NodeSetName)
			assert.Equal(t, entity.State, nse.State)
			assert.Equal(t, entity.Status, nse.Status)
			assert.Equal(t, entity.FailureDetail, nse.FailureDetail)
			assert.Equal(t, entity.MarkedForDelete, nse.MarkedForDelete)
		})

		t.Run("does not exist", func(t *testing.T) {
			nse, apiErr := repo.TryQueryNodeSet(nodeSetName + "foo")
			require.Nil(t, apiErr)
			require.Nil(t, nse)
		})
	})

	t.Run("node set delete", func(t *testing.T) {
		apiErr := repo.DeleteNodeSet(nodeSetName)
		require.Nil(t, apiErr)

		t.Run("no error when deleting a node set that doesn't exist", func(t *testing.T) {
			apiErr := repo.DeleteNodeSet(nodeSetName)
			require.Nil(t, apiErr)
		})
	})
}
