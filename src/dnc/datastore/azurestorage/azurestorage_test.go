package azurestorage

import (
	"strconv"
	"testing"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSliceEntities(t *testing.T) {
	tests := []struct {
		name            string
		entitySize      int
		batchSize       int
		expectedBatches int
	}{
		{
			name:            "entity size below limit",
			entitySize:      90,
			batchSize:       100,
			expectedBatches: 1,
		},
		{
			name:            "entity size same as limit",
			entitySize:      100,
			batchSize:       100,
			expectedBatches: 1,
		},
		{
			name:            "entity size over limit",
			entitySize:      101,
			batchSize:       100,
			expectedBatches: 2,
		},
		{
			name:            "entity size exact multiple of limit",
			entitySize:      500,
			batchSize:       100,
			expectedBatches: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities := generateEntities(tt.entitySize)
			batches := sliceEntities(entities, tt.batchSize)
			assert.Len(t, batches, tt.expectedBatches)
			entityCount := 0
			for _, batch := range batches {
				for _, entity := range batch {
					require.Equal(t, strconv.Itoa(entityCount), entity.RowKey)
					entityCount++
				}
			}
			require.Equal(t, tt.entitySize, entityCount)
		})
	}
}

func generateEntities(count int) []storage.Entity {
	entities := make([]storage.Entity, count)
	for i := 0; i < len(entities); i++ {
		entities[i].RowKey = strconv.Itoa(i)
	}
	return entities
}
