// Copyright 2017 Microsoft. All rights reserved.
// MIT License

// +build linux

package network

import (
	"fmt"
	"net"
	"strings"

	"github.com/Azure/azure-container-networking/common"
	"github.com/Azure/azure-container-networking/ebtables"
	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/netlink"
)

const (
	// Common prefix for all types of host network interface names.
	commonInterfacePrefix = "az"

	// Prefix for host virtual network interface names.
	hostVEthInterfacePrefix = commonInterfacePrefix + "veth"

	// Prefix for container network interface names.
	containerInterfacePrefix = "eth"
)

func (nw *network) setupContainerNicAndRules(hostIfName string, contIfName string, containerIf *net.Interface, epInfo *EndpointInfo) error {
	// Connect host interface to the bridge.
	multitenancy := false
	var err error

	if epInfo.Data != nil {
		if vlanid, ok := epInfo.Data["vlanid"]; ok {
			multitenancy = true
			log.Printf("[net] Setting link %v master %v.", hostIfName, nw.extIf.BridgeName)

			cmd := fmt.Sprintf("ovs-vsctl add-port %v %v tag=%v", nw.extIf.BridgeName, hostIfName, vlanid)
			_, err = common.ExecuteShellCommand(cmd)
			if err != nil {
				log.Printf("[net] Adding port failed with error %v", err)
				return err
			}

			log.Printf("[net] Get ovs port for interface %v.", hostIfName)
			cmd = fmt.Sprintf("ovs-vsctl get Interface %s ofport", hostIfName)
			containerPort, err := common.ExecuteShellCommand(cmd)
			if err != nil {
				log.Printf("[net] Get ofport failed with error %v", err)
				return err
			}

			containerPort = strings.Trim(containerPort, "\n")
			mac := nw.extIf.MacAddress.String()
			macHex := strings.Replace(mac, ":", "", -1)
			log.Printf("[net] OVS - Adding ARP SNAT rule for egress traffic on %v.", hostIfName)

			cmd = fmt.Sprintf(`ovs-ofctl add-flow %v priority=10,arp,in_port=%s,arp_op=1,actions='mod_dl_src:%s,
				load:0x%s->NXM_NX_ARP_SHA[],normal'`, nw.extIf.BridgeName, containerPort, mac, macHex)
			_, err = common.ExecuteShellCommand(cmd)
			if err != nil {
				log.Printf("[net] Adding ARP SNAT rule failed with error %v", err)
				return err
			}

			log.Printf("[net] OVS - Adding IP SNAT rule for egress traffic on %v.", hostIfName)

			cmd = fmt.Sprintf("ovs-ofctl add-flow %v priority=10,ip,in_port=%s,actions=mod_dl_src:%s,normal",
				nw.extIf.BridgeName, containerPort, mac)
			_, err = common.ExecuteShellCommand(cmd)
			if err != nil {
				log.Printf("[net] Adding IP SNAT rule failed with error %v", err)
				return err
			}

			macAddr := containerIf.HardwareAddr.String()
			macAddrHex := strings.Replace(macAddr, ":", "", -1)

			log.Printf("[net] Get ovs port for interface %v.", nw.extIf.Name)
			cmd = fmt.Sprintf("ovs-vsctl get Interface %s ofport", nw.extIf.Name)
			ofport, err := common.ExecuteShellCommand(cmd)
			if err != nil {
				log.Printf("[net] Get ofport failed with error %v", err)
				return err
			}

			ofport = strings.Trim(ofport, "\n")

			for _, ipAddr := range epInfo.IPAddresses {
				// Add ARP reply rule.
				ipAddrInt := common.IpToInt(ipAddr.IP)

				log.Printf("[net] Adding ARP reply rule for IP address %v on %v.", ipAddr.IP.String(), contIfName)
				cmd = fmt.Sprintf(`ovs-ofctl add-flow %s arp,arp_tpa=%s,dl_vlan=%v,arp_op=1,actions='load:0x2->NXM_OF_ARP_OP[],
					move:NXM_OF_ETH_SRC[]->NXM_OF_ETH_DST[],mod_dl_src:%s,
					move:NXM_NX_ARP_SHA[]->NXM_NX_ARP_THA[],move:NXM_OF_ARP_SPA[]->NXM_OF_ARP_TPA[],
					load:0x%s->NXM_NX_ARP_SHA[],load:0x%x->NXM_OF_ARP_SPA[],IN_PORT'`,
					nw.extIf.BridgeName, ipAddr.IP.String(), vlanid, macAddr, macAddrHex, ipAddrInt)
				_, err = common.ExecuteShellCommand(cmd)
				if err != nil {
					log.Printf("[net] Adding ARP reply rule failed with error %v", err)
					return err
				}

				// Add MAC address translation rule.
				log.Printf("[net] Adding MAC DNAT rule for IP address %v on %v.", ipAddr.IP.String(), contIfName)
				cmd = fmt.Sprintf("ovs-ofctl add-flow %s ip,nw_dst=%s,dl_vlan=%v,in_port=%s,actions=mod_dl_dst:%s,normal",
					nw.extIf.BridgeName, ipAddr.IP.String(), vlanid, ofport, macAddr)
				_, err = common.ExecuteShellCommand(cmd)
				//err = ebtables.SetDnatForIPAddress(nw.extIf.Name, ipAddr.IP, containerIf.HardwareAddr, ebtables.Append)
				if err != nil {
					log.Printf("[net] Adding MAC DNAT rule failed with error %v", err)
					return err
				}
			}
		}
	}

	if !multitenancy {
		log.Printf("[net] Setting link %v master %v.", hostIfName, nw.extIf.BridgeName)
		err = netlink.SetLinkMaster(hostIfName, nw.extIf.BridgeName)
		if err != nil {
			return err
		}

		for _, ipAddr := range epInfo.IPAddresses {
			// Add ARP reply rule.
			log.Printf("[net] Adding ARP reply rule for IP address %v on %v.", ipAddr.String(), contIfName)
			err = ebtables.SetArpReply(ipAddr.IP, nw.getArpReplyAddress(containerIf.HardwareAddr), ebtables.Append)
			if err != nil {
				return err
			}

			// Add MAC address translation rule.
			log.Printf("[net] Adding MAC DNAT rule for IP address %v on %v.", ipAddr.String(), contIfName)
			err = ebtables.SetDnatForIPAddress(nw.extIf.Name, ipAddr.IP, containerIf.HardwareAddr, ebtables.Append)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// newEndpointImpl creates a new endpoint in the network.
func (nw *network) newEndpointImpl(epInfo *EndpointInfo) (*endpoint, error) {
	var containerIf *net.Interface
	var ns *Namespace
	var ep *endpoint
	var err error

	if nw.Endpoints[epInfo.Id] != nil {
		log.Printf("[net] Endpoint alreday exists.")
		err = errEndpointExists
		return nil, err
	}

	// Create a veth pair.
	hostIfName := fmt.Sprintf("%s%s", hostVEthInterfacePrefix, epInfo.Id[:7])
	contIfName := fmt.Sprintf("%s%s-2", hostVEthInterfacePrefix, epInfo.Id[:7])

	log.Printf("[net] Creating veth pair %v %v.", hostIfName, contIfName)

	link := netlink.VEthLink{
		LinkInfo: netlink.LinkInfo{
			Type: netlink.LINK_TYPE_VETH,
			Name: contIfName,
		},
		PeerName: hostIfName,
	}

	err = netlink.AddLink(&link)
	if err != nil {
		log.Printf("[net] Failed to create veth pair, err:%v.", err)
		return nil, err
	}

	// On failure, delete the veth pair.
	defer func() {
		if err != nil {
			netlink.DeleteLink(contIfName)
		}
	}()

	//
	// Host network interface setup.
	//

	// Host interface up.
	log.Printf("[net] Setting link %v state up.", hostIfName)
	err = netlink.SetLinkState(hostIfName, true)
	if err != nil {
		return nil, err
	}

	containerIf, err = net.InterfaceByName(contIfName)
	if err != nil {
		return nil, err
	}

	nw.setupContainerNicAndRules(hostIfName, contIfName, containerIf, epInfo)
	//
	// Container network interface setup.
	//

	// Query container network interface info.

	// Setup rules for IP addresses on the container interface.

	// If a network namespace for the container interface is specified...
	if epInfo.NetNsPath != "" {
		// Open the network namespace.
		log.Printf("[net] Opening netns %v.", epInfo.NetNsPath)
		ns, err = OpenNamespace(epInfo.NetNsPath)
		if err != nil {
			return nil, err
		}
		defer ns.Close()

		// Move the container interface to container's network namespace.
		log.Printf("[net] Setting link %v netns %v.", contIfName, epInfo.NetNsPath)
		err = netlink.SetLinkNetNs(contIfName, ns.GetFd())
		if err != nil {
			return nil, err
		}

		// Enter the container network namespace.
		log.Printf("[net] Entering netns %v.", epInfo.NetNsPath)
		err = ns.Enter()
		if err != nil {
			return nil, err
		}

		// Return to host network namespace.
		defer func() {
			log.Printf("[net] Exiting netns %v.", epInfo.NetNsPath)
			err = ns.Exit()
			if err != nil {
				log.Printf("[net] Failed to exit netns, err:%v.", err)
			}
		}()
	}

	// If a name for the container interface is specified...
	if epInfo.IfName != "" {
		// Interface needs to be down before renaming.
		log.Printf("[net] Setting link %v state down.", contIfName)
		err = netlink.SetLinkState(contIfName, false)
		if err != nil {
			return nil, err
		}

		// Rename the container interface.
		log.Printf("[net] Setting link %v name %v.", contIfName, epInfo.IfName)
		err = netlink.SetLinkName(contIfName, epInfo.IfName)
		if err != nil {
			return nil, err
		}
		contIfName = epInfo.IfName

		// Bring the interface back up.
		log.Printf("[net] Setting link %v state up.", contIfName)
		err = netlink.SetLinkState(contIfName, true)
		if err != nil {
			return nil, err
		}
	}

	// Assign IP address to container network interface.
	for _, ipAddr := range epInfo.IPAddresses {
		log.Printf("[net] Adding IP address %v to link %v.", ipAddr.String(), contIfName)
		err = netlink.AddIpAddress(contIfName, ipAddr.IP, &ipAddr)
		if err != nil {
			return nil, err
		}
	}

	// Add IP routes to container network interface.
	for _, route := range epInfo.Routes {
		log.Printf("[net] Adding IP route %+v to link %v.", route, contIfName)

		nlRoute := &netlink.Route{
			Family:    netlink.GetIpAddressFamily(route.Gw),
			Dst:       &route.Dst,
			Gw:        route.Gw,
			LinkIndex: containerIf.Index,
		}

		err = netlink.AddIpRoute(nlRoute)
		if err != nil {
			return nil, err
		}
	}

	// Create the endpoint object.
	ep = &endpoint{
		Id:          epInfo.Id,
		IfName:      contIfName,
		HostIfName:  hostIfName,
		MacAddress:  containerIf.HardwareAddr,
		IPAddresses: epInfo.IPAddresses,
		Gateways:    []net.IP{nw.extIf.IPv4Gateway},
	}

	return ep, nil
}

// deleteEndpointImpl deletes an existing endpoint from the network.
func (nw *network) deleteEndpointImpl(ep *endpoint) error {
	// Delete the veth pair by deleting one of the peer interfaces.
	// Deleting the host interface is more convenient since it does not require
	// entering the container netns and hence works both for CNI and CNM.
	log.Printf("[net] Deleting veth pair %v %v.", ep.HostIfName, ep.IfName)
	err := netlink.DeleteLink(ep.HostIfName)
	if err != nil {
		log.Printf("[net] Failed to delete veth pair %v: %v.", ep.HostIfName, err)
		return err
	}

	// Delete rules for IP addresses on the container interface.
	for _, ipAddr := range ep.IPAddresses {
		// Delete ARP reply rule.
		log.Printf("[net] Deleting ARP reply rule for IP address %v on %v.", ipAddr.String(), ep.Id)
		err = ebtables.SetArpReply(ipAddr.IP, nw.getArpReplyAddress(ep.MacAddress), ebtables.Delete)
		if err != nil {
			log.Printf("[net] Failed to delete ARP reply rule for IP address %v: %v.", ipAddr.String(), err)
		}

		// Delete MAC address translation rule.
		log.Printf("[net] Deleting MAC DNAT rule for IP address %v on %v.", ipAddr.String(), ep.Id)
		err = ebtables.SetDnatForIPAddress(nw.extIf.Name, ipAddr.IP, ep.MacAddress, ebtables.Delete)
		if err != nil {
			log.Printf("[net] Failed to delete MAC DNAT rule for IP address %v: %v.", ipAddr.String(), err)
		}
	}

	return nil
}

// getArpReplyAddress returns the MAC address to use in ARP replies.
func (nw *network) getArpReplyAddress(epMacAddress net.HardwareAddr) net.HardwareAddr {
	var macAddress net.HardwareAddr

	if nw.Mode == opModeTunnel {
		// In tunnel mode, resolve all IP addresses to the virtual MAC address for hairpinning.
		macAddress, _ = net.ParseMAC(virtualMacAddress)
	} else {
		// Otherwise, resolve to actual MAC address.
		macAddress = epMacAddress
	}

	return macAddress
}

// getInfoImpl returns information about the endpoint.
func (ep *endpoint) getInfoImpl(epInfo *EndpointInfo) {
}
