package networkcontainer

import (
	"dnc/api"
	"dnc/datamodel"
	"dnc/datastore/testpersistencestore"
	"dnc/logger/loggermock"
	"dnc/networkcontainer/fakes"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/google/uuid"
)

func Test_Allocate(t *testing.T) {
	allocsRepo := fakes.AllocationRepo{
		Data: []datamodel.IPAllocationEntity{
			{RowKey: "172.16.0.10_32", IPAllocation: "Network_network1_Reservation_res1", AllocationID: ""},
			{RowKey: "172.16.0.13_32", IPAllocation: "Network_network1_Reservation_res2", AllocationID: "uuid1"},
			{RowKey: "172.16.0.20_32", IPAllocation: "Network_network1_Reservation_res2", AllocationID: "uuid2"},
			{RowKey: "172.16.0.21_32", IPAllocation: "Network_network1_Reservation_res2", AllocationID: "uuid3"},
			{RowKey: "172.16.0.22_32", IPAllocation: "Network_network1_Reservation_res2", AllocationID: "uuid4"},
			{RowKey: "172.16.0.23_32", IPAllocation: "Network_network1_Reservation_res2", AllocationID: "uuid5"},
		},
		MaxBatchSize: 100,
	}

	subnetRepo := fakes.SubnetRepo{
		Data: map[string]datamodel.SubnetConfiguration{
			datamodel.GetSubnetKey("network1", "subnet1"): {
				NetworkID:  "network1",
				SubnetName: "subnet1",
				IPSubnet:   "172.16.0.0/24",
			},
		},
	}

	allocator := IPAllocator{
		Allocations:  &allocsRepo,
		Subnets:      &subnetRepo,
		Reservations: fakes.NewReservationRepo(),
		MaxBatchSize: 100,
	}

	req := datamodel.ReservationConfiguration{
		ReservationID: "reservation1",
		NetworkID:     "network1",
		AllocationRequest: api.AllocationRequest{
			SubnetName: "subnet1",
		},
		SecondaryAllocations: map[string]string{
			"alloc1": "",
			"alloc2": "",
			"alloc3": "",
			"alloc4": "",
			"alloc5": "",
			"alloc6": "",
			"alloc7": "",
			"alloc8": "",
			"alloc9": "",
		},
	}
	_, baseSubnet, _ := net.ParseCIDR("172.16.0.0/24")
	complete, err := allocator.ProcessIPReservationAllocation(&req)
	if err != nil {
		t.Fatalf("did not expect error: %+v", err)
	}
	if !complete {
		t.Fatal("expected allocation to be done")
	}
	if req.AllocatedSubnet != "172.16.0.4/32" {
		t.Errorf("expected primary subnet to be 172.16.0.4/32, got %s", req.AllocatedSubnet)
	}
	for _, ip := range req.SecondaryAllocations {
		secIP, _, err := net.ParseCIDR(ip)
		if err != nil {
			t.Errorf("did not expect error: %+v", err)
		}
		if !baseSubnet.Contains(secIP) {
			t.Errorf("secondary ip %s is not within subnet %s", secIP.String(), baseSubnet.String())
		}
	}
}

func TestIncompleteAllocation(t *testing.T) {
	// these are ids that were saved in the allocations table, but not associated to the reservation
	reservationID := "res1"
	allocationID1 := "uuid1"

	// this id has not been allocated an ip yet
	allocationID2 := "uuid3"

	// this is the current state of the allocations table
	allocsRepo := fakes.AllocationRepo{
		Data: []datamodel.IPAllocationEntity{
			{RowKey: "172.16.0.13_32", IPAllocation: "Network_network1_Reservation_res1", AllocationID: reservationID},
			{RowKey: "172.16.0.20_32", IPAllocation: "Network_network1_Reservation_res1", AllocationID: allocationID1},
		},
		MaxBatchSize: 100,
	}

	subnetRepo := fakes.SubnetRepo{
		Data: map[string]datamodel.SubnetConfiguration{
			datamodel.GetSubnetKey("network1", "subnet1"): {
				NetworkID:  "network1",
				SubnetName: "subnet1",
				IPSubnet:   "172.16.0.0/24",
			},
		},
	}

	allocator := IPAllocator{
		Allocations:  &allocsRepo,
		Subnets:      &subnetRepo,
		Reservations: fakes.NewReservationRepo(),
		MaxBatchSize: 100,
	}

	req := datamodel.ReservationConfiguration{
		ReservationID:     reservationID,
		NetworkID:         "network1",
		AllocationRequest: api.AllocationRequest{SubnetName: "subnet1"},
		// none of the allocations were saved to the reservation, but they do exist in the allocations table
		AllocatedSubnet: "",
		SecondaryAllocations: map[string]string{
			allocationID1: "",
			allocationID2: "",
		},
	}
	_, baseSubnet, _ := net.ParseCIDR("172.16.0.0/24")
	complete, err := allocator.ProcessIPReservationAllocation(&req)
	if err != nil {
		t.Fatalf("did not expect error: %+v", err)
	}
	if !complete {
		t.Fatal("expected allocation to be done")
	}
	if req.AllocatedSubnet != "172.16.0.13/32" {
		t.Errorf("expected primary subnet to be 172.16.0.13/32, got %s", req.AllocatedSubnet)
	}

	allocatedIP1, ok := req.SecondaryAllocations[allocationID1]
	if !ok {
		t.Error("expected previous allocation to be associated to reservation")
	}
	if allocatedIP1 != "172.16.0.20/32" {
		t.Errorf("expected associated allocation %s to be 172.16.0.20/32, got %s", allocationID1, allocatedIP1)
	}
	allocatedIP2, ok := req.SecondaryAllocations[allocationID2]
	if !ok {
		t.Error("expected previous allocation to be associated to reservation")
	}
	ip, _, parseErr := net.ParseCIDR(allocatedIP2)
	if parseErr != nil {
		t.Fatalf("did not expect error: %+v", err)
	}
	if !baseSubnet.Contains(ip) {
		t.Errorf("allocated ip %s is not within subnet %s", ip.String(), baseSubnet.String())
	}
}

func TestDeallocate(t *testing.T) {
	initialAllocCount := 240
	allocationRows := make([]datamodel.IPAllocationEntity, initialAllocCount)
	allocationIDs := make([]string, initialAllocCount)
	reservation := datamodel.ReservationConfiguration{
		SecondaryAllocations: make(map[string]string),
	}

	// generate allocation rows, secondary allocations in reservation, and capture ids that can be referenced in request
	_, baseSubnet, _ := net.ParseCIDR("10.0.0.0/16")
	ips, err := findAllocatableIPs(baseSubnet, nil, initialAllocCount)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(allocationRows); i++ {
		allocID := uuid.New().String()
		allocationRows[i] = datamodel.IPAllocationEntity{
			RowKey:       ips[i].String() + "_32",
			IPAllocation: "", // todo: verify
			AllocationID: allocID,
		}
		allocationIDs[i] = allocID
		reservation.SecondaryAllocations[allocID] = ips[i].String() + "/32"
	}
	allocRepo := fakes.AllocationRepo{Data: allocationRows, MaxBatchSize: 100}
	allocator := IPAllocator{Allocations: &allocRepo, Logger: loggermock.NewDummyLogger(), MaxBatchSize: 100}

	// randomly select a number of ids to be deleted
	rand.Shuffle(len(allocationIDs), func(i, j int) {
		allocationIDs[i], allocationIDs[j] = allocationIDs[j], allocationIDs[i]
	})

	deleteCount := 105
	if err := allocator.DeallocateIPs(reservation, allocationIDs[:deleteCount]); err != nil {
		t.Fatalf("did not expect error, got: %+v", err)
	}

	remainingIPIDs := extractAllocationIDs(allocRepo)
	expectedCount := initialAllocCount - deleteCount

	if len(remainingIPIDs) != expectedCount {
		t.Fatalf("expected remaining IPs to be %d, got %d", expectedCount, len(remainingIPIDs))
	}
	for _, id := range allocationIDs[:deleteCount] {
		if _, found := remainingIPIDs[id]; found {
			t.Fatal("found an id that should have been removed")
		}
	}
}

func Test_RemoveSecondaryIpWithEmptyAllocations(t *testing.T) {
	var allocsRepo fakes.AllocationRepo

	subnetRepo := fakes.SubnetRepo{
		Data: map[string]datamodel.SubnetConfiguration{
			datamodel.GetSubnetKey("network1", "subnet1"): {
				NetworkID:  "network1",
				SubnetName: "subnet1",
				IPSubnet:   "172.16.0.0/24",
			},
		},
	}

	allocator := IPAllocator{
		Allocations:  &allocsRepo,
		Subnets:      &subnetRepo,
		Reservations: fakes.NewReservationRepo(),
		Logger:       loggermock.NewDummyLogger(),
		MaxBatchSize: 100,
	}

	req := datamodel.ReservationConfiguration{
		ReservationID: "reservation1",
		NetworkID:     "network1",
		AllocationRequest: api.AllocationRequest{
			SubnetName: "subnet1",
		},
		SecondaryAllocations: map[string]string{
			"alloc1": "",
			"alloc2": "",
			"alloc3": "",
			"alloc4": "",
			"alloc5": "",
			"alloc6": "",
			"alloc7": "",
			"alloc8": "",
			"alloc9": "",
		},
	}

	err := allocator.DeleteReservationAllocations(req.NetworkID, req.AllocationRequest.SubnetName, datamodel.GetReservationKey(req.NetworkID, req.ReservationID))
	if err != nil {
		t.Fatalf("did not expect error: %+v", err)
	}
}

func Test_RemoveAllAllocations(t *testing.T) {
	allocsRepo := fakes.AllocationRepo{
		Data: []datamodel.IPAllocationEntity{
			{RowKey: "172.16.0.13_32", IPAllocation: "Network_network1_Reservation_res1", AllocationID: "alloc1"},
			{RowKey: "172.16.0.14_32", IPAllocation: "Network_network1_Reservation_res1", AllocationID: "alloc2"},
			{RowKey: "172.16.0.15_32", IPAllocation: "Network_network1_Reservation_res1", AllocationID: "alloc3"},
			{RowKey: "172.16.0.16_32", IPAllocation: "Network_network1_Reservation_res1", AllocationID: "alloc4"},
		},
		MaxBatchSize: 100,
	}

	subnetRepo := fakes.SubnetRepo{
		Data: map[string]datamodel.SubnetConfiguration{
			datamodel.GetSubnetKey("network1", "subnet1"): {
				NetworkID:  "network1",
				SubnetName: "subnet1",
				IPSubnet:   "172.16.0.0/24",
			},
		},
	}

	allocator := IPAllocator{
		Allocations:  &allocsRepo,
		Subnets:      &subnetRepo,
		Reservations: fakes.NewReservationRepo(),
		Logger:       loggermock.NewDummyLogger(),
		MaxBatchSize: 100,
	}

	res := datamodel.ReservationConfiguration{
		ReservationID: "res1",
		NetworkID:     "network1",
		AllocationRequest: api.AllocationRequest{
			SubnetName: "subnet1",
		},
		SecondaryAllocations: map[string]string{
			"alloc1": "",
			"alloc2": "",
			"alloc3": "",
			"alloc4": "",
		},
	}

	if err := allocator.DeleteReservationAllocations(res.NetworkID, res.AllocationRequest.SubnetName, datamodel.GetReservationKey(res.NetworkID, res.ReservationID)); err != nil {
		t.Fatalf("did not expect error: %+v", err)
	}

	if len(allocsRepo.Data) != 0 {
		t.Fatalf("expected all allocations to be removed, but had %d left", len(allocsRepo.Data))
	}
}

func extractAllocationIDs(a fakes.AllocationRepo) map[string]struct{} {
	allocIDs := make(map[string]struct{})
	for _, row := range a.Data {
		if row.AllocationID != "" {
			allocIDs[row.AllocationID] = struct{}{}
		}
	}
	return allocIDs
}

func TestGenerateAllRows(t *testing.T) {
	_, subnet, _ := net.ParseCIDR("10.0.0.0/16")
	rows, _ := GenerateIPAllocationEntitiesForSubnet(subnet)
	if len(rows) != 65536 {
		t.Fatalf("expected 65536 rows, got %d", len(rows))
	}

	count := 0
	for _, row := range rows {
		if row.IPAllocation == "Azure Reserved Address" {
			count++
		}
	}

	if count != 5 {
		t.Fatalf("expected 5 reserved, got %d", count)
	}
}

const slash16HostCount = 256 * 256

func TestPrepopulateRows(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		var ds testpersistencestore.TestPersistenceStore
		ds.Init(datamodel.TableToColumnMap, loggermock.NewNoOpLogger())
		allocRepo := datamodel.AllocationRepo{Datastore: &ds}

		cidr := "10.0.0.0/16"
		vnetID := "vnet1"
		subnetName := "subnet1"

		a := IPAllocator{
			Allocations:  &allocRepo,
			Logger:       loggermock.NewDummyLogger(),
			MaxBatchSize: 1000,
		}

		err := a.PrepopulateSubnet(vnetID, subnetName, cidr)
		require.Nil(t, err)

		allocs, err := allocRepo.TryQueryAllIPAllocations(vnetID, subnetName)
		require.Nil(t, err)

		assert.Len(t, allocs, slash16HostCount)
	})

	t.Run("should be retryable", func(t *testing.T) {
		var ds testpersistencestore.TestPersistenceStore
		ds.Init(datamodel.TableToColumnMap, loggermock.NewNoOpLogger())
		allocRepo := datamodel.AllocationRepo{Datastore: &ds}
		maxBatchSize := 1000
		cidr := "10.0.0.0/16"
		vnetID := "vnet1"
		subnetName := "subnet1"

		a := IPAllocator{
			Allocations:  &allocRepo,
			Logger:       loggermock.NewNoOpLogger(),
			MaxBatchSize: maxBatchSize,
		}

		_, subnet, _ := net.ParseCIDR(cidr)
		allAllocs, _ := GenerateIPAllocationEntitiesForSubnet(subnet)
		require.Len(t, allAllocs, slash16HostCount)

		// simulate dispersed batch ranges from previous run that succeeded
		partialAllocs := [][]datamodel.IPAllocationEntity{
			allAllocs[maxBatchSize*2 : maxBatchSize*3],
			allAllocs[maxBatchSize*15 : maxBatchSize*18],
			allAllocs[maxBatchSize*32 : maxBatchSize*50],
		}

		for _, p := range partialAllocs {
			_ = allocRepo.InsertIPAllocationEntities(vnetID, subnetName, p)
		}

		err := a.PrepopulateSubnet(vnetID, subnetName, cidr)
		require.Nil(t, err)

		allocs, err := allocRepo.TryQueryAllIPAllocations(vnetID, subnetName)
		require.Nil(t, err)

		assert.Len(t, allocs, slash16HostCount)
	})

	t.Run("can be called concurrently", func(t *testing.T) {
		var ds testpersistencestore.TestPersistenceStore
		ds.Init(datamodel.TableToColumnMap, loggermock.NewNoOpLogger())
		allocRepo := datamodel.AllocationRepo{Datastore: &ds}
		maxBatchSize := 1000
		cidr := "10.0.0.0/16"
		vnetID := "vnet1"
		subnetName := "subnet1"

		a := IPAllocator{
			Allocations:  &allocRepo,
			Logger:       loggermock.NewNoOpLogger(),
			MaxBatchSize: maxBatchSize,
		}

		// simulate 10 DNCs working on the same subnet
		var wg sync.WaitGroup
		wg.Add(10)
		for i := 0; i < 10; i++ {
			go func(workerID int) {
				defer wg.Done()
				err := a.PrepopulateSubnet(vnetID, subnetName, cidr)
				assert.Nilf(t, err, "worker %d: did not expect error", workerID)
			}(i)
		}
		wg.Wait()

		allocs, err := allocRepo.TryQueryAllIPAllocations(vnetID, subnetName)
		require.Nil(t, err)

		assert.Len(t, allocs, slash16HostCount)
	})

	t.Run("handles throttling", func(t *testing.T) {
		// todo: tested manually, just no easy way to simulate with test data store at the moment
		t.SkipNow()
	})
}

func TestSetToBeDeletingIps(t *testing.T) {
	vnetID := "vnet1"
	subnetName := "subnet1"
	ncId := "nc1"
	resId := "res1"
	ncKey := datamodel.GetNCKey(vnetID, ncId)
	var resKey = datamodel.GetReservationKey(vnetID, resId)
	allocsRepo := fakes.AllocationRepo{
		Data: []datamodel.IPAllocationEntity{
			{RowKey: "172.16.0.13_32", IPAllocation: resKey, AllocationID: "alloc1"},
			{RowKey: "172.16.0.14_32", IPAllocation: resKey, AllocationID: "alloc2"},
			{RowKey: "172.16.0.15_32", IPAllocation: resKey, AllocationID: "alloc3"},
		},
		MaxBatchSize: 100,
	}

	var allocIDsToDelete = make(map[string]string)
	allocIDsToDelete["alloc1"] = "172.16.0.13_32"
	allocIDsToDelete["alloc2"] = ""

	a := IPAllocator{
		Allocations:  &allocsRepo,
		Logger:       loggermock.NewDummyLogger(),
		MaxBatchSize: 100,
	}

	err := a.SetToBeDeleteOwnerToDeletingIps(vnetID, subnetName, resKey, ncKey, allocIDsToDelete)
	require.Nil(t, err)

	updatedAllocs, err := a.Allocations.TryQueryAllIPAllocations(vnetID, subnetName)
	require.Nil(t, err)

	for _, entity := range updatedAllocs {
		if _, found := allocIDsToDelete[entity.AllocationID]; found {
			if entity.ToBeDeletedOwner != ncKey {
				t.Fatalf("ToBeDeletedOwner is not set in IpAllocation, entity %+v, nckey %s", entity, ncKey)
			}
		}
	}
}

func TestMaxBatchLimitInSetToBeDeletingIps(t *testing.T) {
	vnetID := "vnet1"
	subnetName := "subnet1"
	ncId := "nc1"
	resId := "res1"
	ncKey := datamodel.GetNCKey(vnetID, ncId)
	var resKey = datamodel.GetReservationKey(vnetID, resId)

	var entities []datamodel.IPAllocationEntity
	allocIDsToDelete := make(map[string]string)
	for i := 0; i < 200; i++ {
		allocId := fmt.Sprintf("alloc%d", i)
		ipaddress := fmt.Sprintf("10.0.0.%d_32", i)
		entity := datamodel.IPAllocationEntity{
			RowKey:       ipaddress,
			IPAllocation: resKey,
			AllocationID: allocId,
		}

		entities = append(entities, entity)
		allocIDsToDelete[allocId] = ipaddress
	}

	allocsRepo := fakes.AllocationRepo{
		Data:         entities,
		MaxBatchSize: 100,
	}

	a := IPAllocator{
		Allocations:  &allocsRepo,
		Logger:       loggermock.NewDummyLogger(),
		MaxBatchSize: 100,
	}

	err := a.SetToBeDeleteOwnerToDeletingIps(vnetID, subnetName, resKey, ncKey, allocIDsToDelete)
	require.Nil(t, err)

	updatedAllocs, err := a.Allocations.TryQueryAllIPAllocations(vnetID, subnetName)
	require.Nil(t, err)

	for _, entity := range updatedAllocs {
		if entity.ToBeDeletedOwner != ncKey {
			t.Fatalf("ToBeDeletedOwner is not set in IpAllocation, entity %+v, nckey %s", entity, ncKey)
		}
	}
}
