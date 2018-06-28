package network

import (
	"fmt"
	"os"
	"bytes"
	"net"
	"strings"

	"github.com/Azure/azure-container-networking/platform"
	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/netlink"
	"github.com/Azure/azure-container-networking/ovsctl"
)

type OVSNetworkClient struct {
	bridgeName        string
	hostInterfaceName string
	internetBridgeIP  string
	enableSnatOnHost  bool
}

const (
	azureInternetVeth0 = "azintveth0"
	azureInternetVeth1 = "azintveth1"
	internetBridgeName = "azintbr"
	imdsIP 			   = "169.254.169.254/32"
	ovsConfigFile      = "/etc/default/openvswitch-switch"
	ovsOpt             = "OVS_CTL_OPTS='--delete-bridges'"
)

func updateOVSConfig(option string) error {
	f, err := os.OpenFile(ovsConfigFile, os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		log.Printf("Error while opening ovs config %v", err)
		return err
	}

	defer f.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(f)
	contents := buf.String()

	conSplit := strings.Split(contents, "\n")

	for _, existingOption := range conSplit {
		if option == existingOption {
			log.Printf("Not updating ovs config. Found option already written")
			return nil
		}
	}

	log.Printf("writing ovsconfig option %v", option)

	if _, err = f.WriteString(option); err != nil {
		log.Printf("Error while writing ovs config %v", err)
		return err
	}

	return nil
}


func NewOVSClient(bridgeName, hostInterfaceName, internetBridgeIP string, enableSnatOnHost bool) *OVSNetworkClient {
	ovsClient := &OVSNetworkClient{
		bridgeName:        bridgeName,
		hostInterfaceName: hostInterfaceName,
		internetBridgeIP:  internetBridgeIP,
		enableSnatOnHost:  enableSnatOnHost,
	}

	return ovsClient
}

func (client *OVSNetworkClient) CreateBridge() error {
	if err := ovsctl.CreateOVSBridge(client.bridgeName); err != nil {
		return err
	}

	if err := updateOVSConfig(ovsOpt); err != nil {
		return err
	}

	if client.enableSnatOnHost {
		if err := CreateInternetBridge(); err != nil {
			log.Printf("[net] Creating internet bridge failed with erro %v", err)
			return err
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
	
		ip, addr, _ := net.ParseCIDR(client.internetBridgeIP)

		log.Printf("Assigning %v on internet bridge", client.internetBridgeIP)
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

		
		cmd := fmt.Sprintf("iptables -t nat -A POSTROUTING -s %v -j MASQUERADE", addr.String()) 
		log.Printf("Adding iptable snat rule %v", cmd)
		_, err = platform.ExecuteCommand(cmd)
		if err != nil {
			return err
		}

		cmd = "ebtables -t nat -A PREROUTING -p 802_1Q -j DROP"
		log.Printf("Adding ebtable rule to drop vlan traffic on internet bridge %v", cmd)
		_, err = platform.ExecuteCommand(cmd)
		return err

	}

	return nil
}

func deleteMasQueradeRule(interfaceName string) error {
	internetIf, _ := net.InterfaceByName(interfaceName)

	addrs, _ := internetIf.Addrs()
	for _, addr := range addrs {
		ipAddr, ipNet, err := net.ParseCIDR(addr.String())
		if err != nil {
			log.Printf("error %v", err)
			continue
		}
	
		if ipAddr.To4() != nil {
			cmd := fmt.Sprintf("iptables -t nat -D POSTROUTING -s %v -j MASQUERADE", ipNet.String()) 
			log.Printf("Deleting iptable snat rule %v", cmd)
			_, err = platform.ExecuteCommand(cmd)
			if err != nil {
				return err
			}	

			return nil
		}
	}
	
	return nil 
}

func (client *OVSNetworkClient) DeleteBridge() error {
	if err := ovsctl.DeleteOVSBridge(client.bridgeName); err != nil {
		log.Printf("Deleting ovs bridge failed with error %v", err)
		return err
	}

	if client.enableSnatOnHost {

		deleteMasQueradeRule(internetBridgeName)

		cmd := "ebtables -t nat -D PREROUTING -p 802_1Q -j DROP"
		_, err := platform.ExecuteCommand(cmd)
		if err != nil {
			log.Printf("Deleting ebtable vlan drop rule failed with error %v", err)	
		}

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
	
	if client.enableSnatOnHost {
		log.Printf("[ovs] Adding 169.254.169.254 static route")
		var routes []RouteInfo
		_, ipNet, _ := net.ParseCIDR(imdsIP)
		gwIP := net.ParseIP("0.0.0.0")
		route := RouteInfo{Dst: *ipNet, Gw: gwIP}
		routes = append(routes, route)
		if err := addRoutes(client.bridgeName, routes); err != nil {
			if err != nil && !strings.Contains(strings.ToLower(err.Error()), "file exists") {
				log.Printf("addroutes failed with error %v", err)
				return err
			}
		}
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
