package datamodel

import (
	"dnc/api"
	"dnc/datastoreshim"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/storage"
)

// NodeSetRepo performs CRUD operations on the NodeSetsTable.
type NodeSetRepo struct {
	Datastore    datastoreshim.DataStoreShim
	PartitionKey string
}

// TryQueryNodeSet retrieves a node set or returns nil if not found.
func (n *NodeSetRepo) TryQueryNodeSet(nodeSetName string) (*NodeSetEntity, *api.JsonErr) {
	rowKey := EncodeNodeSetRowKey(nodeSetName)
	entity, status, err := n.Datastore.QueryInPartition(NodeSetsTableName, n.PartitionKey, rowKey)
	if err != nil {
		if status == http.StatusNotFound {
			return nil, nil
		}

		return nil, api.InternalServerErrorf(api.StatusInternalServerError, "could not retrieve node set %s: %v", nodeSetName, err)
	}

	nse, err := storageEntityToNodeSetEntity(entity)
	if err != nil {
		msg := fmt.Sprintf("could not convert entity on partition key %s row key %s to node set entity: %v", entity.PartitionKey, entity.RowKey, err)
		return nil, api.InternalServerError(api.StatusInternalServerError, msg)
	}

	return &nse, nil
}

// InsertNodeSet inserts a node set into the repo.
func (n *NodeSetRepo) InsertNodeSet(nse NodeSetEntity) *api.JsonErr {
	e := nse.ToStorageEntity()

	if err := n.Datastore.InsertEntity(NodeSetsTableName, e); err != nil {
		switch e := err.(type) {
		case storage.AzureStorageServiceError:
			return &api.JsonErr{Code: api.AzureStorageError, Text: e.Message, HTTPStatus: e.StatusCode}
		default:
			return api.InternalServerErrorf(api.StatusInternalServerError, "could not insert node set %s in partition %s: %v", nse.NodeSetName, nse.PartitionKey, e)
		}
	}

	return nil
}

// DeleteNodeSet removes a node set from the repo.
func (n *NodeSetRepo) DeleteNodeSet(nodeSetName string) *api.JsonErr {
	rowKey := EncodeNodeSetRowKey(nodeSetName)
	if err := n.Datastore.DeleteInPartition(NodeSetsTableName, n.PartitionKey, rowKey); err != nil {
		switch e := err.(type) {
		case storage.AzureStorageServiceError:
			if e.StatusCode == http.StatusNotFound {
				return nil
			}
			return &api.JsonErr{Code: api.AzureStorageError, Text: e.Message, HTTPStatus: e.StatusCode}
		default:
			return api.InternalServerErrorf(api.StatusInternalServerError, "could not delete node set %s in partition %s: %v", nodeSetName, n.PartitionKey, e)
		}
	}

	return nil
}
