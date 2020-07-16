package utils

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
)

const (
	// PrefixDelimiter - IP Subnet delimiter
	PrefixDelimiter = "/"

	// IPV4DefaultPrefix - IP default prefix
	IPV4DefaultPrefix = "32"
)

// ParseIPNet - Parse IP Subnet string
func ParseIPNet(ipSubnetString string) (*net.IPNet, error) {
	_, ipnet, err := net.ParseCIDR(ipSubnetString)
	if err != nil {
		return nil, err
	}
	return ipnet, nil
}

// IP2Int - Helper to convert IP address to UInt32
func IP2Int(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

// Int2IP - Helper to convert UInt32 to IP address
func Int2IP(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip
}

// UInts - Array of uints
type UInts []uint32

func (u UInts) Len() int { return len(u) }

func (u UInts) Less(i, j int) bool { return u[i] < u[j] }

func (u UInts) Swap(i, j int) {
	u[i], u[j] = u[j], u[i]
}

// GetHostMask - Helper to get host mask from IP Net mask
func GetHostMask(mask net.IPMask) uint32 {
	return binary.BigEndian.Uint32(mask)
}

// GetSubnetMask - Helper to subnet host mask from IP Net mask
func GetSubnetMask(mask net.IPMask) uint32 {
	return ^GetHostMask(mask)
}

// AreIPSubnetsEqual - True if subnets are same.
func AreIPSubnetsEqual(subnet1, subnet2 *net.IPNet) bool {

	subnet1Prefix, _ := subnet1.Mask.Size()
	subnet2Prefix, _ := subnet2.Mask.Size()
	if subnet1Prefix != subnet2Prefix ||
		!(subnet1.IP.Equal(subnet2.IP)) {
		return false
	}

	return true
}

func GetIPAddressStringAndPrefix(ipSubnetString string) (string, int, error) {
	splitIPSubnet := strings.Split(ipSubnetString, "/")

	if len(splitIPSubnet) != 2 {
		err := fmt.Errorf(fmt.Sprintf("Failed to split IpSubnet string: [%s]", ipSubnetString))
		return "", 0, err
	}

	ipAddressString := splitIPSubnet[0]
	prefixInInt, atoiErr := strconv.Atoi(splitIPSubnet[1])

	if atoiErr != nil {
		err := fmt.Errorf(fmt.Sprintf("Failed to acquire subnet prefix from [%s]", splitIPSubnet[1]))
		return "", 0, err
	}

	return ipAddressString, prefixInInt, nil
}
