package datamodel

import (
	"dnc/api"
	"dnc/datastore/testpersistencestore"
	"dnc/logger/loggermock"
	"encoding/json"
	"testing"
)

func TestQueryNodesInNodeSet(t *testing.T) {
	var store testpersistencestore.TestPersistenceStore
	store.Init(TableToColumnMap, loggermock.NewDummyLogger())

	partition := "test"
	node1 := NodeConfiguration{
		NodeID: "node1",
		NodeInfo: api.NodeInfoRequest{
			IPAddresses:           []string{"10.0.0.4"},
			OrchestratorType:      "KubernetesCRD",
			InfrastructureNetwork: "576d174c-57b8-4688-9d53-1b74caa642ca",
		},
	}
	node1Serialized, err := json.Marshal(node1)
	if err != nil {
		t.Fatal(err)
	}

	nodeSet := "set1"
	node2 := NodeConfiguration{
		NodeID: "node2",
		NodeInfo: api.NodeInfoRequest{
			IPAddresses:           []string{"10.0.0.5"},
			OrchestratorType:      "KubernetesCRD",
			InfrastructureNetwork: "576d174c-57b8-4688-9d53-1b74caa642ca",
			NodeSet:               nodeSet,
		},
	}
	node2Serialized, err := json.Marshal(node2)
	if err != nil {
		t.Fatal(err)
	}

	// Insert node1 into db with no nodeSet
	if _, err := store.InsertInPartition(
		nodeTableName,
		partition,
		GetNodeKey(node1.NodeID),
		TableToColumnMap[nodeTableName],
		string(node1Serialized),
	); err != nil {
		t.Fatal(err)
	}

	// Insert node2 into db with nodeSet=set1
	if _, err := store.InsertInPartitionProperties(
		nodeTableName,
		partition,
		GetNodeKey(node2.NodeID),
		map[string]interface{}{
			TableToColumnMap[nodeTableName]: string(node2Serialized),
			NodeSetKeyProperty:              nodeSet,
		},
	); err != nil {
		t.Fatal(err)
	}

	repo := NodeConfigurationRepo{Datastore: &store}

	// Query all nodes, should return both node1 and node2
	var nodeConfigurations []NodeConfiguration
	var apiErr *api.JsonErr

	if nodeConfigurations, apiErr = repo.TryQueryAllNodeConfigurations(""); apiErr != nil {
		t.Fatalf("unexpected error: %v", apiErr)
	}

	if len(nodeConfigurations) != 2 {
		t.Fatalf("Expected 2 nodes, got %d", len(nodeConfigurations))
	}

	// datastore will use this predicate when performing QueryFilter.
	store.QueryFilterFunc = func(rowKey string, properties map[string]interface{}) bool {
		val := properties[NodeSetKeyProperty]
		return val == nodeSet
	}

	// Query all nodes in set1, should return node2 only
	if nodeConfigurations, apiErr = repo.TryQueryAllNodeConfigurations(nodeSet); apiErr != nil {
		t.Fatalf("unexpected error: %v", apiErr)
	}

	if len(nodeConfigurations) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodeConfigurations))
	}

	if nodeConfigurations[0].NodeID != node2.NodeID {
		t.Fatalf("expected nodeID %s, got %s", node2.NodeID, nodeConfigurations[0].NodeID)
	}

	// Query all nodes in non-existent set
	// need to change query filter
	store.QueryFilterFunc = func(rowKey string, properties map[string]interface{}) bool {
		val := properties[NodeSetKeyProperty]
		return val == "this-set-does-not-exist"
	}

	if nodeConfigurations, apiErr = repo.TryQueryAllNodeConfigurations(nodeSet); apiErr != nil {
		t.Fatalf("unexpected error: %v", apiErr)
	}

	if len(nodeConfigurations) != 0 {
		t.Fatalf("expected 0 nodes, got %d", len(nodeConfigurations))
	}
}
