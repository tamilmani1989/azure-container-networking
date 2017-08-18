// Copyright 2017 Microsoft. All rights reserved.
// MIT License

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"azure-overlay/common"
	"azure-overlay/log"
	nw "azure-overlay/network"
	"azure-overlay/overlay/network"
	"azure-overlay/platform"
	"azure-overlay/store"
)

const (
	// Plugin name as used in socket, log and store names.
	name = "azure-overlay"
)

// Version is populated by make during build.
var version string

// Command line arguments for overlay plugin.
var args = common.ArgumentList{
	{
		Name:         common.OptEnvironment,
		Shorthand:    common.OptEnvironmentAlias,
		Description:  "Set the operating environment",
		Type:         "string",
		DefaultValue: common.OptEnvironmentAzure,
		ValueMap: map[string]interface{}{
			common.OptEnvironmentAzure: 0,
			common.OptEnvironmentMAS:   0,
		},
	},
	{
		Name:         common.OptLogLevel,
		Shorthand:    common.OptLogLevelAlias,
		Description:  "Set the logging level",
		Type:         "int",
		DefaultValue: common.OptLogLevelInfo,
		ValueMap: map[string]interface{}{
			common.OptLogLevelInfo:  log.LevelInfo,
			common.OptLogLevelDebug: log.LevelDebug,
		},
	},
	{
		Name:         common.OptLogTarget,
		Shorthand:    common.OptLogTargetAlias,
		Description:  "Set the logging target",
		Type:         "int",
		DefaultValue: common.OptLogTargetFile,
		ValueMap: map[string]interface{}{
			common.OptLogTargetSyslog: log.TargetSyslog,
			common.OptLogTargetStderr: log.TargetStderr,
			common.OptLogTargetFile:   log.TargetLogfile,
		},
	},
	{
		Name:         common.OptIpamQueryInterval,
		Shorthand:    common.OptIpamQueryIntervalAlias,
		Description:  "Set the IPAM plugin query interval",
		Type:         "int",
		DefaultValue: "",
	},
	{
		Name:         common.OptVersion,
		Shorthand:    common.OptVersionAlias,
		Description:  "Print version information",
		Type:         "bool",
		DefaultValue: false,
	},
	{
		Name:         common.OptHostIfName,
		Shorthand:    common.OptHostIfNameAlias,
		Description:  "Host primary interface name",
		Type:         "string",
		DefaultValue: "eth0",
	},
	{
		Name:         common.OptHostIP,
		Shorthand:    common.OptHostIPAlias,
		Description:  "Host primary interface IP",
		Type:         "string",
		DefaultValue: "",
	},
	{
		Name:         common.OptHostGateway,
		Shorthand:    common.OptHostGatewayAlias,
		Description:  "Host default gateway",
		Type:         "string",
		DefaultValue: "",
	},
}

// Prints description and version information.
func printVersion() {
	fmt.Printf("Azure overlay (libnetwork) plugin\n")
	fmt.Printf("Version %v\n", version)
}

// Main is the entry point for overlay plugin.
func main() {
	// Initialize and parse command line arguments.
	common.ParseArgs(&args, printVersion)

	//	environment := common.GetArg(common.OptEnvironment).(string)
	logLevel := common.GetArg(common.OptLogLevel).(int)
	logTarget := common.GetArg(common.OptLogTarget).(int)
	//	ipamQueryInterval, _ := common.GetArg(common.OptIpamQueryInterval).(int)
	vers := common.GetArg(common.OptVersion).(bool)

	if vers {
		printVersion()
		os.Exit(0)
	}
	hostIfName := common.GetArg(common.OptHostIfName).(string)
	hostIP := common.GetArg(common.OptHostIP).(string)
	gateway := common.GetArg(common.OptHostGateway).(string)

	fmt.Println("Getarg", hostIfName, hostIP, gateway)
	if hostIfName == "" || hostIP == "" || gateway == "" {
		fmt.Println("Host ifname/ip/gateway cannot be null")
		os.Exit(0)
	}
	// Initialize plugin common configuration.
	var config common.PluginConfig
	config.Version = version
	config.SockName = name

	// Create a channel to receive unhandled errors from the plugins.
	config.ErrChan = make(chan error, 1)

	// Create network plugin.
	netPlugin, err := network.NewPlugin(&config)
	if err != nil {
		fmt.Printf("Failed to create network plugin, err:%v.\n", err)
		return
	}

	// Create the key value store.
	config.Store, err = store.NewJsonFileStore(platform.RuntimePath + name + ".json")
	if err != nil {
		fmt.Printf("Failed to create store: %v\n", err)
		return
	}

	// Create logging provider.
	log.SetName(name)
	log.SetLevel(logLevel)
	err = log.SetTarget(logTarget)
	if err != nil {
		fmt.Printf("Failed to configure logging: %v\n", err)
		return
	}

	// Log platform information.
	log.Printf("Running on %v", platform.GetOSInfo())
	common.LogNetworkInterfaces()

	// Start plugins.
	if netPlugin != nil {
		err = netPlugin.Start(&config)
		if err != nil {
			fmt.Printf("Failed to start network plugin, err:%v.\n", err)
			return
		}
	}
	err = common.System("ovs-vsctl", "add-br", nw.AzureOvsCommonBridge)
	if err != nil {
		log.Printf("Error while creating common OVS bridge %v", err.Error())
		return
	}

	err = common.System("ifconfig", nw.AzureOvsCommonBridge, hostIP)
	if err != nil {
		log.Printf("Error while assigning IP %v to common OVS bridge %v", hostIP, err.Error())
		return
	}

	err = common.System("ifconfig", hostIfName, "0.0.0.0")
	if err != nil {
		log.Printf("Error while removing IP from primary NIC %v", err.Error())
		return
	}

	err = common.System("ovs-vsctl", "add-port", nw.AzureOvsCommonBridge, hostIfName)
	if err != nil {
		log.Printf("Error while adding primary NIC as port to common OVS bridge %v", err.Error())
		return
	}

	err = common.System("route", "add", "default", "gw", gateway, "dev", nw.AzureOvsCommonBridge)
	if err != nil {
		log.Printf("Error while setting default gw %v to common OVS bridge %v", gateway, err.Error())
		return
	}

	// Relay these incoming signals to OS signal channel.
	osSignalChannel := make(chan os.Signal, 1)
	signal.Notify(osSignalChannel, os.Interrupt, os.Kill, syscall.SIGTERM)

	// Wait until receiving a signal.
	select {
	case sig := <-osSignalChannel:
		log.Printf("Received OS signal <" + sig.String() + ">, shutting down.")
	case err := <-config.ErrChan:
		log.Printf("Received unhandled plugin error %v, shutting down.", err)
	}

	// Cleanup.
	if netPlugin != nil {
		netPlugin.Stop()
	}

	err = common.System("ovs-vsctl", "del-br", nw.AzureOvsCommonBridge)
	if err != nil {
		log.Printf("Error while removing common OVS bridge %v", err.Error())
		return
	}

	err = common.System("ifconfig", hostIfName, hostIP, "up")
	if err != nil {
		log.Printf("Error while assigning IP  %v back to primary NIC %v", hostIP, err.Error())
		return
	}

	err = common.System("route", "add", "default", "gw", gateway, "dev", hostIfName)
	if err != nil {
		log.Printf("Error while setting default gw %v back to common OVS bridge %v", gateway, err.Error())
		return
	}
}
