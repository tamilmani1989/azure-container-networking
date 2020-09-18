package main

import (
	extensionhandler "linuxnetagent/ExtensionHandler"
	"linuxnetagent/servicetunnel"
	"net"

	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/platform"
)

const (
	PrimaryInterfaceName = "eth0"
	BridgeName           = "azurest"
	VETH1                = "azstveth1"
	VETH2                = "azstveth2"
)

func installTools() {
	_, err := platform.ExecuteCommand("apt-get clean")
	if err != nil {
		log.Printf("apt-get failed with error %v", err)
	}

	_, err = platform.ExecuteCommand("apt -y update")
	if err != nil {
		log.Printf("apt update failed with error %v", err)
	}

	_, err = platform.ExecuteCommand("apt install -y ebtables")
	if err != nil {
		log.Printf("apt install ebtables failed with error %v", err)
	}
}

func CheckIfSTBridgeExists() bool {
	_, err := net.InterfaceByName(BridgeName)
	return err == nil
}

func CreateServiceTunnel(stConfig *extensionhandler.ServiceTunnelConfig) error {
	installTools()

	st := &servicetunnel.ServiceTunnel{PrimaryInterfaceName: PrimaryInterfaceName, TargetInterfaceName: BridgeName}

	if err := st.GetPrimaryIP(); err != nil {
		log.Printf("GetPrimaryIP failed with error %v", err)
		return err
	}

	if err := st.GetIPv6Address(); err != nil {
		log.Printf("GetIPv6Address failed with error %v", err)
		return err
	}

	log.Printf("Getting IPv6 succeeded %v", st.IPv6Address)

	if err := st.GetRoutesAndGateway(); err != nil {
		log.Printf("GetRoutesAndGateway failed with error %v", err)
		return err
	}

	log.Printf("Service tunnel %+v", st)

	if err := st.CreateSTBridge(); err != nil {
		log.Printf("CreateAcrBridge failed with error %v", err)
		return err
	}

	log.Printf("Creating Bridge succeeded %v", st.TargetInterfaceName)

	if err := st.ApplyIPAddresses(); err != nil {
		log.Printf("ApplyIPAddresses failed with error %v", err)
		return err
	}

	log.Printf("Apply IP on bridge succeeded %v", st.IPv4Addresses[0])

	if err := st.RemoveIP(); err != nil {
		log.Printf("RemoveIP failed with error %v", err)
		return err
	}

	if err := st.SetBridgeMasterInterface(st.PrimaryInterfaceName); err != nil {
		log.Printf("SetBridgeMasterInterface failed with error %v", err)
		return err
	}

	if err := st.ApplyRoutes(); err != nil {
		log.Printf("ApplyRoutes failed with error %v", err)
		return err
	}

	log.Printf("Apply Routes on bridge succeeded ")

	if err := st.CreateVethPair(VETH1, VETH2); err != nil {
		log.Printf("CreateVethPair failed with error %v", err)
		return err
	}

	log.Printf("Creating Veth Pair succeeded ")

	if err := st.SetBridgeMasterInterface(VETH1); err != nil {
		log.Printf("SetBridgeMasterInterface failed with error %v", err)
		return err
	}

	if err := st.ApplyIPv6ToVeth(); err != nil {
		log.Printf("ApplyIPv6ToVeth failed with error %v", err)
		return err
	}

	log.Printf("Adding IPv6 to Veth succeeded ")

	if err := st.GetVethMac(); err != nil {
		log.Printf("GetInterfaceMacAddress failed with error %v", err)
		return err
	}

	if err := st.ApplyMacDnatRule("A"); err != nil {
		log.Printf("AddMacDnatRule failed with error %v", err)
		return err
	}

	if err := st.SetFakeIPv6Route(); err != nil {
		log.Printf("SetFakeIPv6Route failed with error %v", err)
		return err
	}

	if err := st.SetFakeIPv6Neighbor(); err != nil {
		log.Printf("SetFakeIPv6Route failed with error %v", err)
		return err
	}

	log.Printf("Setting up FakeArp/FakeRoute/Ebtable rules succeeded")

	if stConfig != nil {
		if err := st.ConfgureFirewallRules(servicetunnel.Add, stConfig.PortsToOpen); err != nil {
			log.Printf("ConfgureFirewallRules failed with error %v", err)
			return err
		}

		log.Printf("Configuring firewall rules succeeded")
	}

	return nil
}

func RemoveServiceTunnel(stConfig *extensionhandler.ServiceTunnelConfig) {
	st := &servicetunnel.ServiceTunnel{
		PrimaryInterfaceName: BridgeName,
		TargetInterfaceName:  PrimaryInterfaceName,
		VethName1:            VETH1,
		VethName2:            VETH2,
	}

	if err := st.GetPrimaryIP(); err != nil {
		log.Printf("GetPrimaryIP failed with error %v", err)
	}

	if err := st.GetIPv6Address(); err != nil {
		log.Printf("GetIPv6Address failed with error %v", err)
	}

	if err := st.GetRoutesAndGateway(); err != nil {
		log.Printf("GetRoutesAndGateway failed with error %v", err)
	}

	if err := st.GetVethMac(); err != nil {
		log.Printf("GetInterfaceMacAddress failed with error %v", err)
	}

	if err := st.ApplyMacDnatRule("D"); err != nil {
		log.Printf("Removing mac dnat rule failed with error %v", err)
	}

	if err := st.DeleteVethPair(); err != nil {
		log.Printf("DeleteVethPair failed with error %v", err)
	}

	if err := st.DeleteSTBridge(true); err != nil {
		log.Printf("DeleteSTBridge failed with error %v", err)
	}

	if err := st.ApplyIPAddresses(); err != nil {
		log.Printf("ApplyIPAddresses failed with error %v", err)
	}

	if err := st.ApplyRoutes(); err != nil {
		log.Printf("ApplyRoutes failed with error %v", err)
	}

	if err := st.DeleteFakeIPv6Route(); err != nil {
		log.Printf("DeleteFakeIPv6Route failed with error %v", err)
	}

	if err := st.DeleteFakeIPv6Neighbor(); err != nil {
		log.Printf("DeleteFakeIPv6Neighbor failed with error %v", err)
	}

	if stConfig != nil {
		if err := st.ConfgureFirewallRules(servicetunnel.Del, stConfig.PortsToOpen); err != nil {
			log.Printf("ConfgureFirewallRules failed with error %v", err)
		}
	}
}
