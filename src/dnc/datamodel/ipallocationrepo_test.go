package datamodel

import (
	"dnc/datastore/testpersistencestore"
	"dnc/logger/loggermock"
	"testing"
)

// todo:
//  - cover more repo methods
//  - use testify packages for concise assertions
func TestTryQueryAllIPAllocations(t *testing.T) {
	var store testpersistencestore.TestPersistenceStore
	store.Init(TableToColumnMap, loggermock.NewDummyLogger())

	partition := "Network_08107103-ee55-04bd-8222-c3c4ffd8adcb_Subnet_subnet1"
	rowKey := "10.0.0.1_32"
	reservationID := "reservation1"
	allocationID := "allocation1"
	properties := map[string]interface{}{
		IPAllocationProperty: reservationID,
		AllocationIDProperty: allocationID,
	}

	// todo:
	//  - should add constructor for test data store
	//  - test data store does not handle partitions. note the inputs below and the encoded partition key above. test will still pass
	if _, err := store.InsertInPartitionProperties(IPAllocationsTableName, partition, rowKey, properties); err != nil {
		t.Fatal(err)
	}

	repo := AllocationRepo{Datastore: &store}

	rows, err := repo.TryQueryAllIPAllocations("vnet1", "subnet1")
	if err != nil {
		t.Fatalf("did not expect error: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	row := rows[0]
	if row.RowKey != rowKey {
		t.Errorf("expected row key %s, got %s", rowKey, row.RowKey)
	}
	if row.ETag == "" {
		t.Error("expected etag to not be empty")
	}
	if row.IPAllocation != reservationID {
		t.Errorf("expected ip allocation property to be %s, got %s", reservationID, row.IPAllocation)
	}
	if row.AllocationID != allocationID {
		t.Errorf("expected allocation id property to be %s, got %s", allocationID, row.AllocationID)
	}
	if row.ToBeDeletedOwner != "" {
		t.Error("expected tbd owner to be empty")
	}
}
