package main

import (
	"LinuxServiceTunnel/configuration"
	"LinuxServiceTunnel/servicetunnel"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/platform"
)

const (
	versionID    = "VERSION_ID"
	distroID     = "ID"
	ubuntuStr    = "ubuntu"
	dotDelimiter = "."
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

	log.Printf("[net] OSInfo: %+v", osInfo)

	version := osInfo[versionID]
	distro := osInfo[distroID]

	if strings.EqualFold(distro, ubuntuStr) {
		version = strings.Trim(version, "\"")
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

func CheckIfSTBridgeExists(name string) bool {
	_, err := net.InterfaceByName(name)
	return err == nil
}

func CreateServiceTunnel(stConfig configuration.ServiceTunnelConfig) error {
	installTools()

	st := &servicetunnel.ServiceTunnel{PrimaryInterfaceName: stConfig.PrimaryInterfaceName, TargetInterfaceName: stConfig.BridgeName}

	if err := st.GetPrimaryIP(); err != nil {
		log.Printf("GetPrimaryIP failed with error %v", err)
		os.Exit(1)
	}

	if err := st.GetIPv6Address(); err != nil {
		log.Printf("GetIPv6Address failed with error %v", err)
		os.Exit(1)
	}

	log.Printf("Getting IPv6 succeeded %v", st.IPv6Address)

	if isGreaterOrEqaulUbuntuVersion(17) {
		if err := st.GetDNSConfig(); err != nil {
			log.Printf("GetDNSConfig failed with error %v", err)
		} else {
			log.Printf("GetDNSConfig succeeded %v:%v", st.DNSServerAddresses, st.DNSSearchDomain)
		}
	}

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

	if isGreaterOrEqaulUbuntuVersion(17) {
		if err := st.ApplyDNSConfig(); err != nil {
			log.Printf("ApplyDNSConfig failed with error %v", err)
		} else {
			log.Printf("Apply dns config succeeded")
		}
	}

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

	if err := st.CreateVethPair(stConfig.Veth1Name, stConfig.Veth2Name); err != nil {
		log.Printf("CreateVethPair failed with error %v", err)
		return err
	}

	log.Printf("Creating Veth Pair succeeded ")

	if err := st.SetBridgeMasterInterface(stConfig.Veth1Name); err != nil {
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

	if len(stConfig.PortsToOpen) > 0 {
		if err := st.ConfgureFirewallRules(servicetunnel.Add, stConfig.PortsToOpen); err != nil {
			log.Printf("ConfgureFirewallRules failed with error %v", err)
			return err
		}

		log.Printf("Configuring firewall rules succeeded")
	}

	return nil
}

func RemoveServiceTunnel(stConfig configuration.ServiceTunnelConfig) {
	st := &servicetunnel.ServiceTunnel{
		PrimaryInterfaceName: stConfig.BridgeName,
		TargetInterfaceName:  stConfig.PrimaryInterfaceName,
		VethName1:            stConfig.Veth1Name,
		VethName2:            stConfig.Veth2Name,
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

	if len(stConfig.PortsToOpen) > 0 {
		if err := st.ConfgureFirewallRules(servicetunnel.Del, stConfig.PortsToOpen); err != nil {
			log.Printf("ConfgureFirewallRules failed with error %v", err)
		}
	}
}
