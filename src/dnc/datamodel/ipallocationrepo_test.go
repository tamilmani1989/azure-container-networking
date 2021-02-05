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

func TestQueryIpAllocationsByToBeDeletedOwner(t *testing.T) {
	var store testpersistencestore.TestPersistenceStore
	store.Init(TableToColumnMap, loggermock.NewDummyLogger())
	partition := "Network_vnet1_Subnet_subnet1"
	toBeDeletedOwnerId := "Network_vnet1_NC_nc1"
	repo := AllocationRepo{Datastore: &store}

	rows, err := repo.QueryIPAllocationsByToBeDeletedOwner("vnet1", "subnet1", toBeDeletedOwnerId)
	if err != nil {
		t.Fatalf("did not expect error: %v", err)
	}

	if len(rows) != 0 {
		t.Fatalf("Expected empty ipAllocations with ToBeDeletedOwnerId %s", toBeDeletedOwnerId)
	}

	t.Logf("Create IpAllocations with no ToBeDeletedOwnerId")
	ipallocationsWithoutTBD := map[string]map[string]interface{}{
		"10.0.0.1_32": map[string]interface{}{IPAllocationProperty: "",
			AllocationIDProperty: "alloc1"},
		"10.0.0.2_32": map[string]interface{}{IPAllocationProperty: "res2",
			AllocationIDProperty: "alloc2"},
	}

	for rowkey, properties := range ipallocationsWithoutTBD {
		if _, err := store.InsertInPartitionProperties(IPAllocationsTableName, partition, rowkey, properties); err != nil {
			t.Fatal(err)
		}
	}

	t.Logf("QueryIPAllocationsByToBeDeletedOwner should return empty rows")
	rows, err = repo.QueryIPAllocationsByToBeDeletedOwner("vnet1", "subnet1", toBeDeletedOwnerId)
	if err != nil {
		t.Fatalf("did not expect error: %v", err)
	}

	if len(rows) != 0 {
		t.Fatalf("Expected empty ipAllocations with ToBeDeletedOwnerId %s", toBeDeletedOwnerId)
	}

	t.Logf("Create ipAllocations with expected ToBeDeletedOwnerId %s", toBeDeletedOwnerId)
	ipallocationsWithTBD := map[string]map[string]interface{}{
		"10.0.0.3_32": map[string]interface{}{IPAllocationProperty: "",
			AllocationIDProperty:     "alloc3",
			ToBeDeletedOwnerProperty: toBeDeletedOwnerId},
		"10.0.0.4_32": map[string]interface{}{IPAllocationProperty: "res2",
			AllocationIDProperty:     "alloc4",
			ToBeDeletedOwnerProperty: toBeDeletedOwnerId},
	}

	for rowkey, properties := range ipallocationsWithTBD {
		if _, err := store.InsertInPartitionProperties(IPAllocationsTableName, partition, rowkey, properties); err != nil {
			t.Fatal(err)
		}
	}

	t.Logf("QueryIPAllocationsByToBeDeletedOwner should return valid rows")
	rows, err = repo.QueryIPAllocationsByToBeDeletedOwner("vnet1", "subnet1", toBeDeletedOwnerId)
	if err != nil {
		t.Fatalf("did not expect error: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("Expected valid ipAllocations with ToBeDeletedOwnerId %s, %+v", toBeDeletedOwnerId, rows)
	}

	for _, row := range rows {
		if _, found := ipallocationsWithTBD[row.RowKey]; !found {
			t.Fatalf("Unexpected RowKey in the result %s, %+v", row.RowKey, rows)
		}
	}

	differentToBeDeletedOwnerId := "Network_vnet1_NC_nc2"
	t.Logf("Create ipAllocations with some other ToBeDeletedOwnerId %s", differentToBeDeletedOwnerId)
	ipallocationsWithOtherTBD := map[string]map[string]interface{}{
		"10.0.0.5_32": map[string]interface{}{IPAllocationProperty: "",
			AllocationIDProperty:     "alloc5",
			ToBeDeletedOwnerProperty: differentToBeDeletedOwnerId},
		"10.0.0.6_32": map[string]interface{}{IPAllocationProperty: "",
			AllocationIDProperty:     "alloc6",
			ToBeDeletedOwnerProperty: differentToBeDeletedOwnerId},
	}

	for rowkey, properties := range ipallocationsWithOtherTBD {
		if _, err := store.InsertInPartitionProperties(IPAllocationsTableName, partition, rowkey, properties); err != nil {
			t.Fatal(err)
		}
	}

	t.Logf("QueryIPAllocationsByToBeDeletedOwner should return valid rows, only with expected ToBeDeletedOwner")
	rows, err = repo.QueryIPAllocationsByToBeDeletedOwner("vnet1", "subnet1", toBeDeletedOwnerId)
	if err != nil {
		t.Fatalf("did not expect error: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("Expected valid ipAllocations with ToBeDeletedOwnerId %s, %+v", toBeDeletedOwnerId, rows)
	}

	for _, row := range rows {
		if _, found := ipallocationsWithTBD[row.RowKey]; !found {
			t.Fatalf("Unexpected RowKey in the result %s, %+v", row.RowKey, rows)
		}
	}
}
