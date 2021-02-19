package networkcontainer

import (
	"context"
	"dnc/api"
	"dnc/datamodel"
	"dnc/logger/loggermock"
	"dnc/networkcontainer/fakes"
	"testing"
)

func TestDispatcher(t *testing.T) {
	vnetID := "vnet1"

	subnets := fakes.SubnetRepo{
		Data: map[string]datamodel.SubnetConfiguration{
			datamodel.GetSubnetKey(vnetID, "subnet1"): {NetworkID: vnetID, SubnetName: "subnet1", IPSubnet: "10.0.1.0/24"},
			datamodel.GetSubnetKey(vnetID, "subnet2"): {NetworkID: vnetID, SubnetName: "subnet2", IPSubnet: "10.0.2.0/24"},
			datamodel.GetSubnetKey(vnetID, "subnet3"): {NetworkID: vnetID, SubnetName: "subnet3", IPSubnet: "10.0.3.0/24"},
			datamodel.GetSubnetKey(vnetID, "subnet4"): {NetworkID: vnetID, SubnetName: "subnet4", IPSubnet: "10.0.4.0/24"},
			datamodel.GetSubnetKey(vnetID, "subnet5"): {NetworkID: vnetID, SubnetName: "subnet5", IPSubnet: "10.0.5.0/24"},
		},
	}

	alloc := IPAllocator{
		Nodes:        &fakes.NodeRepo{},
		NCs:          &fakes.NCRepo{},
		Allocations:  &fakes.AllocationRepo{},
		Reservations: fakes.NewReservationRepo(),
		Subnets:      &subnets,
		Logger:       loggermock.NewDummyLogger(),
		MaxBatchSize: 100,
	}

	deps := DispatcherDependencies{
		Logger:       loggermock.NewDummyLogger(),
		Reservations: fakes.NewReservationRepo(),
		Allocator:    &alloc,
		Subnets:      &subnets,
	}

	dispatcherMaxBatchSize := 100

	d := NewIPReservationDispatcher(deps, dispatcherMaxBatchSize, false)

	d.Run(context.Background())

	for _, subnet := range subnets.Data {
		for i := 0; i < 50; i++ {
			res1 := datamodel.ReservationConfiguration{NetworkID: vnetID, AllocationRequest: api.AllocationRequest{SubnetName: subnet.SubnetName}}
			_, _ = d.Reserve(res1, nil)
		}
	}

	if len(d.subnetHandlers) != len(subnets.Data) {
		t.Fatalf("expected %d subnet handlers to have been added, got %d", len(subnets.Data), len(d.subnetHandlers))
	}
}
