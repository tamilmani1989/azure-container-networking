package allocationhelper

import (
	"dnc/api"
	"dnc/logger/loggershim"
	"dnc/utils"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
)

func AllocateSubnetFromNetworkSpace(
	baseSubnet *net.IPNet,
	subnetsToIgnore map[string]*net.IPNet,
	newSubnetPrefix int) (*net.IPNet, error) {
	allocatedSubnets := make(map[uint32]*net.IPNet)
	for _, allocatedSubnet := range subnetsToIgnore {
		firstAllocatedIPAdress := GetFirstAddressAsUInt32InSubnet(allocatedSubnet)
		allocatedSubnets[firstAllocatedIPAdress] = allocatedSubnet
	}
	newSubnet, err := allocateSubnetFromNetworkSpace(
		baseSubnet,
		allocatedSubnets, newSubnetPrefix)

	return newSubnet, err
}
func allocateSubnetFromNetworkSpace(
	baseSubnet *net.IPNet,
	allocatedSubnetsMap map[uint32]*net.IPNet,
	newSubnetPrefix int) (*net.IPNet, error) {
	// Locate the first allocated subnet contained within the vnetSubnet.
	var index int
	var hasAllocatedSubnet bool
	var keys []uint32
	for k := range allocatedSubnetsMap {
		keys = append(keys, k)
	}
	sort.Sort(utils.UInts(keys))

	baseSubnetMaskInUInt32 := utils.GetSubnetMask(baseSubnet.Mask)
	baseHostMaskInUInt32 := utils.GetHostMask(baseSubnet.Mask)
	baseSubnetAddressInUInt32 := utils.IP2Int(baseSubnet.IP)
	for index < len(keys) {
		allocatedSubnetAddressInUInt32 := keys[index]
		if baseSubnetAddressInUInt32 == (allocatedSubnetAddressInUInt32 & baseHostMaskInUInt32) {
			hasAllocatedSubnet = true
			break
		}

		index++
	}
	if hasAllocatedSubnet == false {
		return AllocateSubnetFromAddressRange(
			baseSubnetAddressInUInt32,
			(baseSubnetAddressInUInt32 + baseSubnetMaskInUInt32 + 1),
			newSubnetPrefix)
	}

	// Check the available address ranges between allocated subnets and see if
	// an address range can satisfy the new subnet allocation.
	startingAddress := baseSubnetAddressInUInt32
	loggershim.LS.Log("%d\n", len(keys))
	for index < len(keys) {
		allocatedSubnetAddressInUInt32 := keys[index]
		// If the allocated subnet is outside vnetSubnet, exit the loop.
		if baseSubnetAddressInUInt32 != (allocatedSubnetAddressInUInt32 & baseHostMaskInUInt32) {
			break
		}

		// See if the gap between starting address and the beginning address of
		// the allocated subnet can satisfy the new allocation request.
		subnet, _ := AllocateSubnetFromAddressRange(
			startingAddress,
			allocatedSubnetAddressInUInt32,
			newSubnetPrefix)

		if subnet != nil {
			return subnet, nil
		}

		// Advance the starting address to the end of the allocated subnet.
		allocatedSubnet := allocatedSubnetsMap[allocatedSubnetAddressInUInt32]
		startingAddress =
			(allocatedSubnetAddressInUInt32 +
				utils.GetSubnetMask(allocatedSubnet.Mask) + 1)
		index++
	}
	// No allocation possible from the gaps between starting address and the beginning
	// address of each allocated subnet. Look at the gap from the ending address of
	// the last allocated subnet to the end of vnetSubnet.

	index--
	startingAddress =
		(keys[index] + utils.GetSubnetMask(allocatedSubnetsMap[keys[index]].Mask) + 1)

	return AllocateSubnetFromAddressRange(
		startingAddress,
		(baseSubnetAddressInUInt32 + utils.GetSubnetMask(baseSubnet.Mask) + 1),
		newSubnetPrefix)

}

func GetLastAdressAsUInt32InSubnet(ipSubnet *net.IPNet) uint32 {
	startingIPAddressAsUInt32 := GetFirstAddressAsUInt32InSubnet(ipSubnet)
	subnetHostCount := ^binary.BigEndian.Uint32(ipSubnet.Mask)

	return startingIPAddressAsUInt32 + subnetHostCount
}

func GetLastAdressInSubnet(ipSubnet *net.IPNet) net.IP {
	return utils.Int2IP(GetLastAdressAsUInt32InSubnet(ipSubnet))
}

func GetFirstAddressAsUInt32InSubnet(ipSubnet *net.IPNet) uint32 {
	var hostBitCount uint32
	prefix, _ := ipSubnet.Mask.Size()

	hostBitCount = (32 - (uint32(prefix)))
	// Align the starting address to the size of the subnet to allocate.
	mask := net.CIDRMask(prefix, 32)

	subnetHostCount := ^binary.BigEndian.Uint32(mask) + 1
	startingAddress := utils.IP2Int(ipSubnet.IP)
	startingAddress = ((startingAddress + subnetHostCount - 1) >> hostBitCount)
	startingAddress <<= hostBitCount
	return startingAddress
}

func GetFirstAddressInSubnet(ipSubnet *net.IPNet) net.IP {
	return utils.Int2IP(GetFirstAddressAsUInt32InSubnet(ipSubnet))
}

func GetDNSServerAddressesForSubnet(ipSubnet *net.IPNet) []string {
	dnsServerAddresses := make([]string, 2)
	dnsServerAddresses[0] = utils.Int2IP(GetFirstAddressAsUInt32InSubnet(ipSubnet) + 2).String()
	dnsServerAddresses[1] = utils.Int2IP(GetFirstAddressAsUInt32InSubnet(ipSubnet) + 3).String()

	return dnsServerAddresses
}

func AllocateSubnetFromAddressRange(
	startingAddress uint32,
	endingAddress uint32,
	subnetPrefix int) (*net.IPNet, error) {
	// Determine the number of bits for host part of the IP address.
	var hostBitCount uint32
	hostBitCount = (32 - (uint32(subnetPrefix)))
	loggershim.LS.Log("hostBitCount: %d\n", hostBitCount)
	// Align the starting address to the size of the subnet to allocate.
	mask := net.CIDRMask(subnetPrefix, 32)
	loggershim.LS.Log("Mask: %+v\n", mask)
	subnetHostCount := ^binary.BigEndian.Uint32(mask) + 1

	startingAddress = ((startingAddress + subnetHostCount - 1) >> hostBitCount)
	startingAddress <<= hostBitCount

	// Check if there are enough addresses in the range.
	if (startingAddress + subnetHostCount) <= endingAddress {
		// There are enough addresses, create a new subnet and return it.
		var subnet net.IPNet
		subnet.IP = utils.Int2IP(startingAddress)
		subnet.Mask = mask
		return &subnet, nil
	}

	return nil, nil
}

func GetAllAddressInSubnet(ipSubnet *net.IPNet) ([]net.IP, *api.JsonErr) {
	startingIPAddressAsUInt32 := GetFirstAddressAsUInt32InSubnet(ipSubnet)
	endingIPAddressAsUInt32 := ^binary.BigEndian.Uint32(ipSubnet.Mask) + startingIPAddressAsUInt32
	addresses := make([]net.IP, endingIPAddressAsUInt32-startingIPAddressAsUInt32+1)
	index := 0
	for i := startingIPAddressAsUInt32; i <= endingIPAddressAsUInt32; i++ {
		addresses[index] = utils.Int2IP(i)
		index++
	}

	return addresses, nil
}

func GetIPSubnetForIPAllocationConstraint(ipConstraint string) (*net.IPNet, *api.JsonErr) {
	isIPConstraintASubnet := strings.Contains(ipConstraint, utils.PrefixDelimiter)
	var ipSubnetConstraint string
	if !isIPConstraintASubnet {
		ipSubnetConstraint = ipConstraint + utils.PrefixDelimiter + utils.IPV4DefaultPrefix
	} else {
		ipSubnetConstraint = ipConstraint
	}

	constraintIPSubnet, err := utils.ParseIPNet(ipSubnetConstraint)
	if err != nil {
		return nil, &api.JsonErr{
			Code:       api.InvalidIPConstraint,
			HTTPStatus: http.StatusBadRequest,
			Text:       fmt.Sprintf("Invalid IP constraint:%s", ipSubnetConstraint)}
	}

	return constraintIPSubnet, nil
}

func IsSubnetContainedInBaseSubnet(baseIPSubnet *net.IPNet, ipConstraint *net.IPNet) (bool, *api.JsonErr) {
	ipAddressesToAllocate, _ := GetAllAddressInSubnet(ipConstraint)
	for ipIndex := range ipAddressesToAllocate {
		ipToAllocate := ipAddressesToAllocate[ipIndex]
		if !baseIPSubnet.Contains(ipToAllocate) {
			return false,
				&api.JsonErr{
					Code:       api.InvalidIPConstraint,
					HTTPStatus: http.StatusBadRequest,
					Text:       fmt.Sprintf("IP:%s in the requested range %s is not in subnet %s", ipToAllocate, ipConstraint, baseIPSubnet.String())}
		}
	}

	return true, nil
}
func IsRequestedRangeAllocatable(baseIPSubnet *net.IPNet, allocatedSubnets map[string]*net.IPNet, ipConstraint *net.IPNet) (bool, *api.JsonErr) {
	ipAddressesToAllocate, _ := GetAllAddressInSubnet(ipConstraint)
	for ipIndex := range ipAddressesToAllocate {
		ipToAllocate := ipAddressesToAllocate[ipIndex]
		if !baseIPSubnet.Contains(ipToAllocate) {
			return false, &api.JsonErr{
				Code:       api.InvalidIPConstraint,
				HTTPStatus: http.StatusBadRequest,
				Text:       fmt.Sprintf("IP:%s in the requested range %s is not in subnet %s", ipToAllocate, ipConstraint, baseIPSubnet.String())}
		}

		for _, allocatedSubnet := range allocatedSubnets {
			loggershim.LS.Log("IPToAllocate: %+v AllocatedSubnets: %+v\n", ipToAllocate, allocatedSubnet)
			if allocatedSubnet.Contains(ipToAllocate) {
				return false, &api.JsonErr{
					Code:       api.InvalidIPConstraint,
					HTTPStatus: http.StatusBadRequest,
					Text:       fmt.Sprintf("IP:%s in the requested range %s overlaps with allocated reserved subnet %s", ipToAllocate, ipConstraint, allocatedSubnet.String())}
			}
		}
	}

	return true, nil
}
