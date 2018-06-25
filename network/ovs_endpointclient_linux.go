package network

import (
	"fmt"
	"net"

	"github.com/Azure/azure-container-networking/common"
	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/netlink"
	"github.com/Azure/azure-container-networking/ovsctl"
)

type OVSEndpointClient struct {
	bridgeName        string
	hostPrimaryIfName string
	hostVethName      string
	hostPrimaryMac    string
	containerVethName string
	containerMac      string
	intVethName       string
	vlanID            int
	enableSnatOnHost  bool
}

const (
	intVethInterfacePrefix = commonInterfacePrefix + "vint"
	azureInternetIfName    = "eth1"
)

func NewOVSEndpointClient(
	bridgeName string,
	hostPrimaryIfName string,
	hostVethName string,
	hostPrimaryMac string,
	containerVethName string,
	vlanid int,
	enableSnatOnHost bool) *OVSEndpointClient {

	client := &OVSEndpointClient{
		bridgeName:        bridgeName,
		hostPrimaryIfName: hostPrimaryIfName,
		hostVethName:      hostVethName,
		hostPrimaryMac:    hostPrimaryMac,
		containerVethName: containerVethName,
		vlanID:            vlanid,
		enableSnatOnHost:  enableSnatOnHost,
	}

	return client
}

func (client *OVSEndpointClient) AddEndpoints(epInfo *EndpointInfo) error {
	if err := createEndpoint(client.hostVethName, client.containerVethName); err != nil {
		return err
	}

	containerIf, err := net.InterfaceByName(client.containerVethName)
	if err != nil {
		return err
	}

	client.containerMac = containerIf.HardwareAddr.String()

	if client.enableSnatOnHost {
		hostIfName := fmt.Sprintf("%s%s", intVethInterfacePrefix, epInfo.Id[:7])
		contIfName := fmt.Sprintf("%s%s-2", intVethInterfacePrefix, epInfo.Id[:7])

		if err := createEndpoint(hostIfName, contIfName); err != nil {
			return err
		}

		if err := netlink.SetLinkMaster(hostIfName, internetBridgeName); err != nil {
			return err
		}

		internetIf, err := net.InterfaceByName(contIfName)
		if err != nil {
			return err
		}

		intVethMac := internetIf.HardwareAddr.String()
		arpCmd := fmt.Sprintf("arp -s 169.254.0.2 %v", intVethMac)
		log.Printf("Adding arp entry for internet veth %v", arpCmd)
		if _, err := common.ExecuteShellCommand(arpCmd); err != nil {
			log.Printf("Setting static arp failed for internet interface %v", err)
		}

		client.intVethName = contIfName
	}

	return nil
}

func (client *OVSEndpointClient) AddEndpointRules(epInfo *EndpointInfo) error {
	log.Printf("[ovs] Setting link %v master %v.", client.hostVethName, client.bridgeName)
	if err := ovsctl.AddPortOnOVSBridge(client.hostVethName, client.bridgeName, client.vlanID); err != nil {
		return err
	}

	log.Printf("[ovs] Get ovs port for interface %v.", client.hostVethName)
	containerPort, err := ovsctl.GetOVSPortNumber(client.hostVethName)
	if err != nil {
		log.Printf("[ovs] Get ofport failed with error %v", err)
		return err
	}

	log.Printf("[ovs] Get ovs port for interface %v.", client.hostPrimaryIfName)
	hostPort, err := ovsctl.GetOVSPortNumber(client.hostPrimaryIfName)
	if err != nil {
		log.Printf("[ovs] Get ofport failed with error %v", err)
		return err
	}

	// IP SNAT Rule
	log.Printf("[ovs] Adding IP SNAT rule for egress traffic on %v.", containerPort)
	if err := ovsctl.AddIpSnatRule(client.bridgeName, containerPort, client.hostPrimaryMac); err != nil {
		return err
	}

	for _, ipAddr := range epInfo.IPAddresses {

		// Add Arp Reply Rules
		// Set Vlan id on arp request packet and forward it to table 1
		if err := ovsctl.AddFakeArpReply(client.bridgeName, ipAddr.IP); err != nil {
			return err
		}

		// Add IP DNAT rule based on dst ip and vlanid
		log.Printf("[ovs] Adding MAC DNAT rule for IP address %v on %v.", ipAddr.IP.String(), hostPort)
		if err := ovsctl.AddMacDnatRule(client.bridgeName, hostPort, ipAddr.IP, client.containerMac, client.vlanID); err != nil {
			return err
		}
	}

	return nil
}

func (client *OVSEndpointClient) DeleteEndpointRules(ep *endpoint) {
	log.Printf("[ovs] Get ovs port for interface %v.", ep.HostIfName)
	containerPort, err := ovsctl.GetOVSPortNumber(client.hostVethName)
	if err != nil {
		log.Printf("[ovs] Get portnum failed with error %v", err)
	}

	log.Printf("[ovs] Get ovs port for interface %v.", client.hostPrimaryIfName)
	hostPort, err := ovsctl.GetOVSPortNumber(client.hostPrimaryIfName)
	if err != nil {
		log.Printf("[ovs] Get portnum failed with error %v", err)
	}

	// Delete IP SNAT
	log.Printf("[ovs] Deleting IP SNAT for port %v", containerPort)
	ovsctl.DeleteIPSnatRule(client.bridgeName, containerPort)

	// Delete Arp Reply Rules for container
	log.Printf("[ovs] Deleting ARP reply rule for ip %v vlanid %v for container port", ep.IPAddresses[0].IP.String(), ep.VlanID, containerPort)
	ovsctl.DeleteArpReplyRule(client.bridgeName, containerPort, ep.IPAddresses[0].IP, ep.VlanID)

	// Delete MAC address translation rule.
	log.Printf("[ovs] Deleting MAC DNAT rule for IP address %v and vlan %v.", ep.IPAddresses[0].IP.String(), ep.VlanID)
	ovsctl.DeleteMacDnatRule(client.bridgeName, hostPort, ep.IPAddresses[0].IP, ep.VlanID)

	// Delete port from ovs bridge
	log.Printf("[ovs] Deleting interface %v from bridge %v", client.hostVethName, client.bridgeName)
	ovsctl.DeletePortFromOVS(client.bridgeName, client.hostVethName)
}

func (client *OVSEndpointClient) MoveEndpointsToContainerNS(epInfo *EndpointInfo, nsID uintptr) error {
	// Move the container interface to container's network namespace.
	log.Printf("[ovs] Setting link %v netns %v.", client.containerVethName, epInfo.NetNsPath)
	if err := netlink.SetLinkNetNs(client.containerVethName, nsID); err != nil {
		return err
	}

	if client.enableSnatOnHost {
		log.Printf("[ovs] Setting link %v netns %v.", client.intVethName, epInfo.NetNsPath)
		if err := netlink.SetLinkNetNs(client.intVethName, nsID); err != nil {
			return err
		}
	}

	return nil
}

func (client *OVSEndpointClient) SetupContainerInterfaces(epInfo *EndpointInfo) error {

	if err := setupContainerInterface(client.containerVethName, epInfo.IfName); err != nil {
		return err
	}

	client.containerVethName = epInfo.IfName

	if client.enableSnatOnHost {
		if err := setupContainerInterface(client.intVethName, azureInternetIfName); err != nil {
			return err
		}
		client.intVethName = azureInternetIfName
	}

	return nil
}

func (client *OVSEndpointClient) ConfigureContainerInterfacesAndRoutes(epInfo *EndpointInfo) error {
	if err := assignIPToInterface(client.containerVethName, epInfo.IPAddresses); err != nil {
		return err
	}

	if client.enableSnatOnHost {
		log.Printf("[ovs] Adding IP address 169.254.0.2/16 to link %v.", client.intVethName)
		ip, intIpAddr, _ := net.ParseCIDR("169.254.0.2/16")
		if err := netlink.AddIpAddress(client.intVethName, ip, intIpAddr); err != nil {
			return err
		}
	}

	if err := addRoutes(client.containerVethName, epInfo.Routes); err != nil {
		return err
	}

	return nil
}

func (client *OVSEndpointClient) DeleteEndpoints(ep *endpoint) error {
	log.Printf("[ovs] Deleting veth pair %v %v.", ep.HostIfName, ep.IfName)
	err := netlink.DeleteLink(ep.HostIfName)
	if err != nil {
		log.Printf("[ovs] Failed to delete veth pair %v: %v.", ep.HostIfName, err)
		return err
	}

	if client.enableSnatOnHost {
		arpCmd := fmt.Sprintf("arp -d 169.254.0.2")
	
		log.Printf("Adding arp entry for internet veth %v", arpCmd)
		if _, err := common.ExecuteShellCommand(arpCmd); err != nil {
			log.Printf("Setting static arp failed for internet interface %v", err)
		}


		hostIfName := fmt.Sprintf("%s%s", intVethInterfacePrefix, ep.Id[:7])
		log.Printf("[ovs] Deleting internet veth pair %v.", hostIfName)
		err = netlink.DeleteLink(hostIfName)
		if err != nil {
			log.Printf("[ovs] Failed to delete veth pair %v: %v.", hostIfName, err)
			return err
		}
	}

	return nil
}
