// Copyright 2017 Microsoft. All rights reserved.
// MIT License

package network

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
	// Prefix for bridge names.
	bridgePrefix = "azure"
	// Virtual MAC address used by Azure VNET.
	virtualMacAddress     = "12:34:56:78:9a:bc"
	versionID             = "VERSION_ID"
	distroID              = "ID"
	ubuntuStr             = "ubuntu"
	dnsServersStr         = "DNS Servers"
	ubuntuVersion18       = 18
	defaultDnsServerIP    = "168.63.129.16"
	systemdResolvConfFile = "/run/systemd/resolve/resolv.conf"
	SnatBridgeIPKey       = "snatBridgeIP"
	LocalIPKey            = "localIP"
	InfraVnetIPKey        = "infraVnetIP"
	OptVethName           = "vethname"
)

const (
	lineDelimiter  = "\n"
	colonDelimiter = ":"
	dotDelimiter   = "."
)

// Linux implementation of route.
type route netlink.Route

// NewNetworkImpl creates a new container network.
func (nm *networkManager) newNetworkImpl(nwInfo *NetworkInfo, extIf *externalInterface) (*network, error) {
	// Connect the external interface.
	var vlanid int
	opt, _ := nwInfo.Options[genericData].(map[string]interface{})
	log.Printf("opt %+v options %+v", opt, nwInfo.Options)

	switch nwInfo.Mode {
	case opModeTunnel:
		fallthrough
	case opModeBridge:
		log.Printf("create bridge")
		if err := nm.connectExternalInterface(extIf, nwInfo); err != nil {
			return nil, err
		}

		if opt != nil && opt[VlanIDKey] != nil {
			vlanid, _ = strconv.Atoi(opt[VlanIDKey].(string))
		}
	case opModeTransparent:
		break
	default:
		return nil, errNetworkModeInvalid
	}

	// Create the network object.
	nw := &network{
		Id:               nwInfo.Id,
		Mode:             nwInfo.Mode,
		Endpoints:        make(map[string]*endpoint),
		extIf:            extIf,
		VlanId:           vlanid,
		DNS:              nwInfo.DNS,
		EnableSnatOnHost: nwInfo.EnableSnatOnHost,
	}

	return nw, nil
}

// DeleteNetworkImpl deletes an existing container network.
func (nm *networkManager) deleteNetworkImpl(nw *network) error {
	var networkClient NetworkClient

	if nw.VlanId != 0 {
		networkClient = NewOVSClient(nw.extIf.BridgeName, nw.extIf.Name, "", nw.DNS.Servers, nw.EnableSnatOnHost)
	} else {
		networkClient = NewLinuxBridgeClient(nw.extIf.BridgeName, nw.extIf.Name, nw.Mode)
	}

	// Disconnect the interface if this was the last network using it.
	if len(nw.extIf.Networks) == 1 {
		nm.disconnectExternalInterface(nw.extIf, networkClient)
	}

	return nil
}

//  SaveIPConfig saves the IP configuration of an interface.
func (nm *networkManager) saveIPConfig(hostIf *net.Interface, extIf *externalInterface) error {
	// Save the default routes on the interface.
	routes, err := netlink.GetIpRoute(&netlink.Route{Dst: &net.IPNet{}, LinkIndex: hostIf.Index})
	if err != nil {
		log.Printf("[net] Failed to query routes: %v.", err)
		return err
	}

	for _, r := range routes {
		if r.Dst == nil {
			if r.Family == unix.AF_INET {
				extIf.IPv4Gateway = r.Gw
			} else if r.Family == unix.AF_INET6 {
				extIf.IPv6Gateway = r.Gw
			}
		}

		extIf.Routes = append(extIf.Routes, (*route)(r))
	}

	// Save global unicast IP addresses on the interface.
	addrs, err := hostIf.Addrs()
	for _, addr := range addrs {
		ipAddr, ipNet, err := net.ParseCIDR(addr.String())
		ipNet.IP = ipAddr
		if err != nil {
			continue
		}

		if !ipAddr.IsGlobalUnicast() {
			continue
		}

		extIf.IPAddresses = append(extIf.IPAddresses, ipNet)

		log.Printf("[net] Deleting IP address %v from interface %v.", ipNet, hostIf.Name)

		err = netlink.DeleteIpAddress(hostIf.Name, ipAddr, ipNet)
		if err != nil {
			break
		}
	}

	log.Printf("[net] Saved interface IP configuration %+v.", extIf)

	return err
}

func getMajorVersion(version string) (int, error) {
	versionSplit := strings.Split(version, dotDelimiter)
	if len(versionSplit) > 0 {
		retrieved_version, err := strconv.Atoi(versionSplit[0])
		if err != nil {
			return 0, err
		}

		return retrieved_version, err
	}

	return 0, fmt.Errorf("[net] Error getting major version")
}

func isGreaterOrEqaulUbuntuVersion(versionToMatch int) bool {
	osInfo, err := platform.GetOSDetails()
	if err != nil {
		log.Printf("[net] Unable to get OS Details: %v", err)
		return false
	}

	version := osInfo[versionID]
	distro := osInfo[distroID]

	if strings.EqualFold(distro, ubuntuStr) {
		version = strings.Trim(version, "\"")
		log.Printf("[net] Ubuntu version %s", version)

		retrieved_version, err := getMajorVersion(version)
		if err != nil {
			log.Printf("[net] Not setting dns. Unable to retrieve major version: %v", err)
			return false
		}

		if retrieved_version >= versionToMatch {
			return true
		}
	}

	return false
}

func readDnsServerIP(ifName string) (string, error) {
	cmd := fmt.Sprintf("systemd-resolve --status %s", ifName)
	out, err := platform.ExecuteCommand(cmd)
	if err != nil {
		return "", err
	}

	lineArr := strings.Split(out, lineDelimiter)
	if len(lineArr) <= 0 {
		return "", fmt.Errorf("[net] Output of cmd %s returned nothing", cmd)
	}

	for _, line := range lineArr {
		if strings.Contains(line, dnsServersStr) {
			dnsServerSplit := strings.Split(line, colonDelimiter)
			if len(dnsServerSplit) > 1 {
				return dnsServerSplit[1], nil
			}
		}
	}

	return "", fmt.Errorf("[net] Dns server ip not set on interface %v", ifName)
}

func saveDnsConfig(extIf *externalInterface) {
	var err error

	extIf.DNSServerIP, err = readDnsServerIP(extIf.Name)
	if err != nil {
		log.Printf("[net] Failed to read dns server IP from interface %v: %v", extIf.Name, err)
		extIf.DNSServerIP = defaultDnsServerIP
		return
	}

	extIf.DNSServerIP = strings.TrimSpace(extIf.DNSServerIP)
	log.Printf("[net] Saved DNS server IP %v from %v", extIf.DNSServerIP, extIf.Name)
}

// ApplyIPConfig applies a previously saved IP configuration to an interface.
func (nm *networkManager) applyIPConfig(extIf *externalInterface, targetIf *net.Interface) error {
	// Add IP addresses.
	for _, addr := range extIf.IPAddresses {
		log.Printf("[net] Adding IP address %v to interface %v.", addr, targetIf.Name)

		err := netlink.AddIpAddress(targetIf.Name, addr.IP, addr)
		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "file exists") {
			log.Printf("[net] Failed to add IP address %v: %v.", addr, err)
			return err
		}
	}

	// Add IP routes.
	for _, route := range extIf.Routes {
		route.LinkIndex = targetIf.Index

		log.Printf("[net] Adding IP route %+v.", route)

		err := netlink.AddIpRoute((*netlink.Route)(route))
		if err != nil {
			log.Printf("[net] Failed to add IP route %v: %v.", route, err)
			return err
		}
	}

	return nil
}

func applyDnsConfig(extIf *externalInterface, ifName string) error {
	cmd := fmt.Sprintf("systemd-resolve --interface=%s --set-dns=%s", ifName, extIf.DNSServerIP)
	_, err := platform.ExecuteCommand(cmd)
	return err
}

// ConnectExternalInterface connects the given host interface to a bridge.
func (nm *networkManager) connectExternalInterface(extIf *externalInterface, nwInfo *NetworkInfo) error {
	var err error
	var networkClient NetworkClient
	log.Printf("[net] Connecting interface %v.", extIf.Name)
	defer func() { log.Printf("[net] Connecting interface %v completed with err:%v.", extIf.Name, err) }()

	// Check whether this interface is already connected.
	if extIf.BridgeName != "" {
		log.Printf("[net] Interface is already connected to bridge %v.", extIf.BridgeName)
		return nil
	}

	// Find the external interface.
	hostIf, err := net.InterfaceByName(extIf.Name)
	if err != nil {
		return err
	}

	// If a bridge name is not specified, generate one based on the external interface index.
	bridgeName := nwInfo.BridgeName
	if bridgeName == "" {
		bridgeName = fmt.Sprintf("%s%d", bridgePrefix, hostIf.Index)
	}

	opt, _ := nwInfo.Options[genericData].(map[string]interface{})
	if opt != nil && opt[VlanIDKey] != nil {
		snatBridgeIP := ""

		if opt != nil && opt[SnatBridgeIPKey] != nil {
			snatBridgeIP, _ = opt[SnatBridgeIPKey].(string)
		}

		networkClient = NewOVSClient(bridgeName, extIf.Name, snatBridgeIP, nwInfo.DNS.Servers, nwInfo.EnableSnatOnHost)
	} else {
		networkClient = NewLinuxBridgeClient(bridgeName, extIf.Name, nwInfo.Mode)
	}

	// Check if the bridge already exists.
	bridge, err := net.InterfaceByName(bridgeName)
	if err != nil {
		// Create the bridge.
		if err := networkClient.CreateBridge(); err != nil {
			log.Printf("Error while creating bridge %+v", err)
			return err
		}

		// On failure, delete the bridge.
		defer func() {
			if err != nil {
				networkClient.DeleteBridge()
			}
		}()

		bridge, err = net.InterfaceByName(bridgeName)
		if err != nil {
			return err
		}
	} else {
		// Use the existing bridge.
		log.Printf("[net] Found existing bridge %v.", bridgeName)
	}

	// Save host IP configuration.
	err = nm.saveIPConfig(hostIf, extIf)
	if err != nil {
		log.Printf("[net] Failed to save IP configuration for interface %v: %v.", hostIf.Name, err)
	}

	isGreaterOrEqualUbuntu18 := isGreaterOrEqaulUbuntuVersion(ubuntuVersion18)
	if isGreaterOrEqualUbuntu18 {
		log.Printf("[net] Saving dns config from %v", extIf.Name)
		saveDnsConfig(extIf)
	}

	// External interface down.
	log.Printf("[net] Setting link %v state down.", hostIf.Name)
	err = netlink.SetLinkState(hostIf.Name, false)
	if err != nil {
		return err
	}

	// Connect the external interface to the bridge.
	log.Printf("[net] Setting link %v master %v.", hostIf.Name, bridgeName)
	if err := networkClient.SetBridgeMasterToHostInterface(); err != nil {
		return err
	}

	// External interface up.
	log.Printf("[net] Setting link %v state up.", hostIf.Name)
	err = netlink.SetLinkState(hostIf.Name, true)
	if err != nil {
		return err
	}

	// Bridge up.
	log.Printf("[net] Setting link %v state up.", bridgeName)
	err = netlink.SetLinkState(bridgeName, true)
	if err != nil {
		return err
	}

	// Add the bridge rules.
	err = networkClient.AddL2Rules(extIf)
	if err != nil {
		return err
	}

	// External interface hairpin on.
	log.Printf("[net] Setting link %v hairpin on.", hostIf.Name)
	if err := networkClient.SetHairpinOnHostInterface(true); err != nil {
		return err
	}

	// Apply IP configuration to the bridge for host traffic.
	err = nm.applyIPConfig(extIf, bridge)
	if err != nil {
		log.Printf("[net] Failed to apply interface IP configuration: %v.", err)
		return err
	}

	if isGreaterOrEqualUbuntu18 {
		log.Printf("[net] Applying dns config on %v", bridgeName)

		if err := applyDnsConfig(extIf, bridgeName); err != nil {
			log.Printf("[net] Failed to apply DNS configuration: %v.", err)
			//return err
		}

		log.Printf("[net] Applied dns IP %v on %v", extIf.DNSServerIP, bridgeName)
	}

	extIf.BridgeName = bridgeName
	log.Printf("[net] Connected interface %v to bridge %v.", extIf.Name, extIf.BridgeName)

	return nil
}

// DisconnectExternalInterface disconnects a host interface from its bridge.
func (nm *networkManager) disconnectExternalInterface(extIf *externalInterface, networkClient NetworkClient) {
	log.Printf("[net] Disconnecting interface %v.", extIf.Name)

	log.Printf("[net] Deleting bridge rules")
	// Delete bridge rules set on the external interface.
	networkClient.DeleteL2Rules(extIf)

	log.Printf("[net] Deleting bridge")
	// Delete Bridge
	networkClient.DeleteBridge()

	extIf.BridgeName = ""
	log.Printf("Restoring ipconfig with primary interface %v", extIf.Name)

	// Restore IP configuration.
	hostIf, _ := net.InterfaceByName(extIf.Name)
	err := nm.applyIPConfig(extIf, hostIf)
	if err != nil {
		log.Printf("[net] Failed to apply IP configuration: %v.", err)
	}

	extIf.IPAddresses = nil
	extIf.Routes = nil

	log.Printf("[net] Disconnected interface %v.", extIf.Name)
}

func getNetworkInfoImpl(nwInfo *NetworkInfo, nw *network) {
	if nw.VlanId != 0 {
		vlanMap := make(map[string]interface{})
		vlanMap[VlanIDKey] = strconv.Itoa(nw.VlanId)
		nwInfo.Options[genericData] = vlanMap
	}
}

func AddStaticRoute(ip string, interfaceName string) error {
	log.Printf("[ovs] Adding %v static route", ip)
	var routes []RouteInfo
	_, ipNet, _ := net.ParseCIDR(ip)
	gwIP := net.ParseIP("0.0.0.0")
	route := RouteInfo{Dst: *ipNet, Gw: gwIP}
	routes = append(routes, route)
	if err := addRoutes(interfaceName, routes); err != nil {
		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "file exists") {
			log.Printf("addroutes failed with error %v", err)
			return err
		}
	}

	return nil
}
