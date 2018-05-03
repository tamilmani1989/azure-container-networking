package network

import (
	"strings"

	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/ovsrules"
)

type OVSNetworkClient struct {
	bridgeName        string
	hostInterfaceName string
}

func NewOVSClient(bridgeName string, hostInterfaceName string) *OVSNetworkClient {
	ovsClient := &OVSNetworkClient{
		bridgeName:        bridgeName,
		hostInterfaceName: hostInterfaceName,
	}

	return ovsClient
}

func (client *OVSNetworkClient) CreateBridge() error {
	return ovsrules.CreateOVSBridge(client.bridgeName)
}

func (client *OVSNetworkClient) DeleteBridge() error {
	return ovsrules.DeleteOVSBridge(client.bridgeName)
}

func (client *OVSNetworkClient) AddBridgeRules(extIf *externalInterface) error {
	primary := extIf.IPAddresses[0].IP.String()
	mac := extIf.MacAddress.String()
	macHex := strings.Replace(mac, ":", "", -1)

	if err := ovsrules.AddVMIpAcceptRule(client.bridgeName, primary, mac); err != nil {
		return err
	}

	ofport, err := ovsrules.GetOVSPortNumber(client.hostInterfaceName)
	if err != nil {
		return err
	}

	/*	log.Printf("[ovs] Adding ARP SNAT rule for egress traffic on interface %v", client.hostInterfaceName)
		// Arp SNAT Rule
		if err := ovsrules.AddArpSnatRule(client.bridgeName, mac, macHex, ofport); err != nil {
			return err
		}
	*/
	log.Printf("[ovs] Adding DNAT rule for ingress ARP traffic on interface %v.", client.hostInterfaceName)
	if err := ovsrules.AddArpDnatRule(client.bridgeName, ofport, macHex); err != nil {
		return err
	}

	return nil
}

func (client *OVSNetworkClient) DeleteBridgeRules(extIf *externalInterface) {
	ovsrules.DeletePortFromOVS(client.bridgeName, client.hostInterfaceName)
}

func (client *OVSNetworkClient) SetBridgeMasterToHostInterface() error {
	return ovsrules.AddPortOnOVSBridge(client.hostInterfaceName, client.bridgeName, 0)
}

func (client *OVSNetworkClient) SetHairpinOnHostInterface(enable bool) error {
	return nil
}
