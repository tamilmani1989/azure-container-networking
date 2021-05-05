package datastore

import (
	"net/http"

	"github.com/Azure/azure-sdk-for-go/storage"
)

type datastoreClient interface {
	QueryAllRowsInPartition(tableName, PartitionKey string, pageToken interface{}) ([]*storage.Entity, bool, interface{}, int, error)
	QueryInPartition(tableName string, partitionKey string, rowKey string) (*storage.Entity, int, error)
}

type mockDatastoreClient struct {
	rows map[string]*storage.Entity
}

func (m *mockDatastoreClient) QueryAllRowsInPartition(tableName, PartitionKey string, pageToken interface{}) ([]*storage.Entity, bool, interface{}, int, error) {
	vals := make([]*storage.Entity, len(m.rows))

	i := 0
	for _, val := range m.rows {
		vals[i] = val
		i++
	}

	return vals, false, struct{}{}, http.StatusOK, nil
}

func (m *mockDatastoreClient) QueryInPartition(tableName string, partitionKey string, rowKey string) (*storage.Entity, int, error) {
	if val, found := m.rows[rowKey]; found {
		return val, http.StatusOK, nil
	}

	return nil, http.StatusNotFound, nil
}
