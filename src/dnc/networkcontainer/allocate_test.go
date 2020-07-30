package networkcontainer

import (
	"dnc/api"
	"dnc/datamodel"
	"dnc/networkcontainer/fakes"
	"net"
	"testing"
)

func Test_Allocate(t *testing.T) {
	allocsRepo := fakes.AllocationRepo{
		Data: []struct {
			Key   string
			Value interface{}
		}{
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
		Payload: datamodel.SubnetConfiguration{
			NetworkID:  "network1",
			SubnetName: "subnet1",
			IPSubnet:   "172.16.0.0/24",
		},
	}

	allocator := IPAllocator{
		Allocations:            &allocsRepo,
		Subnets:                &subnetRepo,
		Reservations:           fakes.NewReservationRepo(),
		EnableSubnetDelegation: true,
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
		Data: []struct {
			Key   string
			Value interface{}
		}{
			{"172.16.0.13_32", map[string]interface{}{ipAllocProperty: "Network_network1_Reservation_res1", allocIDProperty: reservationID}},
			{"172.16.0.20_32", map[string]interface{}{ipAllocProperty: "Network_network1_Reservation_res1", allocIDProperty: allocationID1}},
		},
		MaxPageSize: 5,
	}

	subnetRepo := fakes.SubnetRepo{
		Payload: datamodel.SubnetConfiguration{
			NetworkID:  "network1",
			SubnetName: "subnet1",
			IPSubnet:   "172.16.0.0/24",
		},
	}

	allocator := IPAllocator{
		Allocations:            &allocsRepo,
		Subnets:                &subnetRepo,
		Reservations:           fakes.NewReservationRepo(),
		EnableSubnetDelegation: true,
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
