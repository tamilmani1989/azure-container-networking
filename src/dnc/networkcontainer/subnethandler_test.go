package networkcontainer

import (
	"dnc/datamodel"
	"dnc/logger/loggermock"
	"dnc/networkcontainer/fakes"
	"net"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSubnetHandlerProcess_CreationFromScratch(t *testing.T) {
	allocsRepo := fakes.AllocationRepo{}
	resRepo := fakes.NewReservationRepo()

	_, baseSubnet, _ := net.ParseCIDR("10.1.0.0/16")
	vnetID := "vnet1"
	subnet := "subnet1"

	logger := loggermock.NewDummyLogger()

	subnetInfo := subnetInfo{
		vnetID:     vnetID,
		subnetName: subnet,
		baseSubnet: baseSubnet,
	}

	cfg := handlerConfig{
		subnetInfo:   subnetInfo,
		reservations: resRepo,
		scaleStrategy: &scanSubnetStrategy{
			allocator:  &IPAllocator{Allocations: &allocsRepo, Logger: logger, MaxBatchSize: 100},
			subnetInfo: subnetInfo,
			logger:     logger,
		},
		logger:       logger,
		maxBatchSize: 100,
	}

	sh := newSubnetHandler(cfg)

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
	require.Len(t, reservationIPs, expectedAllocationCount)
	require.Len(t, allocsRepo.Data, expectedAllocationCount)

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
		Data: []datamodel.IPAllocationEntity{
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

	logger := loggermock.NewDummyLogger()

	subnetInfo := subnetInfo{
		vnetID:     vnetID,
		subnetName: subnet,
		baseSubnet: baseSubnet,
	}

	cfg := handlerConfig{
		subnetInfo:   subnetInfo,
		reservations: resRepo,
		scaleStrategy: &scanSubnetStrategy{
			allocator:  &IPAllocator{Allocations: &allocsRepo, Logger: logger, MaxBatchSize: 100},
			subnetInfo: subnetInfo,
			logger:     logger,
		},
		logger:       logger,
		maxBatchSize: 100,
	}

	sh := newSubnetHandler(cfg)

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
	require.Len(t, reservationIPs, expectedAllocationCount)
	require.Len(t, allocsRepo.Data, expectedAllocationCount)

	assertCompleteResponses(t, respChans, secondaryIPCount)
}

func assertCompleteResponses(t *testing.T, respChans []chan response, expectedSecondaryIPCount int) {
	for _, respChan := range respChans {
		select {
		case resp := <-respChan:
			require.NoError(t, resp.err)
			assertCompleteReservation(t, resp.reservation, expectedSecondaryIPCount)
		default: // don't block since we expect all channels to be buffered and contain a response
			t.Fatal("response channel for request did not contain response")
		}
	}
}

func assertCompleteReservation(t *testing.T, rsv datamodel.ReservationConfiguration, expectedSecondaryIPCount int) {
	require.NotEmptyf(t, rsv.AllocatedSubnet, "expected allocated subnet (primary ip) to not be empty. reservation %+v", rsv)
	require.Len(t, rsv.SecondaryAllocations, expectedSecondaryIPCount)

	for _, ip := range rsv.SecondaryAllocations {
		require.NotEmpty(t, ip)
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
