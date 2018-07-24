package servicetunnel

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/netlink"
	"github.com/Azure/azure-container-networking/platform"
	"golang.org/x/sys/unix"
)

const (
	Add = "allow"
	Del = "delete allow"
)

const (
	routePrefix = "fd00::/8"
	fakeGateway = "fe80::944:63c6:6ed9:2336"
	hostMac     = "12:34:56:78:9a:bc"
	ipV6Prefix  = "2603:10e1:100:2:"
)

type ServiceTunnel struct {
	PrimaryInterfaceName string
	TargetInterfaceName  string
	IPv4Addresses        []string
	IPv6Address          string
	IPv4Gateway          string
	IPv6Gateway          string
	VethName1            string
	VethName2            string
	VethMac              string
	Routes               []*netlink.Route
}

func (st *ServiceTunnel) InstallTools() error {
	if _, err := platform.ExecuteCommand("apt -y update"); err != nil {
		return err
	}

	if _, err := platform.ExecuteCommand("apt install -y ebtables"); err != nil {
		return err
	}

	return nil
}

func (st *ServiceTunnel) GetPrimaryIP() error {
	if st.PrimaryInterfaceName == "" {
		return fmt.Errorf("PrimaryInterfaceName not initialized")
	}

	hostIf, err := net.InterfaceByName(st.PrimaryInterfaceName)
	if err != nil {
		log.Printf("Failed to get interface %v", err)
		return err
	}

	addrs, err := hostIf.Addrs()
	for _, addr := range addrs {
		ipAddr, _, _ := net.ParseCIDR(addr.String())
		if ipAddr.IsGlobalUnicast() && ipAddr.To4() != nil {
			st.IPv4Addresses = append(st.IPv4Addresses, addr.String())
		}
	}

	return nil
}

func (st *ServiceTunnel) GetIPv6Address() error {
	if len(st.IPv4Addresses) <= 0 || st.IPv4Addresses[0] == "" {
		return fmt.Errorf("Ipv4Address is empty")
	}

	ipaddr, _, _ := net.ParseCIDR(st.IPv4Addresses[0])
	ipSplit := strings.Split(ipaddr.String(), ".")
	ipLen := len(ipSplit)
	st.IPv6Address = ipV6Prefix

	for i := 0; i < ipLen; i++ {
		if i%2 == 0 {
			iphex, _ := strconv.Atoi(ipSplit[i])
			st.IPv6Address = st.IPv6Address + ":" + fmt.Sprintf("%x", iphex)
		} else {
			iphex, _ := strconv.Atoi(ipSplit[i])
			st.IPv6Address = st.IPv6Address + fmt.Sprintf("%02x", iphex)
		}
	}

	return nil
}

func (st *ServiceTunnel) GetRoutesAndGateway() error {
	if st.PrimaryInterfaceName == "" {
		return fmt.Errorf("PrimaryInterfaceName not initialized")
	}

	hostIf, err := net.InterfaceByName(st.PrimaryInterfaceName)
	if err != nil {
		return err
	}

	routes, err := netlink.GetIpRoute(&netlink.Route{Dst: &net.IPNet{}, LinkIndex: hostIf.Index})
	if err != nil {
		return err
	}

	for _, r := range routes {
		if r.Dst == nil {
			if r.Family == unix.AF_INET {
				st.IPv4Gateway = r.Gw.String()
			} else if r.Family == unix.AF_INET6 {
				st.IPv6Gateway = r.Gw.String()
			}
		}

		st.Routes = append(st.Routes, (*netlink.Route)(r))
	}

	return nil
}

func (st *ServiceTunnel) CreateSTBridge() error {
	if st.TargetInterfaceName == "" {
		return fmt.Errorf("BridgeName not initialized")
	}

	link := &netlink.BridgeLink{
		LinkInfo: netlink.LinkInfo{
			Type: netlink.LINK_TYPE_BRIDGE,
			Name: st.TargetInterfaceName,
		},
	}

	return netlink.AddLink(link)
}

func (st *ServiceTunnel) ApplyIPAddresses() error {
	if st.TargetInterfaceName == "" || len(st.IPv4Addresses) <= 0 {
		return fmt.Errorf("BridgeName/IPv4Addr not specified")
	}

	for _, addr := range st.IPv4Addresses {
		ip, ipnet, _ := net.ParseCIDR(addr)

		err := netlink.AddIpAddress(st.TargetInterfaceName, ip, ipnet)
		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "file exists") {
			return fmt.Errorf("[net] Failed to add IP address %v: %v.", ip, err)
		}
	}

	return netlink.SetLinkState(st.TargetInterfaceName, true)
}

func (st *ServiceTunnel) ApplyRoutes() error {
	targetIf, err := net.InterfaceByName(st.TargetInterfaceName)
	if err != nil {
		return err
	}

	for _, route := range st.Routes {
		route.LinkIndex = targetIf.Index

		err := netlink.AddIpRoute((*netlink.Route)(route))
		if err != nil {
			return fmt.Errorf("[net] Failed to add IP route %v: %v.", route, err)
		}
	}

	return nil
}

func (st *ServiceTunnel) RemoveIP() error {
	if st.PrimaryInterfaceName == "" || len(st.IPv4Addresses) <= 0 {
		return fmt.Errorf("Primary Interface/ipv4addr not specified")
	}

	ip, ipnet, _ := net.ParseCIDR(st.IPv4Addresses[0])

	return netlink.DeleteIpAddress(st.PrimaryInterfaceName, ip, ipnet)
}

func (st *ServiceTunnel) SetBridgeMasterInterface(slaveInterface string) error {
	if err := netlink.SetLinkState(slaveInterface, false); err != nil {
		return err
	}

	if err := netlink.SetLinkMaster(slaveInterface, st.TargetInterfaceName); err != nil {
		return err
	}

	if err := netlink.SetLinkState(slaveInterface, true); err != nil {
		return err
	}

	return nil
}

func (st *ServiceTunnel) CreateVethPair(veth1, veth2 string) error {
	st.VethName1 = veth1
	st.VethName2 = veth2

	link := netlink.VEthLink{
		LinkInfo: netlink.LinkInfo{
			Type: netlink.LINK_TYPE_VETH,
			Name: veth1,
		},
		PeerName: veth2,
	}

	if err := netlink.AddLink(&link); err != nil {
		return err
	}

	if err := netlink.SetLinkState(veth1, true); err != nil {
		return err
	}

	if err := netlink.SetLinkState(veth2, true); err != nil {
		return err
	}

	return nil
}

func (st *ServiceTunnel) ApplyIPv6ToVeth() error {
	if st.VethName2 == "" {
		return fmt.Errorf("Veth not created")
	}

	ipv6WithPrefix := st.IPv6Address + "/128"
	ip, ipnet, _ := net.ParseCIDR(ipv6WithPrefix)

	err := netlink.AddIpAddress(st.VethName2, ip, ipnet)
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "file exists") {
		return fmt.Errorf("[net] Failed to add IP address %v: %v.", ip, err)
	}

	return nil
}

func (st *ServiceTunnel) GetVethMac() error {
	if st.VethName2 == "" {
		return fmt.Errorf("Veth not created")
	}

	hostIf, err := net.InterfaceByName(st.VethName2)
	if err != nil {
		return err
	}

	st.VethMac = hostIf.HardwareAddr.String()
	return nil
}

func (st *ServiceTunnel) SetFakeIPv6Route() error {
	if st.PrimaryInterfaceName == "" {
		return fmt.Errorf("Primary Interface not specified")
	}

	route := &netlink.Route{}

	targetIf, err := net.InterfaceByName(st.PrimaryInterfaceName)
	if err != nil {
		return err
	}

	route.Family = unix.AF_INET6
	_, route.Dst, _ = net.ParseCIDR(routePrefix)
	route.Gw = net.ParseIP(fakeGateway)
	route.LinkIndex = targetIf.Index

	if err = netlink.AddIpRoute(route); err != nil {
		return fmt.Errorf("[net] Failed to add IP route %v: %v.", route, err)
	}

	return nil
}

func (st *ServiceTunnel) SetFakeIPv6Neighbor() error {
	if st.PrimaryInterfaceName == "" {
		return fmt.Errorf("Primary Interface not specified")
	}

	cmd := fmt.Sprintf("ip -6 neigh add %v lladdr %v dev %s", fakeGateway, hostMac, st.PrimaryInterfaceName)
	log.Printf("Executing cmd: %v", cmd)
	_, err := platform.ExecuteCommand(cmd)

	return err
}

func (st *ServiceTunnel) DeleteFakeIPv6Route() error {
	if st.TargetInterfaceName == "" {
		return fmt.Errorf("Primary Interface not specified")
	}

	route := &netlink.Route{}

	targetIf, err := net.InterfaceByName(st.TargetInterfaceName)
	if err != nil {
		return err
	}

	route.Family = unix.AF_INET6
	_, route.Dst, _ = net.ParseCIDR(routePrefix)
	route.Gw = net.ParseIP(fakeGateway)
	route.LinkIndex = targetIf.Index

	if err = netlink.DeleteIpRoute(route); err != nil {
		return fmt.Errorf("[net] Failed to delete IP route %v: %v.", route, err)
	}

	return nil
}

func (st *ServiceTunnel) DeleteFakeIPv6Neighbor() error {
	if st.TargetInterfaceName == "" {
		return fmt.Errorf("Primary Interface not specified")
	}

	cmd := fmt.Sprintf("ip -6 neigh del %v lladdr %v dev %s", fakeGateway, hostMac, st.TargetInterfaceName)
	_, err := platform.ExecuteCommand(cmd)

	return err
}

func (st *ServiceTunnel) ApplyMacDnatRule(action string) error {
	if st.IPv6Address == "" || st.VethMac == "" {
		return fmt.Errorf("Either ipv6 or veth mac addr is empty")
	}

	cmd := fmt.Sprintf("ebtables -t nat -%s PREROUTING -i eth0 -p ip6 --ip6-destination %s -j dnat --to-dst %s --dnat-target ACCEPT",
		action, st.IPv6Address, st.VethMac)
	if _, err := platform.ExecuteCommand(cmd); err != nil {
		return err
	}

	return nil
}

func (st *ServiceTunnel) DeleteVethPair() error {
	return netlink.DeleteLink(st.VethName1)
}

func (st *ServiceTunnel) DeleteSTBridge(disconnect bool) error {
	if disconnect {
		err := netlink.SetLinkMaster(st.TargetInterfaceName, "")
		if err != nil {
			log.Printf("[net] Failed to disconnect interface %v from bridge, err:%v.", st.TargetInterfaceName, err)
			return err
		}
	}

	// Delete the bridge.
	return netlink.DeleteLink(st.PrimaryInterfaceName)
}

func (st *ServiceTunnel) ConfgureFirewallRules(cmd string, portsToOpen []int) error {
	if len(st.IPv4Addresses) <= 0 {
		return fmt.Errorf("IPV6 not specified")
	}

	ip, _, _ := net.ParseCIDR(st.IPv4Addresses[0])

	for _, port := range portsToOpen {
		ufwCmd := fmt.Sprintf("ufw %v to %v port %v", cmd, ip.String(), port)
		_, err := platform.ExecuteCommand(ufwCmd)
		if err != nil {
			log.Printf("add ufw command failed with error %v", err)
			return err
		}

		ufwCmd = fmt.Sprintf("ufw %v to %v port %v", cmd, st.IPv6Address, port)
		_, err = platform.ExecuteCommand(ufwCmd)
		if err != nil {
			log.Printf("add ufw command failed with error %v", err)
			return err
		}
	}

	if cmd == Add {
		if err := addIPv6ForwardRule(st.IPv6Address); err != nil {
			log.Printf("Adding ip6table forward rule failed with error %v", err)
			return err
		}
	} else if cmd == Del {
		if err := deleteIPv6ForwardRule(st.IPv6Address); err != nil {
			log.Printf("Deleting ip6table forward rule failed with error %v", err)
			return err
		}
	}

	return nil
}

func addIPv6ForwardRule(ipv6Address string) error {
	cmd := fmt.Sprintf("ip6tables -t filter -C FORWARD -d %v -j ACCEPT", ipv6Address)
	_, err := platform.ExecuteCommand(cmd)
	if err == nil {
		log.Printf("ip6table forward rule already exists")
		return nil
	}

	cmd = fmt.Sprintf("ip6tables -t filter -I FORWARD 1 -d %v -j ACCEPT", ipv6Address)
	_, err = platform.ExecuteCommand(cmd)
	return err
}

func deleteIPv6ForwardRule(ipv6Address string) error {
	cmd := fmt.Sprintf("ip6tables -t filter -D FORWARD -d %v -j ACCEPT", ipv6Address)
	_, err := platform.ExecuteCommand(cmd)
	return err
}
