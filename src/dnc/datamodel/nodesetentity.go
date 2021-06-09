package datamodel

import (
	"dnc/api"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/storage"
)

// These are properties of the cosmos/storage table for node sets.
const (
	NodeSetsTableName               = "NodeSetsTable"
	NodeSetNameProperty             = "NodeSetName"
	NodeSetStateProperty            = "State"
	NodeSetStatusProperty           = "Status"
	NodeSetMarkedForDeleteProperty  = "MarkedForDelete"
	NodeSetFailureErrorCodeProperty = "FailureErrorCode"
	NodeSetFailureErrorMsgProperty  = "FailureErrorMsg"
	NodeSetFailureHTTPCodeProperty  = "FailureHttpCode"
)

// A NodeSetEntity represents a node set record in the NodeSetsTable.
type NodeSetEntity struct {
	// PartitionKey is the partition key to which this entity belongs.
	PartitionKey string
	// RowKey is a unique key within a partition. Encoded as "NodeSet_<node set name>".
	RowKey string
	// ETag is a metadata property used for optimistic locking.
	ETag string
	// NodeSetName is the name of the node set.
	NodeSetName string
	// State represents the processing stage of a node set.
	State NodeSetState
	// Status represents the status of the processing stage.
	Status Status
	// FailureDetail captures any errors during processing.
	FailureDetail api.JsonErr
	// MarkedForDelete is a flag that signals if the node set is being deleted.
	MarkedForDelete bool
}

const nodeSetKeyFmt = "NodeSet_%s"

// EncodeNodeSetRowKey creates a node set row key from a node set name.
func EncodeNodeSetRowKey(nodeSetName string) string {
	return fmt.Sprintf(nodeSetKeyFmt, nodeSetName)
}

// ToStorageEntity converts the NodeSetEntity to a storage.Entity, suitable for db operations.
func (n *NodeSetEntity) ToStorageEntity() storage.Entity {
	return storage.Entity{
		Table:        &storage.Table{Name: NodeSetsTableName},
		PartitionKey: n.PartitionKey,
		RowKey:       n.RowKey,
		OdataEtag:    n.ETag,
		Properties: map[string]interface{}{
			NodeSetNameProperty:             n.NodeSetName,
			NodeSetStateProperty:            n.State.String(),
			NodeSetStatusProperty:           n.Status.String(),
			NodeSetMarkedForDeleteProperty:  n.MarkedForDelete,
			NodeSetFailureErrorCodeProperty: int64(n.FailureDetail.Code),
			NodeSetFailureErrorMsgProperty:  n.FailureDetail.Text,
			NodeSetFailureHTTPCodeProperty:  int64(n.FailureDetail.HTTPStatus),
		},
	}
}

func storageEntityToNodeSetEntity(e *storage.Entity) (NodeSetEntity, error) {
	state := ExtractStringProperty(e, NodeSetStateProperty)
	nodeSetState := stringToNodeSetState(state)

	status := ExtractStringProperty(e, NodeSetStatusProperty)
	nodeSetStatus := StringToStatus(status)

	errorCode, err := ExtractInt64Property(e, NodeSetFailureErrorCodeProperty)
	if err != nil {
		return NodeSetEntity{}, err
	}

	errorHTTPCode, err := ExtractInt64Property(e, NodeSetFailureHTTPCodeProperty)
	if err != nil {
		return NodeSetEntity{}, err
	}

	markedForDelete, err := ExtractBoolProperty(e, NodeSetMarkedForDeleteProperty)
	if err != nil {
		return NodeSetEntity{}, err
	}

	return NodeSetEntity{
		PartitionKey: e.PartitionKey,
		RowKey:       e.RowKey,
		ETag:         e.OdataEtag,
		NodeSetName:  ExtractStringProperty(e, NodeSetNameProperty),
		State:        nodeSetState,
		Status:       nodeSetStatus,
		FailureDetail: api.JsonErr{
			Code:       api.ErrorCode(errorCode),
			Text:       ExtractStringProperty(e, NodeSetFailureErrorMsgProperty),
			HTTPStatus: int(errorHTTPCode),
		},
		MarkedForDelete: markedForDelete,
	}, nil
}

// NodeSetState represents possible processing states for the node set.
type NodeSetState int

// possible processing states for the node set.
const (
	NodeSetStateUndefined NodeSetState = iota
	NodeSetCreate
	NodeSetDelete
)

func (n NodeSetState) String() string {
	switch n {
	case NodeSetCreate:
		return "Create"
	case NodeSetDelete:
		return "Delete"
	}
	return "Undefined"
}

func stringToNodeSetState(s string) NodeSetState {
	switch strings.ToLower(s) {
	case "create":
		return NodeSetCreate
	case "delete":
		return NodeSetDelete
	}
	return NodeSetStateUndefined
}
