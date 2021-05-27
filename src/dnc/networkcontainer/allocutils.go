package networkcontainer

import (
	"dnc/datamodel"
	"dnc/utils"
	"encoding/binary"
	"errors"
	"net"
	"sort"
)

var errCapacityExceeded = errors.New("subnet capacity exceeded")

// findAllocatableIPs returns ips within boundaries of a subnet that are currently not in use.
func findAllocatableIPs(baseSubnet *net.IPNet, ipsInUse []net.IP, count int) ([]net.IP, error) {
	if count < 1 {
		return nil, nil
	}

	currAllocCount := len(ipsInUse)
	subnetBlockSize := subnetMaxHostCount(baseSubnet)
	if count > subnetBlockSize-currAllocCount {
		return nil, errCapacityExceeded
	}

	loIPBound := int(utils.IP2Int(baseSubnet.IP)) - 1
	hiIPBound := loIPBound + subnetBlockSize + 1
	encodedIPs := []int{loIPBound, hiIPBound}
	for _, ip := range ipsInUse {
		if !baseSubnet.Contains(ip) {
			continue
		}
		encodedIPs = append(encodedIPs, int(utils.IP2Int(ip)))
	}
	sort.Ints(encodedIPs)

	var newIPs []net.IP
FOR:
	for i := 0; i < len(encodedIPs)-1; i++ {
		lo, hi := encodedIPs[i], encodedIPs[i+1]
		if lo+1 == hi { // scan until non-contiguous ips
			continue
		}

		for newIP := lo + 1; newIP < hi; newIP++ {
			newIPs = append(newIPs, utils.Int2IP(uint32(newIP)))
			if len(newIPs) == count {
				break FOR
			}
		}
	}
	return newIPs, nil
}

func subnetMaxHostCount(subnet *net.IPNet) int {
	return int(^binary.BigEndian.Uint32(subnet.Mask)) + 1
}

// reconcileReservation reconciles a reservation with the latest allocation state, and returns whether it is ready for completion
// if not ready, the reservation still contains the correct state with which to continue allocation
func 	reconcileReservation(req *datamodel.ReservationRow, currState map[string]*net.IPNet) bool {
	allocatedIP := currState[req.Configuration.ReservationID]
	if allocatedIP != nil {
		req.Configuration.AllocatedSubnet = allocatedIP.String()
	}

	incomplete := false
	for allocID, ipAddress := range req.Configuration.SecondaryAllocations {
		secIP := currState[allocID]
		if secIP != nil {
			req.Configuration.SecondaryAllocations[allocID] = secIP.String()
			continue
		}
		if ipAddress != "" {
			// the allocation was removed, but the reservation was not mutated
			delete(req.Configuration.SecondaryAllocations, allocID)
			continue
		}
		incomplete = true
	}

	return req.Configuration.AllocatedSubnet != "" && !incomplete
}
