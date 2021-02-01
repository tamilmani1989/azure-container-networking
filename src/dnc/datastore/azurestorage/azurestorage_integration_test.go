// +build integration

package azurestorage

import (
	"dnc/configuration"
	"dnc/logger/loggermock"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
  This test suite is currently intended for local integration testing against a cosmos or storage db,
  and thus guarded with build tags.
  todo:
   - use appropriate build tags when pr pipeline is updated
   - use env vars set by pipeline for db info and credentials
   - expand coverage to other methods
*/

// test table to play with API operations. assume that it can be dropped at any time.
const testTable = "TestTable"

var client *AzureStorage

func TestMain(m *testing.M) {
	cfg := configuration.AzureStorageInfo{
		AccountName:     "",
		Key:             "",
		PartitionKey:    "",
		IsUsingCosmosDb: true,
	}

	tableInit := map[string]string{testTable: ""}

	cl, err := NewAzureStorage(cfg, loggermock.NewDummyLogger(), tableInit, 5000, 1000)
	if err != nil {
		log.Fatalf("could not initialize client for integration test: %v", err)
	}
	client = cl

	os.Exit(m.Run())
}

func TestBatch(t *testing.T) {
	partitionKey := "batch_operations_test_" + uuid.New().String()

	t.Logf("Starting batch operations test on partition %s", partitionKey)

	entities := generateEntities(testTable, partitionKey, 100)
	var keysToDelete []string
	for _, entity := range entities {
		keysToDelete = append(keysToDelete, entity.RowKey)
	}

	t.Run("insertion of new entities", func(t *testing.T) {
		err := client.InsertEntityBatch(testTable, entities)
		require.NoError(t, err)
	})

	t.Run("insertion of existing entities should fail with conflict", func(t *testing.T) {
		err := client.InsertEntityBatch(testTable, entities)
		assert.Error(t, err)
		tableAPIErr, ok := err.(storage.AzureStorageServiceError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusConflict, tableAPIErr.StatusCode)
	})

	var updatedEntities []storage.Entity

	t.Run("update of existing entities with etags", func(t *testing.T) {
		storedEntities, err := client.QueryFilterf(testTable, "PartitionKey eq '%s'", partitionKey)
		require.NoError(t, err)
		require.NotEmpty(t, storedEntities)

		for _, se := range storedEntities {
			ue := *se
			if ue.Properties == nil {
				ue.Properties = make(map[string]interface{})
			}
			ue.Properties["x"] = "y"
			updatedEntities = append(updatedEntities, ue)
		}

		err = client.UpdateEntityBatch(testTable, updatedEntities)
		assert.NoError(t, err)
	})

	t.Run("update of existing entities with expired etags should fail with precondition failed", func(t *testing.T) {
		err := client.UpdateEntityBatch(testTable, updatedEntities)
		assert.Error(t, err)
		tableAPIErr, ok := err.(storage.AzureStorageServiceError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusPreconditionFailed, tableAPIErr.StatusCode)
	})

	t.Run("deletion of entities", func(t *testing.T) {
		err := client.DeleteBatch(testTable, partitionKey, keysToDelete)
		assert.NoError(t, err)
	})
}

func generateEntities(tableName, partitionKey string, count int) []storage.Entity {
	var out []storage.Entity
	for i := 0; i < count; i++ {
		e := storage.Entity{
			Table:        &storage.Table{Name: tableName},
			PartitionKey: partitionKey,
			RowKey:       uuid.New().String(),
		}
		out = append(out, e)
	}

	return out
}
