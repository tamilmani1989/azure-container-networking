package iptables

import (
	"fmt"

	"github.com/Azure/azure-container-networking/log"

	"github.com/Azure/azure-container-networking/platform"
)

// cni iptable chains
const (
	CNIInputChain  = "AZURECNIINPUT"
	CNIOutputChain = "AZURECNIOUTPUT"
)

// standard iptable chains
const (
	Input  = "INPUT"
	Output = "OUTPUT"
)

// Standard Table names
const (
	Filter = "filter"
)

// target
const (
	Accept = "ACCEPT"
	Drop   = "Drop"
)

// actions
const (
	Insert = "I"
	Append = "A"
	Delete = "D"
)

func ChainExists(tableName, chainName string) bool {
	cmd := fmt.Sprintf("iptables -t %s -L %s", tableName, chainName)
	if _, err := platform.ExecuteCommand(cmd); err != nil {
		return false
	}

	return true
}

func CreateCNIChain(tableName, chainName string) error {
	var err error

	if !ChainExists(tableName, chainName) {
		cmd := fmt.Sprintf("iptables -t %s -N %s", tableName, chainName)
		_, err = platform.ExecuteCommand(cmd)
	} else {
		log.Printf("%s Chain exists in table %s", chainName, tableName)
	}

	return err
}

func RuleExists(tableName, chainName, match, target string) bool {
	cmd := fmt.Sprintf("iptables -t %s -C %s %s -j %s", tableName, chainName, match, target)
	_, err := platform.ExecuteCommand(cmd)
	if err != nil {
		return false
	}

	return true
}

func InsertIptableRule(tableName, chainName, match, target string) error {
	if RuleExists(tableName, chainName, match, target) {
		log.Printf("Rule already exists")
		return nil
	}

	cmd := fmt.Sprintf("iptables -t %s -I %s 1 %s -j %s", tableName, chainName, match, target)
	_, err := platform.ExecuteCommand(cmd)
	return err
}

func AppendIptableRule(tableName, chainName, match, target string) error {
	if RuleExists(tableName, chainName, match, target) {
		log.Printf("Rule already exists")
		return nil
	}

	cmd := fmt.Sprintf("iptables -t %s -A %s %s -j %s", tableName, chainName, match, target)
	_, err := platform.ExecuteCommand(cmd)
	return err
}

func DeleteIptableRule(tableName, chainName, match, target string) error {
	cmd := fmt.Sprintf("iptables -t %s -D %s %s -j %s", tableName, chainName, match, target)
	_, err := platform.ExecuteCommand(cmd)
	return err
}
