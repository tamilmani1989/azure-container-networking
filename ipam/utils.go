package ipam

import "net"

// retrieves all IP's from a given subnet
func getIPsFromAddresses(ip net.IP, ipnet *net.IPNet) []net.IP {
	ips := make([]net.IP, 0)
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incrementIP(ip) {
		ips = append(ips, duplicateIP(ip))
	}

	return ips
}

// increment the IP by one
func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// Create a copy of a net.IP, used to populate an slice of IP's in a subnet
// net.IP is slice of bytes, slices are reference types, when calculating every
// ip in a subnet, need to save current net.IP slice in it's own memory.
func duplicateIP(ip net.IP) net.IP {
	dup := make(net.IP, len(ip))
	copy(dup, ip)
	return dup
}
