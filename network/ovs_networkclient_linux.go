package network

import (
	"net"
	"strings"

	"github.com/Azure/azure-container-networking/common"
	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/netlink"
	"github.com/Azure/azure-container-networking/ovsctl"
)

type OVSNetworkClient struct {
	bridgeName        string
	hostInterfaceName string
	enableSnatOnHost  bool
}

const (
	azureInternetVeth0 = "azintveth0"
	azureInternetVeth1 = "azintveth1"
	internetBridgeName = "azintbr"
)

func NewOVSClient(bridgeName string, hostInterfaceName string, enableSnatOnHost bool) *OVSNetworkClient {
	ovsClient := &OVSNetworkClient{
		bridgeName:        bridgeName,
		hostInterfaceName: hostInterfaceName,
		enableSnatOnHost:  enableSnatOnHost,
	}

	return ovsClient
}

func (client *OVSNetworkClient) CreateBridge() error {
	if err := ovsctl.CreateOVSBridge(client.bridgeName); err != nil {
		return err
	}

	if err := CreateInternetBridge(); err != nil {
		log.Printf("[net] Creating internet bridge failed with erro %v", err)
		//return err
	}

	link := netlink.VEthLink{
		LinkInfo: netlink.LinkInfo{
			Type: netlink.LINK_TYPE_VETH,
			Name: azureInternetVeth0,
		},
		PeerName: azureInternetVeth1,
	}

	err := netlink.AddLink(&link)
	if err != nil {
		log.Printf("[net] Failed to create veth pair, err:%v.", err)
		return err
	}

	if client.enableSnatOnHost {
		ip, addr, _ := net.ParseCIDR("169.254.0.1/16")
		err = netlink.AddIpAddress(internetBridgeName, ip, addr)
		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "file exists") {
			log.Printf("[net] Failed to add IP address %v: %v.", addr, err)
			return err
		}

		if err := netlink.SetLinkState(internetBridgeName, true); err != nil {
			return err
		}

		if err := netlink.SetLinkState(azureInternetVeth0, true); err != nil {
			return err
		}

		if err := netlink.SetLinkMaster(azureInternetVeth0, internetBridgeName); err != nil {
			return err
		}

		if err := netlink.SetLinkState(azureInternetVeth1, true); err != nil {
			return err
		}

		if err := ovsctl.AddPortOnOVSBridge(azureInternetVeth1, client.bridgeName, 0); err != nil {
			return err
		}

		/*var routes []RouteInfo
		route = RouteInfo{Dst:net.ParseCIDR("169.254.169.254/32")}
		routes = append(routes, route)
		if err := addRoutes(client.bridgeName, routes); err != nil {
			log.Printf("addroutes failed with error %v", err)
			return err
		}*/

		cmd := "iptables -t nat -A POSTROUTING -j MASQUERADE"
		log.Printf("Adding iptable snat rule %v", cmd)
		_, err := common.ExecuteShellCommand(cmd)
		if err != nil {
			return err
		}

		cmd = "ebtables -t nat -A PREROUTING -p 802_1Q -j DROP"
		log.Printf("Adding ebtable rule to drop vlan traffic on internet bridge %v", cmd)
		_, err = common.ExecuteShellCommand(cmd)
		return err

	}

	return nil
}

func (client *OVSNetworkClient) DeleteBridge() error {
	if err := ovsctl.DeleteOVSBridge(client.bridgeName); err != nil {
		log.Printf("Deleting ovs bridge failed with error %v", err)
		return err
	}

	if client.enableSnatOnHost {
		if err := DeleteInternetBridge(); err != nil {
			log.Printf("Deleting internet bridge failed with error %v", err)
			return err
		}

		return netlink.DeleteLink(azureInternetVeth0)
	}

	return nil
}

func CreateInternetBridge() error {
	log.Printf("[net] Creating Internet bridge %v.", internetBridgeName)

	link := netlink.BridgeLink{
		LinkInfo: netlink.LinkInfo{
			Type: netlink.LINK_TYPE_BRIDGE,
			Name: internetBridgeName,
		},
	}

	return netlink.AddLink(&link)
}

func DeleteInternetBridge() error {
	// Delete the bridge.
	err := netlink.DeleteLink(internetBridgeName)
	if err != nil {
		log.Printf("[net] Failed to delete bridge %v, err:%v.", internetBridgeName, err)
	}

	return nil
}

func (client *OVSNetworkClient) AddBridgeRules(extIf *externalInterface) error {
	//primary := extIf.IPAddresses[0].IP.String()
	mac := extIf.MacAddress.String()
	macHex := strings.Replace(mac, ":", "", -1)

	/*if err := ovsctl.AddVMIpAcceptRule(client.bridgeName, primary, mac); err != nil {
		return err
	}*/

	ofport, err := ovsctl.GetOVSPortNumber(client.hostInterfaceName)
	if err != nil {
		return err
	}

	// Arp SNAT Rule
	log.Printf("[ovs] Adding ARP SNAT rule for egress traffic on interface %v", client.hostInterfaceName)
	if err := ovsctl.AddArpSnatRule(client.bridgeName, mac, macHex, ofport); err != nil {
		return err
	}

	log.Printf("[ovs] Adding DNAT rule for ingress ARP traffic on interface %v.", client.hostInterfaceName)
	if err := ovsctl.AddArpDnatRule(client.bridgeName, ofport, macHex); err != nil {
		return err
	}

	return nil
}

func (client *OVSNetworkClient) DeleteBridgeRules(extIf *externalInterface) {
	ovsctl.DeletePortFromOVS(client.bridgeName, client.hostInterfaceName)
}

func (client *OVSNetworkClient) SetBridgeMasterToHostInterface() error {
	return ovsctl.AddPortOnOVSBridge(client.hostInterfaceName, client.bridgeName, 0)
}

func (client *OVSNetworkClient) SetHairpinOnHostInterface(enable bool) error {
	return nil
}
