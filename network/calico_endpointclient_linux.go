package network

import (
	"fmt"
	"net"

	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/netlink"
	"github.com/Azure/azure-container-networking/network/epcommon"
	"github.com/Azure/azure-container-networking/platform"
)

const (
	FAKE_GW_IP = "169.254.1.1/32"
	DEFAULT_GW = "0.0.0.0/0"
)

type CalicoEndpointClient struct {
	bridgeName        string
	hostPrimaryIfName string
	hostVethName      string
	containerVethName string
	hostPrimaryMac    net.HardwareAddr
	containerMac      net.HardwareAddr
	hostVethMac       net.HardwareAddr
	mode              string
}

func NewCalicoEndpointClient(
	extIf *externalInterface,
	hostVethName string,
	containerVethName string,
	mode string,
) *CalicoEndpointClient {

	client := &CalicoEndpointClient{
		bridgeName:        extIf.BridgeName,
		hostPrimaryIfName: extIf.Name,
		hostVethName:      hostVethName,
		containerVethName: containerVethName,
		hostPrimaryMac:    extIf.MacAddress,
		mode:              mode,
	}

	return client
}

func (client *CalicoEndpointClient) AddEndpoints(epInfo *EndpointInfo) error {
	if err := epcommon.CreateEndpoint(client.hostVethName, client.containerVethName); err != nil {
		return err
	}

	containerIf, err := net.InterfaceByName(client.containerVethName)
	if err != nil {
		return err
	}

	client.containerMac = containerIf.HardwareAddr

	hostVethIf, err := net.InterfaceByName(client.hostVethName)
	if err != nil {
		return err
	}

	client.hostVethMac = hostVethIf.HardwareAddr

	return nil
}

func (client *CalicoEndpointClient) AddEndpointRules(epInfo *EndpointInfo) error {
	var routeInfoList []RouteInfo

	// ip route add <podip> dev <hostveth>
	// This route is needed for incoming packets to pod to route via hostveth
	for _, ipAddr := range epInfo.IPAddresses {
		var routeInfo RouteInfo
		ipNet := net.IPNet{IP: ipAddr.IP, Mask: net.CIDRMask(32, 32)}
		log.Printf("[net] Adding route for the ip %v", ipNet.String())
		routeInfo.Dst = ipNet
		routeInfoList = append(routeInfoList, routeInfo)
		if err := addRoutes(client.hostVethName, routeInfoList); err != nil {
			return err
		}
	}
	return nil
}

func (client *CalicoEndpointClient) DeleteEndpointRules(ep *endpoint) {
	var routeInfoList []RouteInfo

	// ip route del <podip> dev <hostveth>
	// Deleting the route set up for routing the incoming packets to pod
	for _, ipAddr := range ep.IPAddresses {
		var routeInfo RouteInfo
		ipNet := net.IPNet{IP: ipAddr.IP, Mask: net.CIDRMask(32, 32)}
		log.Printf("[net] Deleting route for the ip %v", ipNet.String())
		routeInfo.Dst = ipNet
		routeInfoList = append(routeInfoList, routeInfo)
		deleteRoutes(client.hostVethName, routeInfoList)
	}
}

func (client *CalicoEndpointClient) MoveEndpointsToContainerNS(epInfo *EndpointInfo, nsID uintptr) error {
	// Move the container interface to container's network namespace.
	log.Printf("[net] Setting link %v netns %v.", client.containerVethName, epInfo.NetNsPath)
	if err := netlink.SetLinkNetNs(client.containerVethName, nsID); err != nil {
		return err
	}

	return nil
}

func (client *CalicoEndpointClient) SetupContainerInterfaces(epInfo *EndpointInfo) error {
	if err := epcommon.SetupContainerInterface(client.containerVethName, epInfo.IfName); err != nil {
		return err
	}

	client.containerVethName = epInfo.IfName

	return nil
}

func (client *CalicoEndpointClient) ConfigureContainerInterfacesAndRoutes(epInfo *EndpointInfo) error {
	if err := epcommon.AssignIPToInterface(client.containerVethName, epInfo.IPAddresses); err != nil {
		return err
	}

	var routeInfo RouteInfo
	var routeInfoList []RouteInfo

	// ip route add 169.254.1.1/32 dev eth0 scope link
	gwIP, gwIPNet, _ := net.ParseCIDR(FAKE_GW_IP)
	routeInfo.Dst = *gwIPNet
	routeInfo.Scope = netlink.RT_SCOPE_LINK
	routeInfoList = append(routeInfoList, routeInfo)

	// ip route add default gw 169.254.1.1 dev eth0
	routeInfo = RouteInfo{}
	_, defIPNet, _ := net.ParseCIDR(DEFAULT_GW)
	routeInfo.Dst = *defIPNet
	routeInfo.Gw = net.ParseIP(gwIP.String())
	routeInfoList = append(routeInfoList, routeInfo)

	// add the above routes
	if err := addRoutes(client.containerVethName, routeInfoList); err != nil {
		return err
	}

	routeInfoList = routeInfoList[:0]

	// Removing the route added by setipaddress while assigning IP to interface
	for _, ipAddr := range epInfo.IPAddresses {
		routeInfo = RouteInfo{}
		ip, ipNet, _ := net.ParseCIDR(ipAddr.String())
		log.Printf("Removing route %v", ipNet.String())
		routeInfo.Dst = *ipNet
		routeInfo.Scope = netlink.RT_SCOPE_LINK
		routeInfo.Src = ip
		routeInfo.Protocol = netlink.RTPROT_KERNEL
		routeInfoList = append(routeInfoList, routeInfo)
	}

	// delete the above route
	if err := deleteRoutes(client.containerVethName, routeInfoList); err != nil {
		log.Printf("Deleting route failed with err %v", err)
	}

	// set arp entry for fake gateway in pod
	arpCmd := fmt.Sprintf("arp -s 169.254.1.1 %v", client.hostVethMac.String())
	_, err := platform.ExecuteCommand(arpCmd)
	if err != nil {
		log.Printf("Setting static arp for ip 169.254.1.1 mac %v failed with error %v", client)
	}

	return nil
}

func (client *CalicoEndpointClient) DeleteEndpoints(ep *endpoint) error {
	log.Printf("[net] Deleting veth pair %v %v.", ep.HostIfName, ep.IfName)
	err := netlink.DeleteLink(ep.HostIfName)
	if err != nil {
		log.Printf("[net] Failed to delete veth pair %v: %v.", ep.HostIfName, err)
		return err
	}

	return nil
}
