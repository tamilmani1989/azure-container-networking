package networkcontainer

import (
	"dnc/datamodel"
	"dnc/networkcontainer/fakes"
	"net"
	"testing"

	"github.com/google/uuid"
)

func TestSubnetHandlerProcess_CreationFromScratch(t *testing.T) {
	allocsRepo := fakes.AllocationRepo{}
	resRepo := fakes.NewReservationRepo()

	_, baseSubnet, _ := net.ParseCIDR("10.1.0.0/16")
	vnetID := "vnet1"
	subnet := "subnet1"

	sh := subnetHandler{
		reservations: resRepo,
		subnets: &fakes.SubnetRepo{
			Data: map[string]datamodel.SubnetConfiguration{
				datamodel.GetSubnetKey(vnetID, subnet): {},
			},
		},
		allocator: &IPAllocator{
			Allocations:  &allocsRepo,
			Logger:       new(dummyLogger),
			MaxBatchSize: 100,
		},
		logger: new(dummyLogger),
		subnetInfo: subnetInfo{
			vnetID:     vnetID,
			subnetName: subnet,
			baseSubnet: baseSubnet,
		},
		maxBatchSize: 100,
	}

	reservationCount := 100
	secondaryIPCount := 32

	set, respChans := generateBatch(reservationCount, secondaryIPCount, vnetID)

	sh.process(set)

	reservationIPs := make(map[string]struct{})

	// inspect stored reservations
	for _, rsv := range resRepo.Data {
		assertCompleteReservation(t, *rsv.Configuration, secondaryIPCount)

		reservationIPs[rsv.Configuration.AllocatedSubnet] = struct{}{}
		for _, ip := range rsv.Configuration.SecondaryAllocations {
			reservationIPs[ip] = struct{}{}
		}
	}

	expectedAllocationCount := reservationCount * (1 + secondaryIPCount) // accounting for primary ip

	// all the ips stored in reservations, whether primary or secondary, should be unique
	if len(reservationIPs) != expectedAllocationCount {
		t.Fatalf("ips in same subnet reservations are not unique, expected %d, got %d", expectedAllocationCount, len(reservationIPs))
	}

	if len(allocsRepo.Data) != expectedAllocationCount {
		t.Fatalf("expected %d allocations, got %d", expectedAllocationCount, len(allocsRepo.Data))
	}

	assertCompleteResponses(t, respChans, secondaryIPCount)
}

func TestSubnetHandlerProcess_RecoveringFromErrors(t *testing.T) {
	/*
		This test begins with partial state, possibly caused by transient errors in previous processing of
		some elements in the batch. The purpose is to show that batch processing is idempotent.
		An example is, allocations for a subset of the batch had been previously processed, but the reservations
		could not be updated due to a network error. Allocations in the db have references to reservations, but
		reservations have no knowledge of their corresponding allocations.
	*/

	_, baseSubnet, _ := net.ParseCIDR("10.1.0.0/20")
	vnetID := "vnet1"
	subnet := "subnet1"

	reservationCount := 10
	secondaryIPCount := 5

	set, respChans := generateBatch(reservationCount, secondaryIPCount, vnetID)

	partialCount := 2
	partialIDs := make([]string, partialCount)
	allocIDs := make(map[string][]string)

	i := 0
	for _, v := range set {
		resID := v.reservation.ReservationID
		partialIDs[i] = resID
		for allocID := range v.reservation.SecondaryAllocations {
			allocIDs[resID] = append(allocIDs[resID], allocID)
		}
		i++
		if i == partialCount {
			break
		}
	}

	allocsRepo := fakes.AllocationRepo{
		Data: []datamodel.AllocationRow{
			// one reservation with complete allocations (primary and secondaries)
			{RowKey: "10.1.0.10_32", IPAllocation: "Network_" + vnetID + "_Reservation_" + partialIDs[0], AllocationID: partialIDs[0]},
			{RowKey: "10.1.0.11_32", IPAllocation: "Network_" + vnetID + "_Reservation_" + partialIDs[0], AllocationID: allocIDs[partialIDs[0]][0]},
			{RowKey: "10.1.0.12_32", IPAllocation: "Network_" + vnetID + "_Reservation_" + partialIDs[0], AllocationID: allocIDs[partialIDs[0]][1]},
			{RowKey: "10.1.0.13_32", IPAllocation: "Network_" + vnetID + "_Reservation_" + partialIDs[0], AllocationID: allocIDs[partialIDs[0]][2]},
			{RowKey: "10.1.0.14_32", IPAllocation: "Network_" + vnetID + "_Reservation_" + partialIDs[0], AllocationID: allocIDs[partialIDs[0]][3]},
			{RowKey: "10.1.0.15_32", IPAllocation: "Network_" + vnetID + "_Reservation_" + partialIDs[0], AllocationID: allocIDs[partialIDs[0]][4]},
			// one reservation with partial allocations (primary and only 2 secondaries)
			{RowKey: "10.1.0.20_32", IPAllocation: "Network_" + vnetID + "_Reservation_" + partialIDs[1], AllocationID: partialIDs[1]},
			{RowKey: "10.1.0.21_32", IPAllocation: "Network_" + vnetID + "_Reservation_" + partialIDs[1], AllocationID: allocIDs[partialIDs[1]][0]},
			{RowKey: "10.1.0.22_32", IPAllocation: "Network_" + vnetID + "_Reservation_" + partialIDs[1], AllocationID: allocIDs[partialIDs[1]][1]},
		},
	}
	resRepo := fakes.NewReservationRepo()

	sh := subnetHandler{
		reservations: resRepo,
		subnets: &fakes.SubnetRepo{
			Data: map[string]datamodel.SubnetConfiguration{
				datamodel.GetSubnetKey(vnetID, subnet): {},
			},
		},
		allocator: &IPAllocator{
			Allocations:  &allocsRepo,
			Logger:       new(dummyLogger),
			MaxBatchSize: 100,
		},
		logger: new(dummyLogger),
		subnetInfo: subnetInfo{
			vnetID:     vnetID,
			subnetName: subnet,
			baseSubnet: baseSubnet,
		},
		maxBatchSize: 100,
	}

	sh.process(set)

	reservationIPs := make(map[string]struct{})

	// inspect stored reservations
	for _, rsv := range resRepo.Data {
		assertCompleteReservation(t, *rsv.Configuration, secondaryIPCount)

		reservationIPs[rsv.Configuration.AllocatedSubnet] = struct{}{}
		for _, ip := range rsv.Configuration.SecondaryAllocations {
			reservationIPs[ip] = struct{}{}
		}
	}

	expectedAllocationCount := reservationCount * (1 + secondaryIPCount) // accounting for primary ip

	// all the ips stored in reservations, whether primary or secondary, should be unique
	if len(reservationIPs) != expectedAllocationCount {
		t.Fatalf("ips in same subnet reservations are not unique, expected %d, got %d", expectedAllocationCount, len(reservationIPs))
	}

	if len(allocsRepo.Data) != expectedAllocationCount {
		t.Fatalf("expected %d allocations, got %d", expectedAllocationCount, len(allocsRepo.Data))
	}

	assertCompleteResponses(t, respChans, secondaryIPCount)
}

func TestDeallocateBatch(t *testing.T) {
	_, baseSubnet, _ := net.ParseCIDR("10.1.0.0/16")
	vnetID := "vnet1"
	subnet := "subnet1"

	allocsRepo := fakes.AllocationRepo{
		Data: []datamodel.AllocationRow{
			{RowKey: "10.1.0.10_32", IPAllocation: "Network_vnet1_Reservation_res1", AllocationID: "res1"},
			{RowKey: "10.1.0.11_32", IPAllocation: "Network_vnet1_Reservation_res1", AllocationID: "alloc1"},
			{RowKey: "10.1.0.12_32", IPAllocation: "Network_vnet1_Reservation_res1", AllocationID: "alloc2"},
			{RowKey: "10.1.0.13_32", IPAllocation: "Network_vnet1_Reservation_res1", AllocationID: "alloc3"},

			{RowKey: "10.1.0.20_32", IPAllocation: "Network_vnet1_Reservation_res2", AllocationID: "res2"},
			{RowKey: "10.1.0.21_32", IPAllocation: "Network_vnet1_Reservation_res2", AllocationID: "alloc4"},
			{RowKey: "10.1.0.22_32", IPAllocation: "Network_vnet1_Reservation_res2", AllocationID: "alloc5"},
			{RowKey: "10.1.0.23_32", IPAllocation: "Network_vnet1_Reservation_res2", AllocationID: "alloc6"},
		},
	}

	reservations := []datamodel.ReservationConfiguration{
		{
			ReservationID:   "res1",
			NetworkID:       "vnet1",
			AllocatedSubnet: "10.1.0.10/32",
			SecondaryAllocations: map[string]string{
				"alloc1": "10.1.0.11/32",
				"alloc2": "10.1.0.12/32",
				"alloc3": "10.1.0.13/32",
			},
		},
		{
			ReservationID:   "res2",
			NetworkID:       "vnet1",
			AllocatedSubnet: "10.1.0.20/32",
			SecondaryAllocations: map[string]string{
				"alloc4": "10.1.0.21/32",
				"alloc5": "10.1.0.22/32",
				"alloc6": "10.1.0.23/32",
			},
		},
	}

	resRepo := fakes.NewReservationRepo()
	for _, res := range reservations {
		key := datamodel.GetReservationKey(res.NetworkID, res.ReservationID)
		reservationRow := datamodel.ReservationRow{
			RowKey: key,
			Configuration: &res,
			SubnetKey: datamodel.GetSubnetKey(res.NetworkID, res.AllocationRequest.SubnetName),
		}
		resRepo.Data[key] = reservationRow
	}

	sh := subnetHandler{
		reservations: resRepo,
		allocator: &IPAllocator{
			Allocations:  &allocsRepo,
			Logger:       new(dummyLogger),
			MaxBatchSize: 100,
		},
		logger: new(dummyLogger),
		subnetInfo: subnetInfo{
			vnetID:     vnetID,
			subnetName: subnet,
			baseSubnet: baseSubnet,
		},
		maxBatchSize: 100,
	}

	tbds := map[string][]string{
		"res1": {"alloc2", "alloc3"},
		"res2": {"alloc4", "alloc6"},
	}

	output, err := sh.deallocateBatch(reservations, tbds)
	if err != nil {
		t.Fatalf("did not expect error: %v", err)
	}

	remainingKeys := make(map[string]struct{})
	for _, row := range allocsRepo.Data {
		remainingKeys[row.RowKey] = struct{}{}
	}

	expectedRemovedAllocKeys := []string{"10.1.0.12_32", "10.1.0.13_32", "10.1.0.21_32", "10.1.0.23_32"}
	for _, key := range expectedRemovedAllocKeys {
		if _, found := remainingKeys[key]; found {
			t.Errorf("expected key %q to have been deleted from allocations", key)
		}
	}

	modifiedReservations := make(map[string]datamodel.ReservationConfiguration)
	for _, res := range output {
		modifiedReservations[res.ReservationID] = res
	}

	for _, res := range modifiedReservations {
		deletedAllocs := tbds[res.ReservationID]
		for _, allocID := range deletedAllocs {
			if _, found := res.SecondaryAllocations[allocID]; found {
				t.Errorf("expected allocation id %s to have been removed from reservation %s", allocID, res.ReservationID)
			}
		}
	}
}

func assertCompleteResponses(t *testing.T, respChans []chan response, expectedSecondaryIPCount int) {
	for _, respChan := range respChans {
		select {
		case resp := <-respChan:
			if resp.err != nil {
				t.Fatalf("did not expect an error in response, got %v", resp.err)
			}
			assertCompleteReservation(t, resp.reservation, expectedSecondaryIPCount)
		default: // don't block since we expect all channels to be buffered and contain a response
			t.Fatal("response channel for request did not contain response")
		}
	}
}

func assertCompleteReservation(t *testing.T, rsv datamodel.ReservationConfiguration, expectedSecondaryIPCount int) {
	if rsv.AllocatedSubnet == "" {
		t.Fatal("expected allocated subnet (primary ip) to not be empty")
	}
	if len(rsv.SecondaryAllocations) != expectedSecondaryIPCount {
		t.Fatalf("expected secondary ip count to be %d, got %d", expectedSecondaryIPCount, len(rsv.SecondaryAllocations))
	}
	for _, ip := range rsv.SecondaryAllocations {
		if ip == "" {
			t.Fatal("expected secondary allocation to not be empty")
		}
	}
}

func generateBatch(reservationCount, secondaryIPCount int, vnetID string) (requestSet, []chan response) {
	set := make(requestSet)
	var chs []chan response

	for i := 0; i < reservationCount; i++ {
		secAllocs := make(map[string]string)
		for i := 0; i < secondaryIPCount; i++ {
			secAllocs[uuid.New().String()] = ""
		}

		// we buffer the response channel, so we don't have to spawn goroutines and can check sequentially
		responseCh := make(chan response, 1)
		chs = append(chs, responseCh)

		set.put(reservationRequest{
			reservation: datamodel.ReservationConfiguration{
				ReservationID:        uuid.New().String(),
				NetworkID:            vnetID,
				SecondaryAllocations: secAllocs,
			},
			responseCh: responseCh,
		})
	}

	return set, chs
}
