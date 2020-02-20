// Copyright 2017 Microsoft. All rights reserved.
// MIT License

package cniv6

import (
	"github.com/Azure/azure-container-networking/cni"
	"github.com/Azure/azure-container-networking/common"
	"github.com/Azure/azure-container-networking/ipam"
	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/platform"
	"github.com/Azure/azure-container-networking/telemetry"

	cniSkel "github.com/containernetworking/cni/pkg/skel"
	cniTypesCurr "github.com/containernetworking/cni/pkg/types/current"
)

// cniV6Plugin represents the CNI IPV6 IPAM plugin.
type cniV6Plugin struct {
	*cni.Plugin
	am     ipam.AddressManager
	report *telemetry.CNIReport
	tb     *telemetry.TelemetryBuffer
}

// NewPlugin creates a new ipamPlugin object.
func NewPlugin(name string, config *common.PluginConfig) (*cniV6Plugin, error) {
	// Setup base plugin.
	plugin, err := cni.NewPlugin(name, config.Version)
	if err != nil {
		return nil, err
	}

	// Setup ipv6 address manager.
	am, err := ipam.NewAddressManager()
	if err != nil {
		return nil, err
	}

	// Create IPAM plugin.
	v6Plg := &cniV6Plugin{
		Plugin: plugin,
		am:     am,
	}

	return v6Plg, nil
}

// Starts the plugin.
func (plugin *cniV6Plugin) Start(config *common.PluginConfig) error {
	// Initialize base plugin.
	err := plugin.Initialize(config)
	if err != nil {
		log.Printf("[cni-v6] Failed to initialize base plugin, err:%v.", err)
		return err
	}

	// Log platform information.
	log.Printf("[cni-v6] Plugin %v version %v.", plugin.Name, plugin.Version)
	log.Printf("[cni-v6] Running on %v", platform.GetOSInfo())

	//TODO: Set plugin.Options.Environment to IPV6NatSource

	// Initialize address manager.
	err = plugin.am.Initialize(config, plugin.Options)
	if err != nil {
		log.Printf("[cni-v6] Failed to initialize address manager, err:%v.", err)
		return err
	}

	log.Printf("[cni-v6] Plugin started.")

	return nil
}

// Stops the plugin.
func (plugin *cniV6Plugin) Stop() {
	plugin.am.Uninitialize()
	plugin.Uninitialize()
	log.Printf("[cni-v6] Plugin stopped.")
}

func (plugin *cniV6Plugin) SetCNIReport(report *telemetry.CNIReport, tb *telemetry.TelemetryBuffer) {
	plugin.report = report
	plugin.tb = tb
}

//
// CNI implementation
// https://github.com/containernetworking/cni/blob/master/SPEC.md
//

// Add handles CNI add commands.
func (plugin *cniV6Plugin) Add(args *cniSkel.CmdArgs) error {
	var (
		result *cniTypesCurr.Result
		err    error
	)

	log.Printf("[cni-v6] Processing ADD command with args {ContainerID:%v Netns:%v IfName:%v Args:%v Path:%v StdinData:%s}.",
		args.ContainerID, args.Netns, args.IfName, args.Args, args.Path, args.StdinData)

	defer func() { log.Printf("[cni-v6] ADD command completed with result:%+v err:%v.", result, err) }()

	return nil
}

// Delete handles CNI delete commands.
func (plugin *cniV6Plugin) Delete(args *cniSkel.CmdArgs) error {
	var err error

	log.Printf("[cni-v6] Processing DEL command with args {ContainerID:%v Netns:%v IfName:%v Args:%v Path:%v StdinData:%s}.",
		args.ContainerID, args.Netns, args.IfName, args.Args, args.Path, args.StdinData)

	defer func() { log.Printf("[cni-v6] DEL command completed with err:%v.", err) }()

	return nil
}

// Get handles CNIV6 Get commands.
func (plugin *cniV6Plugin) Get(args *cniSkel.CmdArgs) error {
	return nil
}

// Update handles CNIV6 update command.
func (plugin *cniV6Plugin) Update(args *cniSkel.CmdArgs) error {
	return nil
}
