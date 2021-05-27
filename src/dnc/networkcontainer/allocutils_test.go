package networkcontainer

import (
	"dnc/datamodel"
	"net"
	"testing"
)

func Test_FindAllocatableIPs(t *testing.T) {
	_, baseSubnet, _ := net.ParseCIDR("10.0.1.0/24")

	t.Run("interspersed previous allocations", func(t *testing.T) {
		allocations := make(map[string]net.IP)
		allocations["10.0.1.0"] = net.ParseIP("10.0.1.0")
		allocations["10.0.1.1"] = net.ParseIP("10.0.1.1")
		allocations["10.0.1.5"] = net.ParseIP("10.0.1.5")
		allocations["10.0.1.10"] = net.ParseIP("10.0.1.10")
		allocations["10.0.1.15"] = net.ParseIP("10.0.1.15")
		allocations["10.0.1.255"] = net.ParseIP("10.0.1.255")
		count := 10

		var allocatedIPs []net.IP
		for _, allocatedIP := range allocations {
			allocatedIPs = append(allocatedIPs, allocatedIP)
		}

		newIPs, err := findAllocatableIPs(baseSubnet, allocatedIPs, count)
		if err != nil {
			t.Fatalf("did not expect error: %+v", err)
		}
		if len(newIPs) != count {
			t.Fatalf("expected %d new ips, got %d", count, len(newIPs))
		}
		for _, ip := range newIPs {
			if _, ok := allocations[ip.String()]; ok {
				t.Errorf("new ip %s conflicts with existing allocation", ip.String())
			}
			if !baseSubnet.Contains(ip) {
				t.Errorf("new ip %s is not within subnet %s", ip.String(), baseSubnet.String())
			}
		}
	})

	t.Run("no previous allocations", func(t *testing.T) {
		count := subnetMaxHostCount(baseSubnet)
		newIPs, err := findAllocatableIPs(baseSubnet, nil, count)
		if err != nil {
			t.Fatalf("did not expect error: %+v", err)
		}
		if len(newIPs) != count {
			t.Fatalf("expected %d new ips, got %d", count, len(newIPs))
		}
		for _, ip := range newIPs {
			if !baseSubnet.Contains(ip) {
				t.Errorf("new ip %s is not within subnet %s", ip.String(), baseSubnet.String())
			}
		}
	})
}

func Test_FindFreeIPs_Error(t *testing.T) {
	_, baseSubnet, _ := net.ParseCIDR("10.0.1.0/29") // should have 8 max hosts
	count := subnetMaxHostCount(baseSubnet) + 1
	_, err := findAllocatableIPs(baseSubnet, nil, count)
	if err != errCapacityExceeded {
		t.Fatal("expected errCapacityExceeded error")
	}
}

func TestReconcileReservation(t *testing.T) {
	currState := map[string]*net.IPNet{
		"key1": {IP: net.IP{10, 0, 0, 4}, Mask: net.IPMask{255, 255, 255, 255}},
		"key2": {IP: net.IP{10, 0, 0, 5}, Mask: net.IPMask{255, 255, 255, 255}},
		"key3": {IP: net.IP{10, 0, 0, 6}, Mask: net.IPMask{255, 255, 255, 255}},
	}

	t.Run("reservation already in sync with allocations", func(t *testing.T) {
		res := datamodel.ReservationRow{
			RowKey:        datamodel.GetReservationKey("vnet1", "key1"),
			Configuration: &datamodel.ReservationConfiguration{
				ReservationID:   "key1",
				AllocatedSubnet: "10.0.0.4/32",
				SecondaryAllocations: map[string]string{
					"key2": "10.0.0.5/32",
					"key3": "10.0.0.6/32"}},
			SubnetKey:     datamodel.GetReservationKey("vnet1", "subnet1"),
			SetKey:        "",
		}

		completed := reconcileReservation(&res, currState)
		if !completed {
			t.Error("expected reservation to be complete, as it is consistent with current db state")
		}
	})

	t.Run("reservation in sync after reconcile", func(t *testing.T) {
		res := datamodel.ReservationRow{
			RowKey:        datamodel.GetReservationKey("vnet1", "key1"),
			Configuration: &datamodel.ReservationConfiguration{
				ReservationID:   "key1",
				AllocatedSubnet: "",
				SecondaryAllocations: map[string]string{
					"key2": "",
					"key3": ""}},
			SubnetKey:     datamodel.GetReservationKey("vnet1", "subnet1"),
			SetKey:        "",
		}

		completed := reconcileReservation(&res, currState)
		if !completed {
			t.Error("expected reservation to be complete, as it is consistent with current db state")
		}
	})

	t.Run("reservation still needs allocations", func(t *testing.T) {
		res := datamodel.ReservationRow{
			RowKey:        datamodel.GetReservationKey("vnet1", "key1"),
			Configuration: &datamodel.ReservationConfiguration{
				ReservationID:   "key1",
				AllocatedSubnet: "10.0.0.4/32",
				SecondaryAllocations: map[string]string{
					"key2": "10.0.0.5/32",
					"key3": "10.0.0.6/32",
					"key4": "",
					"key5": ""}},
			SubnetKey:     datamodel.GetReservationKey("vnet1", "subnet1"),
			SetKey:        "",
		}

		completed := reconcileReservation(&res, currState)
		if completed {
			t.Error("expected reservation to be incomplete, as allocations still need to happen")
		}
	})
}
