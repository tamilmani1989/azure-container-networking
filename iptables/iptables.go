package iptables

// This package contains wrapper functions to program iptables rules

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
	Input       = "INPUT"
	Output      = "OUTPUT"
	Forward     = "FORWARD"
	Prerouting  = "PREROUTING"
	Postrouting = "POSTROUTING"
)

// Standard Table names
const (
	Filter = "filter"
	Nat    = "nat"
)

// target
const (
	Accept     = "ACCEPT"
	Drop       = "DROP"
	Masquerade = "MASQUERADE"
)

// actions
const (
	Insert = "I"
	Append = "A"
	Delete = "D"
)

// states
const (
	Established = "ESTABLISHED"
	Related     = "RELATED"
)

const (
	iptables    = "iptables"
	lockTimeout = 60
)

// Run iptables command
func runCmd(params string) error {
	cmd := fmt.Sprintf("%s -w %d %s", iptables, lockTimeout, params)
	if _, err := platform.ExecuteCommand(cmd); err != nil {
		return err
	}

	return nil
}

// check if iptable chain alreay exists
func ChainExists(tableName, chainName string) bool {
	params := fmt.Sprintf("-t %s -L %s", tableName, chainName)
	if err := runCmd(params); err != nil {
		return false
	}

	return true
}

// create new iptable chain under specified table name
func CreateChain(tableName, chainName string) error {
	var err error

	if !ChainExists(tableName, chainName) {
		params := fmt.Sprintf("-t %s -N %s", tableName, chainName)
		err = runCmd(params)
	} else {
		log.Printf("%s Chain exists in table %s", chainName, tableName)
	}

	return err
}

// check if iptable rule alreay exists
func RuleExists(tableName, chainName, match, target string) bool {
	params := fmt.Sprintf("-t %s -C %s %s -j %s", tableName, chainName, match, target)
	if err := runCmd(params); err != nil {
		return false
	}
	return true
}

// Insert iptable rule at beginning of iptable chain
func InsertIptableRule(tableName, chainName, match, target string) error {
	if RuleExists(tableName, chainName, match, target) {
		log.Printf("Rule already exists")
		return nil
	}

	params := fmt.Sprintf("-t %s -I %s 1 %s -j %s", tableName, chainName, match, target)
	return runCmd(params)
}

// Append iptable rule at end of iptable chain
func AppendIptableRule(tableName, chainName, match, target string) error {
	if RuleExists(tableName, chainName, match, target) {
		log.Printf("Rule already exists")
		return nil
	}

	params := fmt.Sprintf("-t %s -A %s %s -j %s", tableName, chainName, match, target)
	return runCmd(params)
}

// Delete matched iptable rule
func DeleteIptableRule(tableName, chainName, match, target string) error {
	params := fmt.Sprintf("-t %s -D %s %s -j %s", tableName, chainName, match, target)
	return runCmd(params)
}
