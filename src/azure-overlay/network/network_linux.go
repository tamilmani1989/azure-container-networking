// Copyright 2017 Microsoft. All rights reserved.

// +build linux

package network

import (
	"azure-overlay/common"
	"azure-overlay/log"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"

	"github.com/vishvananda/netlink"
)

const (
	// Prefix for bridge names.
	bridgePrefix = "azure"

	bridgeName = "azure-overlay"

	genericData = "com.docker.network.generic"

	AzureOvsCommonBridge = "azure-ovs"
)

func (nm *networkManager) overlayCleanup(netns, brName, intBrName string) {

	if brName != "" {
		log.Printf("Deleting container overlay bridge %v", brName)
		common.System("ovs-vsctl", "del-br", brName)
	}

	if intBrName != "" {
		log.Printf("Deleting container internet bridge %v", intBrName)
		common.System("ovs-vsctl", "del-br", intBrName)
		common.System("ovs-vsctl", "del-port", AzureOvsCommonBridge, "v0"+intBrName)
		common.System("ip", "link", "del", "dev", "v0"+intBrName)
	}

	if netns != "" {
		log.Printf("Deleting overlay namespace %v", netns)
		common.System("ip", "netns", "delete", netns)
	}

	if len(nm.ExternalInterfaces) == 0 {
		log.Printf("Deleting common OVS bridge")
		nm.deleteCommonOVSBridge()
	}
}

func (nm *networkManager) createCommonOVSBridge() error {
	err := common.System("ovs-vsctl", "add-br", AzureOvsCommonBridge)
	if err != nil {
		log.Printf("Error while creating common OVS bridge %v", err.Error())
		return err
	}

	err = common.System("ifconfig", AzureOvsCommonBridge, nm.HostIP)
	if err != nil {
		log.Printf("Error while assigning IP %v to common OVS bridge %v", nm.HostIP, err.Error())
		return err
	}

	err = common.System("ifconfig", nm.HostInterface, "0.0.0.0")
	if err != nil {
		log.Printf("Error while removing IP from primary NIC %v", err.Error())
		return err
	}

	err = common.System("ovs-vsctl", "add-port", AzureOvsCommonBridge, nm.HostInterface)
	if err != nil {
		log.Printf("Error while adding primary NIC as port to common OVS bridge %v", err.Error())
		return err
	}

	err = common.System("route", "add", "default", "gw", nm.HostGateway, "dev", AzureOvsCommonBridge)
	if err != nil {
		log.Printf("Error while setting default gw %v to common OVS bridge %v", nm.HostGateway, err.Error())
		return err
	}
	err = common.System("iptables", "-t", "nat", "-A", "POSTROUTING", "-m", "iprange", "!", "--dst-range", "168.63.129.16", "-m", "addrtype", "!", "--dst-type", "local", "!", "-d", nm.VnetSpace, "-j", "MASQUERADE")
	if err != nil {
		log.Printf("Error while setting iptable snat vnet space %v error %v", nm.VnetSpace, err.Error())
		return err
	}
	return nil
}

func findInterfaceByIP(hostIP string) (string, string) {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Printf("Error while retrieving interfaces %v", err.Error())
		return "", ""
	}

	// handle err
	ip := net.ParseIP(hostIP)

	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			log.Printf("Error in getting addrs %v error %v", hostIP, err.Error())
			return "", ""
		}
		// handle err
		for _, address := range addrs {
			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil && reflect.DeepEqual(ipnet.IP, ip) {
					log.Printf("Interface found %v", i.Name)
					return ipnet.String(), i.Name
				}
			}
		}
	}
	return "", ""
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func (nm *networkManager) createOvsInterface(nwInfo *NetworkInfo) *externalInterface {

	if brName, ok := nm.Bridges[nwInfo.Id]; ok {
		log.Printf("Bridge %v already exists for network id %v. Nothing to do.", brName, nwInfo)
		return nil
	}

	brName := fmt.Sprintf("%s%d", bridgeName, nm.BridgeCount+1)
	subnet := nwInfo.Subnets[0].Prefix
	gateway := nwInfo.Subnets[0].Gateway
	// convert CIDR notation into Gateway address as needed by plugin.

	maskStr := fmt.Sprintf("%d.%d.%d.%d", subnet.Mask[0], subnet.Mask[1], subnet.Mask[2], subnet.Mask[3])
	containerGwIP := gateway.String()

	log.Printf("Container gw ip %v", containerGwIP)

	opt, _ := nwInfo.Options[genericData].(map[string]interface{})
	clusterInfo := opt["nodeList"].(string)
	vxlanID := opt["vxlanID"].(string)
	internetBridgeIP := opt["internetBridgeIP"].(string)
	vnetSpace := opt["vnetSpace"].(string)

	nm.VnetSpace = vnetSpace

	_, err := net.InterfaceByName(AzureOvsCommonBridge)
	if err != nil {
		nm.HostIP = opt["localNodeIP"].(string)
		nm.HostGateway = opt["localNodeGateway"].(string)
		nm.HostInterface = opt["localNodeInterface"].(string)
		intBridgeSubnet := opt["internetBridgeSubnet"].(string)

		routes, _ := netlink.RouteList(nil, netlink.FAMILY_V4)

		if nm.HostGateway == "" {
			for _, route := range routes {
				if route.Dst == nil && route.Src == nil {
					nm.HostGateway = route.Gw.String()
					break
				}
			}
		}

		log.Printf("Host gateway %v", nm.HostGateway)

		if nm.HostInterface == "" {
			nm.HostIP, nm.HostInterface = findInterfaceByIP(nm.HostIP)
			if nm.HostInterface == "" {
				log.Printf("Failed to find interface name for this IP %v", nm.HostIP)
				return nil
			}
			log.Printf("Got interface %v for  IP %v", nm.HostInterface, nm.HostIP)
		}

		if nm.HostIP == "" || nm.HostGateway == "" || nm.HostInterface == "" {
			log.Printf("Anyone of HostIP/HostGateway/HostInterface is not specified")
			return nil
		}

		err = nm.createCommonOVSBridge()
		if err != nil {
			log.Printf("Error in creating common ovs bridge with ip %v gateway %v primary ifname %v", nm.HostIP, nm.HostGateway, nm.HostInterface)
			nm.overlayCleanup("", "", "")
			return nil
		}

		log.Printf("Add static route %v in host for internet connectivity", intBridgeSubnet)

		common.System("ip", "route", "del", intBridgeSubnet, "dev", AzureOvsCommonBridge)
		err = common.System("ip", "route", "add", intBridgeSubnet, "dev", AzureOvsCommonBridge)
		if err != nil {
			log.Printf("Error while adding static route for internet bridge space %v error %v", intBridgeSubnet, err.Error())
			nm.overlayCleanup("", "", "")
			return nil
		}
	}

	nodeList := strings.Split(clusterInfo, ",")
	log.Printf("nodes %v", nodeList)

	netnsName := "netns" + brName
	err = common.System("ip", "netns", "add", netnsName)
	if err != nil {
		log.Printf("Error while creating network namespace %v", err.Error())
		nm.overlayCleanup(netnsName, "", "")
		return nil
	}

	var clusterNodeInfoList = make([]string, len(nodeList))
	i := 0

	for _, node := range nodeList {
		clusterNodeInfoList[i] = opt[node].(string)
		i = i + 1
	}

	log.Printf("Nodeinfolist %v", clusterNodeInfoList)
	log.Printf("ovs ip %v name %v", containerGwIP, brName)

	err = common.System("ovs-vsctl", "add-br", brName)
	if err != nil {
		log.Printf("Error while creating overalay bridge for network %v", err.Error())
		nm.overlayCleanup(netnsName, "", "")
		return nil
	}

	hostIf, err := net.InterfaceByName(brName)
	if err != nil {
		nm.overlayCleanup(netnsName, brName, "")
		return nil
	}

	err = common.System("ovs-vsctl", "set", "bridge", brName, "stp_enable=true")
	if err != nil {
		log.Printf("Error while enabling stp on ovs bridge %v", err.Error())
		nm.overlayCleanup(netnsName, brName, "")
		return nil
	}

	for _, node := range clusterNodeInfoList {
		log.Printf("VxlanCount %d", nm.VxlanCount)
		vxlanIf := fmt.Sprintf("%s%d", "vxlan", nm.VxlanCount)
		nodeInfo := strings.Split(node, ",")
		optRemoteIp := fmt.Sprintf("%s:%s=%s", "options", "remote_ip", nodeInfo[0])
		optVxLANID := fmt.Sprintf("%s:%s=%s", "options", "key", vxlanID)
		// options:key=<VNI>
		err = common.System("ovs-vsctl", "add-port", brName, vxlanIf, "--", "set", "interface", vxlanIf, "type=vxlan", optRemoteIp, optVxLANID)
		if err != nil {
			log.Printf("Error while adding vxlan on ovs bridge %v", err.Error())
			nm.overlayCleanup(netnsName, brName, "")
			return nil
		}
		common.System("ip", "route", "add", nodeInfo[1], "dev", brName)
		nm.VxlanCount++
	}

	log.Printf("Assigning %v to new ns ", brName)
	err = common.System("ip", "link", "set", brName, "netns", netnsName)
	if err != nil {
		log.Printf("Error while moving ovs bridge to new network namespace %v", err.Error())
		nm.overlayCleanup(netnsName, brName, "")
		return nil
	}

	err = common.System("ip", "netns", "exec", netnsName, "ifconfig", brName, containerGwIP, "netmask", maskStr, "up")
	if err != nil {
		log.Printf("Error assigning ip address to ovs bridge %v", err.Error())
		nm.overlayCleanup(netnsName, brName, "")
		return nil
	}

	log.Printf("Add gw %v for private vnet space in new namespace", gateway.String())
	err = common.System("ip", "netns", "exec", netnsName, "ip", "route", "add", vnetSpace, "via", gateway.String(), "dev", brName)
	if err != nil {
		log.Printf("Error adding default gw %v for new network namespace %v", gateway.String(), err.Error())
		nm.overlayCleanup(netnsName, brName, "")
		return nil
	}

	internetBridgeName := "intbr" + strconv.Itoa(nm.BridgeCount)
	log.Printf("IP %v", internetBridgeIP)

	err = nm.setupInternetBridge(internetBridgeName, netnsName, internetBridgeIP, vnetSpace)
	if err != nil {
		log.Printf("Error while setting up internet bridge %v with ip %v\n", internetBridgeName, internetBridgeIP)
		nm.overlayCleanup(netnsName, brName, internetBridgeName)
		return nil
	}

	extIf := externalInterface{
		Name:               brName,
		InternetBridgeName: internetBridgeName,
		Networks:           make(map[string]*network),
		MacAddress:         hostIf.HardwareAddr,
		IPv4Gateway:        gateway,
		IPv6Gateway:        net.IPv6unspecified,
	}

	extIf.Subnets = append(extIf.Subnets, subnet.String())
	nm.ExternalInterfaces[brName] = &extIf
	nm.Bridges[nwInfo.Id] = brName

	//TODO: Handle overflow and unique bridge names
	nm.BridgeCount = nm.BridgeCount + 1
	return &extIf
}

func (nm *networkManager) setupInternetBridge(internetBrName, netnsName, netBrIP, vnetSpace string) error {

	err := common.System("ovs-vsctl", "add-br", internetBrName)
	if err != nil {
		log.Printf("Error while creating internet ovs bridge %v", err.Error())
		return err
	}

	err = common.System("ip", "link", "add", "v0"+internetBrName, "type", "veth", "peer", "name", "v1"+internetBrName)
	if err != nil {
		log.Printf("Error while creating veth pairs for internet bridge %v", err.Error())
		return err
	}

	err = common.System("ovs-vsctl", "add-port", internetBrName, "v0"+internetBrName)
	if err != nil {
		log.Printf("Error while adding veth to internet bridge %v", err.Error())
		return err
	}

	err = common.System("ovs-vsctl", "add-port", AzureOvsCommonBridge, "v1"+internetBrName)
	if err != nil {
		log.Printf("Error while adding veth to azure common bridge %v", err.Error())
		return err
	}

	err = common.System("ip", "link", "set", "v0"+internetBrName, "up")
	if err != nil {
		log.Printf("Error while setting veth pair %v up %v", "v0"+internetBrName, err.Error())
		return err
	}

	err = common.System("ip", "link", "set", "v1"+internetBrName, "up")
	if err != nil {
		log.Printf("Error while setting veth pair %v up %v", "v1"+internetBrName, err.Error())
		return err
	}

	err = common.System("ip", "link", "set", internetBrName, "netns", netnsName)
	if err != nil {
		log.Printf("Error while moving internet bridge to new network namespace %v %v", netnsName, err.Error())
		return err
	}

	err = common.System("ip", "netns", "exec", netnsName, "ifconfig", internetBrName, netBrIP, "up")
	if err != nil {
		log.Printf("Error while moving internet bridge to new network namespace %v", err.Error())
		return err
	}

	routeIP := strings.Split(nm.HostIP, "/")
	err = common.System("ip", "netns", "exec", netnsName, "ip", "route", "add", routeIP[0]+"/32", "dev", internetBrName)
	if err != nil {
		log.Printf("Error while adding host IP %v route via internet bridge %v", routeIP[0], err.Error())
		nm.overlayCleanup(internetBrName, "azure-overlay1", netnsName)
		return err
	}

	err = common.System("ip", "netns", "exec", netnsName, "route", "add", "default", "gw", routeIP[0], "dev", internetBrName)
	if err != nil {
		log.Printf("Error while adding default gateway via internet bridge %v", routeIP[0], err.Error())
		return err
	}

	//iptables -t nat -A POSTROUTING -m iprange ! --dst-range 168.63.129.16 -m addrtype ! --dst-type local ! -d {{{vnetCidr}}} -j MASQUERADE
	err = common.System("ip", "netns", "exec", netnsName, "iptables", "-t", "nat", "-A", "POSTROUTING", "-m", "iprange", "!", "--dst-range", "168.63.129.16", "-m", "addrtype", "!", "--dst-type", "local", "!", "-d", vnetSpace, "-j", "MASQUERADE")
	if err != nil {
		log.Printf("Error while adding snat iptables rule inside overlay namespace %v", err.Error())
		return err
	}

	return nil
}

// Linux implementation of route.
type route netlink.Route

// NewNetworkImpl creates a new container network.
func (nm *networkManager) newNetworkImpl(nwInfo *NetworkInfo) (*network, *externalInterface, error) {
	var extIf *externalInterface
	// Connect the external interface.
	switch nwInfo.Mode {
	case opModeOverlay:
		extIf = nm.createOvsInterface(nwInfo)
		if extIf == nil {
			return nil, nil, errOvsNotCreated
		}
	default:
		return nil, nil, errNetworkModeInvalid
	}

	// Create the network object.
	nw := &network{
		Id:        nwInfo.Id,
		Mode:      nwInfo.Mode,
		Endpoints: make(map[string]*endpoint),
		extIf:     extIf,
	}

	return nw, extIf, nil
}

func (nm *networkManager) deleteCommonOVSBridge() {
	err := common.System("ovs-vsctl", "del-br", AzureOvsCommonBridge)
	if err != nil {
		log.Printf("Error while removing common OVS bridge %v", err.Error())
		return
	}

	err = common.System("ifconfig", nm.HostInterface, nm.HostIP, "up")
	if err != nil {
		log.Printf("Error while assigning IP  %v back to primary NIC %v", nm.HostIP, err.Error())
		return
	}

	err = common.System("route", "add", "default", "gw", nm.HostGateway, "dev", nm.HostInterface)
	if err != nil {
		log.Printf("Error while setting default gw %v back to common OVS bridge %v", nm.HostGateway, err.Error())
		return
	}
	err = common.System("iptables", "-t", "nat", "-D", "POSTROUTING", "-m", "iprange", "!", "--dst-range", "168.63.129.16", "-m", "addrtype", "!", "--dst-type", "local", "!", "-d", nm.VnetSpace, "-j", "MASQUERADE")
	if err != nil {
		log.Printf("Error while deleting iptable snat vnet space %v error %v", nm.VnetSpace, err.Error())
		return
	}
}

// DeleteNetworkImpl deletes an existing container network.
func (nm *networkManager) deleteNetworkImpl(nw *network) error {
	// Delete the ovs bridge

	if len(nw.extIf.Networks) == 1 {
		delete(nm.Bridges, nw.Id) // does not return anything
		delete(nm.ExternalInterfaces, nw.extIf.Name)
		netnsName := "netns" + nw.extIf.Name
		nm.overlayCleanup(netnsName, nw.extIf.Name, nw.extIf.InternetBridgeName)
	}

	return nil
}
