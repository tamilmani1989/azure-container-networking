package networkcontainer

import (
	"dnc/api"
	"dnc/configuration"
	"dnc/datamodel"
	"dnc/networkcontainer/fakes"
	"math/rand"
	"net"
	"testing"

	"github.com/google/uuid"
)

func Test_Allocate(t *testing.T) {
	allocsRepo := fakes.AllocationRepo{
		Data: []fakes.RepoKV{
			{"172.16.0.10_32", map[string]interface{}{ipAllocProperty: "Network_network1_Reservation_res1"}},
			{"172.16.0.13_32", map[string]interface{}{ipAllocProperty: "Network_network1_Reservation_res2", allocIDProperty: "uuid1"}},
			{"172.16.0.20_32", map[string]interface{}{ipAllocProperty: "Network_network1_Reservation_res2", allocIDProperty: "uuid2"}},
			{"172.16.0.21_32", map[string]interface{}{ipAllocProperty: "Network_network1_Reservation_res2", allocIDProperty: "uuid3"}},
			{"172.16.0.22_32", map[string]interface{}{ipAllocProperty: "Network_network1_Reservation_res2", allocIDProperty: "uuid4"}},
			{"172.16.0.23_32", map[string]interface{}{ipAllocProperty: "Network_network1_Reservation_res2", allocIDProperty: "uuid5"}},
		},
		MaxPageSize: 3,
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
		Allocations:      &allocsRepo,
		Subnets:          &subnetRepo,
		Reservations:     fakes.NewReservationRepo(),
		IsFeatureEnabled: enableSubnetDelegation,
		MaxBatchSize:     100,
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
		Data: []fakes.RepoKV{
			{"172.16.0.13_32", map[string]interface{}{ipAllocProperty: "Network_network1_Reservation_res1", allocIDProperty: reservationID}},
			{"172.16.0.20_32", map[string]interface{}{ipAllocProperty: "Network_network1_Reservation_res1", allocIDProperty: allocationID1}},
		},
		MaxPageSize: 5,
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
		Allocations:      &allocsRepo,
		Subnets:          &subnetRepo,
		Reservations:     fakes.NewReservationRepo(),
		IsFeatureEnabled: enableSubnetDelegation,
		MaxBatchSize:     100,
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
	allocationRows := make([]fakes.RepoKV, initialAllocCount)
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
		allocationRows[i] = fakes.RepoKV{Key: ips[i].String() + "_32", Value: map[string]interface{}{allocIDProperty: allocID}}
		allocationIDs[i] = allocID
		reservation.SecondaryAllocations[allocID] = ips[i].String() + "/32"
	}
	allocRepo := fakes.AllocationRepo{Data: allocationRows, MaxPageSize: 100}
	allocator := IPAllocator{Allocations: &allocRepo, Logger: new(dummyLogger), MaxBatchSize: 100}

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
	allocsRepo := fakes.AllocationRepo{
		Data:        []fakes.RepoKV{},
		MaxPageSize: 3,
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
		Allocations:      &allocsRepo,
		Subnets:          &subnetRepo,
		Reservations:     fakes.NewReservationRepo(),
		IsFeatureEnabled: enableSubnetDelegation,
		Logger:           &dummyLogger{},
		MaxBatchSize:     100,
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

	err := allocator.RemoveAllAllocations(req)
	if err != nil {
		t.Fatalf("did not expect error: %+v", err)
	}
}

func Test_RemoveAllAllocations(t *testing.T) {
	allocsRepo := fakes.AllocationRepo{
		Data: []fakes.RepoKV{
			{"172.16.0.13_32", map[string]interface{}{ipAllocProperty: "Network_network1_Reservation_res1", allocIDProperty: "alloc1"}},
			{"172.16.0.14_32", map[string]interface{}{ipAllocProperty: "Network_network1_Reservation_res1", allocIDProperty: "alloc2"}},
			{"172.16.0.15_32", map[string]interface{}{ipAllocProperty: "Network_network1_Reservation_res1", allocIDProperty: "alloc3"}},
			{"172.16.0.16_32", map[string]interface{}{ipAllocProperty: "Network_network1_Reservation_res1", allocIDProperty: "alloc4"}},
		},
		MaxPageSize: 5,
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
		Allocations:      &allocsRepo,
		Subnets:          &subnetRepo,
		Reservations:     fakes.NewReservationRepo(),
		IsFeatureEnabled: enableSubnetDelegation,
		MaxBatchSize:     100,
	}

	res := datamodel.ReservationConfiguration{
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
		},
	}

	if err := allocator.RemoveAllAllocations(res); err != nil {
		t.Fatalf("did not expect error: %+v", err)
	}

	if len(allocsRepo.Data) != 0 {
		t.Fatalf("expected all allocations to be removed, but had %d left", len(allocsRepo.Data))
	}
}

func extractAllocationIDs(a fakes.AllocationRepo) map[string]struct{} {
	allocIDs := make(map[string]struct{})
	for _, kv := range a.Data {
		allocID, ok := kv.Value.(map[string]interface{})[allocIDProperty].(string)
		if !ok {
			continue
		}
		allocIDs[allocID] = struct{}{}
	}
	return allocIDs
}

// mock function to signal that subnet delegation is enabled
func enableSubnetDelegation(i int) bool { return i == configuration.SharedSubnetDelegation }
