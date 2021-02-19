package networkcontainer

import (
	"dnc/datamodel"
	"dnc/logger/loggermock"
	"dnc/networkcontainer/fakes"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanSubnetStrategyScaleDown(t *testing.T) {
	_, baseSubnet, _ := net.ParseCIDR("10.1.0.0/16")
	vnetID := "vnet1"
	subnet := "subnet1"

	allocsRepo := fakes.AllocationRepo{
		Data: []datamodel.IPAllocationEntity{
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

	sh := scanSubnetStrategy{
		allocator: &IPAllocator{
			Allocations:  &allocsRepo,
			Logger:       loggermock.NewDummyLogger(),
			MaxBatchSize: 100,
		},
		subnetInfo: subnetInfo{
			vnetID:     vnetID,
			subnetName: subnet,
			baseSubnet: baseSubnet,
		},
		logger: loggermock.NewDummyLogger(),
	}

	tbds := map[string][]string{
		"res1": {"alloc2", "alloc3"},
		"res2": {"alloc4", "alloc6"},
	}

	output, err := sh.scaleDown(reservations, tbds)
	require.NoError(t, err)

	remainingKeys := make(map[string]struct{})
	for _, row := range allocsRepo.Data {
		remainingKeys[row.RowKey] = struct{}{}
	}

	expectedRemovedAllocKeys := []string{"10.1.0.12_32", "10.1.0.13_32", "10.1.0.21_32", "10.1.0.23_32"}
	for _, key := range expectedRemovedAllocKeys {
		assert.NotContainsf(t, remainingKeys, key, "expected key %q to have been deleted from allocations", key)
	}

	modifiedReservations := make(map[string]datamodel.ReservationConfiguration)
	for _, res := range output {
		modifiedReservations[res.ReservationID] = res
	}

	for _, res := range modifiedReservations {
		deletedAllocs := tbds[res.ReservationID]
		for _, allocID := range deletedAllocs {
			assert.NotContainsf(t, res.SecondaryAllocations, allocID, "expected allocation id %s to have been removed from reservation %s", allocID, res.ReservationID)
		}
	}
}
