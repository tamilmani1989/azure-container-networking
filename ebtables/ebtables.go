// Copyright 2017 Microsoft. All rights reserved.
// MIT License

package ebtables

import (
	"fmt"
	"io/ioutil"
	"net"
	"os/exec"
	"strings"

	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/platform"
)

const (
	// Ebtables actions.
	Append = "-A"
	Delete = "-D"
)

const (
	azurePreRoutingChain  = "AZUREPREROUTING"
	azurePostRoutingChain = "AZUREPOSTROUTING"
)

// InstallEbtables installs the ebtables package.
func installEbtables() {
	version, _ := ioutil.ReadFile("/proc/version")
	os := strings.ToLower(string(version))

	if strings.Contains(os, "ubuntu") {
		executeShellCommand("apt-get install ebtables")
	} else if strings.Contains(os, "redhat") {
		executeShellCommand("yum install ebtables")
	} else {
		log.Printf("Unable to detect OS platform. Please make sure the ebtables package is installed.")
	}
}

func createChainIfNotExist(targetChainName string) error {
	command := fmt.Sprintf("ebtables -t nat -L %v", targetChainName)
	if err := executeShellCommand(command); err == nil {
		log.Printf("%v chain exists", targetChainName)
		return nil
	}

	command = fmt.Sprintf("ebtables -t nat -N %v -P RETURN", targetChainName)
	return executeShellCommand(command)
}

func deleteChain(targetChainName string) error {
	command := fmt.Sprintf("ebtables -t nat -X %v", targetChainName)
	return executeShellCommand(command)
}

func insertRuleToForwardToAzureChain(targetChainName string, existingChainName string) error {
	command := fmt.Sprintf("ebtables -t nat -L %v", existingChainName)
	out, err := platform.ExecuteCommand(command)
	if err != nil {
		log.Printf("Listing %v chain rules failed with error %v", existingChainName, err)
		return err
	}

	splitStr := strings.Split(out, "\n")
	targetStr := fmt.Sprintf("-j %v", targetChainName)

	for _, item := range splitStr {
		if item == targetStr {
			log.Printf("rule to forward from %v to %v chain already exists", existingChainName, targetChainName)
			return nil
		}
	}

	command = fmt.Sprintf("ebtables -t nat -I %v 1 -j %v", existingChainName, targetChainName)
	return executeShellCommand(command)

}

func deleteRuleToForwardToAzureChain(targetChainName string, existingChainName string) error {
	command := fmt.Sprintf("ebtables -t nat -D %v -j %v", existingChainName, targetChainName)
	return executeShellCommand(command)
}

// Initialize creates new chain for creating l2rules for azure container networking
func Initialize() error {
	if err := createChainIfNotExist(azurePreRoutingChain); err != nil {
		log.Printf("creating %v chain failed with error %v", azurePreRoutingChain, err)
		return err
	}

	if err := createChainIfNotExist(azurePostRoutingChain); err != nil {
		log.Printf("creating %v chain failed with error %v", azurePostRoutingChain, err)
		return err
	}

	if err := insertRuleToForwardToAzureChain(azurePreRoutingChain, "PREROUTING"); err != nil {
		log.Printf("insertRuleToForwardToAzureChain for %v chain failed with error %v", azurePreRoutingChain, err)
		return err
	}

	if err := insertRuleToForwardToAzureChain(azurePostRoutingChain, "POSTROUTING"); err != nil {
		log.Printf("insertRuleToForwardToAzureChain for %v chain failed with error %v", azurePostRoutingChain, err)
		return err
	}

	return nil
}

// UnInitialize deletes the l2 chain and rules created for azure container networking
func UnInitialize() error {
	if err := deleteRuleToForwardToAzureChain(azurePreRoutingChain, "PREROUTING"); err != nil {
		log.Printf("deleteRuleToForwardToAzureChain for %v chain failed with error %v", azurePreRoutingChain, err)
		return err
	}

	if err := deleteRuleToForwardToAzureChain(azurePostRoutingChain, "POSTROUTING"); err != nil {
		log.Printf("deleteRuleToForwardToAzureChain for %v chain failed with error %v", azurePreRoutingChain, err)
		return err
	}

	if err := deleteChain(azurePreRoutingChain); err != nil {
		log.Printf("deleting %v chain failed with error %v", azurePreRoutingChain, err)
		return err
	}

	if err := deleteChain(azurePostRoutingChain); err != nil {
		log.Printf("deleting %v chain failed with error %v", azurePostRoutingChain, err)
		return err
	}

	return nil
}

// SetSnatForInterface sets a MAC SNAT rule for an interface.
func SetSnatForInterface(interfaceName string, macAddress net.HardwareAddr, action string) error {
	command := fmt.Sprintf(
		"ebtables -t nat %s %s -s unicast -o %s -j snat --to-src %s --snat-arp --snat-target ACCEPT",
		action, azurePostRoutingChain, interfaceName, macAddress.String())

	return executeShellCommand(command)
}

// SetArpReply sets an ARP reply rule for the given target IP address and MAC address.
func SetArpReply(ipAddress net.IP, macAddress net.HardwareAddr, action string) error {
	command := fmt.Sprintf(
		"ebtables -t nat %s %s -p ARP --arp-op Request --arp-ip-dst %s -j arpreply --arpreply-mac %s --arpreply-target DROP",
		action, azurePreRoutingChain, ipAddress, macAddress.String())

	return executeShellCommand(command)
}

// SetDnatForArpReplies sets a MAC DNAT rule for ARP replies received on an interface.
func SetDnatForArpReplies(interfaceName string, action string) error {
	command := fmt.Sprintf(
		"ebtables -t nat %s %s -p ARP -i %s --arp-op Reply -j dnat --to-dst ff:ff:ff:ff:ff:ff --dnat-target ACCEPT",
		action, azurePreRoutingChain, interfaceName)

	return executeShellCommand(command)
}

// SetVepaMode sets the VEPA mode for a bridge and its ports.
func SetVepaMode(bridgeName string, downstreamIfNamePrefix string, upstreamMacAddress string, action string) error {
	if !strings.HasPrefix(bridgeName, downstreamIfNamePrefix) {
		command := fmt.Sprintf(
			"ebtables -t nat %s %s -i %s -j dnat --to-dst %s --dnat-target ACCEPT",
			action, azurePreRoutingChain, bridgeName, upstreamMacAddress)

		err := executeShellCommand(command)
		if err != nil {
			return err
		}
	}

	command := fmt.Sprintf(
		"ebtables -t nat %s %s -i %s+ -j dnat --to-dst %s --dnat-target ACCEPT",
		action, azurePreRoutingChain, downstreamIfNamePrefix, upstreamMacAddress)

	return executeShellCommand(command)
}

// SetDnatForIPAddress sets a MAC DNAT rule for an IP address.
func SetDnatForIPAddress(interfaceName string, ipAddress net.IP, macAddress net.HardwareAddr, action string) error {
	command := fmt.Sprintf(
		"ebtables -t nat %s %s -p IPv4 -i %s --ip-dst %s -j dnat --to-dst %s --dnat-target ACCEPT",
		action, azurePreRoutingChain, interfaceName, ipAddress.String(), macAddress.String())

	return executeShellCommand(command)
}

func executeShellCommand(command string) error {
	log.Printf("[ebtables] %s", command)
	cmd := exec.Command("sh", "-c", command)
	err := cmd.Start()
	if err != nil {
		return err
	}
	return cmd.Wait()
}
