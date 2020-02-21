// Copyright 2017 Microsoft. All rights reserved.
// MIT License

package cniv6

import (
	"os"

	"github.com/Azure/azure-container-networking/cni"
	"github.com/Azure/azure-container-networking/common"
	"github.com/Azure/azure-container-networking/ipam"
	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/network"
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
		nwCfg  *cni.NetworkConfig
		ns     *network.Namespace
		err    error
	)

	log.Printf("[cni-v6] Processing ADD command with args {ContainerID:%v Netns:%v IfName:%v Args:%v Path:%v StdinData:%s}.",
		args.ContainerID, args.Netns, args.IfName, args.Args, args.Path, args.StdinData)

	// Parse network configuration from stdin.
	nwCfg, err = cni.ParseNetworkConfig(args.StdinData)
	if err != nil {
		err = plugin.Errorf("Failed to parse network configuration: %v.", err)
		return err
	}

	log.Printf("[cni-net] Read network configuration %+v.", nwCfg)

	defer func() {
		if result == nil {
			result = &cniTypesCurr.Result{}
		}

		// _, ipv6net, _ := net.ParseCIDR("fc00::2/64")
		// ipv6net := net.IPNet{IP: ipaddr, Mask: net.CIDRMask(24, 32)}
		// ip := &cniTypesCurr.IPConfig{
		// 	Version: "6",
		// 	Address: ipv6net,
		// 	Gateway: net.ParseIP("fc00::1"),
		// }

		// result.IPs = append(result.IPs, ip)

		iface := &cniTypesCurr.Interface{
			Name: args.IfName,
		}

		result.Interfaces = append(result.Interfaces, iface)

		res, vererr := result.GetAsVersion(nwCfg.CNIVersion)
		if vererr != nil {
			log.Printf("GetAsVersion failed with error %v", vererr)
			plugin.Error(vererr)
		}

		if err == nil && res != nil {
			// Output the result to stdout.
			res.Print()
		}

		log.Printf("[cni-v6] ADD command completed with result:%+v err:%v.", result, err)
	}()

	log.Printf("[cni-v6] Entering container namespace")
	ns, err = network.OpenNamespace(args.Netns)
	if err != nil {
		return err
	}
	defer ns.Close()

	// Enter the container network namespace.
	log.Printf("[net] Entering netns %v.", args.Netns)
	if err = ns.Enter(); err != nil {
		return err
	}

	// Return to host network namespace.
	defer func() {
		log.Printf("[net] Exiting netns %v.", args.Netns)
		if err := ns.Exit(); err != nil {
			log.Printf("[net] Failed to exit netns, err:%v.", err)
		}
	}()

	platform.ExecuteCommand("sysctl -w net.ipv6.conf.all.disable_ipv6=0")
	platform.ExecuteCommand("ip -6 addr add fc00::2/64 dev eth0")

	return nil
}

// Delete handles CNI delete commands.
func (plugin *cniV6Plugin) Delete(args *cniSkel.CmdArgs) error {
	var (
		result *cniTypesCurr.Result
		nwCfg  *cni.NetworkConfig
		err    error
	)

	log.Printf("[cni-v6] Processing DEL command with args {ContainerID:%v Netns:%v IfName:%v Args:%v Path:%v StdinData:%s}.",
		args.ContainerID, args.Netns, args.IfName, args.Args, args.Path, args.StdinData)
	// Parse network configuration from stdin.
	nwCfg, err = cni.ParseNetworkConfig(args.StdinData)
	if err != nil {
		err = plugin.Errorf("Failed to parse network configuration: %v.", err)
		return err
	}

	log.Printf("[cni-net] Read network configuration %+v.", nwCfg)

	defer func() {
		if result == nil {
			result = &cniTypesCurr.Result{}
		}

		res, vererr := result.GetAsVersion(nwCfg.CNIVersion)
		if vererr != nil {
			log.Printf("GetAsVersion failed with error %v", vererr)
			plugin.Error(vererr)
		}

		if err == nil && res != nil {
			// Output the result to stdout.
			res.Print()
		}

		log.Printf("[cni-v6] DEL command completed with err:%v.", err)
	}()

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

// Namespace represents a network namespace.
type Namespace struct {
	file   *os.File
	prevNs *Namespace
}

// // OpenNamespace creates a new namespace object for the given netns path.
// func OpenNamespace(nsPath string) (*Namespace, error) {
// 	fd, err := os.Open(nsPath)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &Namespace{file: fd}, nil
// }

// func (ns *Namespace) Close() error {
// 	if ns.file == nil {
// 		return nil
// 	}

// 	err := ns.file.Close()
// 	if err != nil {
// 		return fmt.Errorf("Failed to close namespace %v, err:%v", ns.file.Name(), err)
// 	}

// 	ns.file = nil

// 	return nil
// }

// // Enter puts the caller thread inside the namespace.
// func (ns *Namespace) Enter() error {
// 	var err error

// 	ns.prevNs, err = GetCurrentThreadNamespace()
// 	if err != nil {
// 		return err
// 	}

// 	runtime.LockOSThread()

// 	err = ns.set()
// 	if err != nil {
// 		runtime.UnlockOSThread()
// 		return err
// 	}

// 	// Recycle the netlink socket for the new network namespace.
// 	netlink.ResetSocket()

// 	return nil
// }

// // Exit puts the caller thread to its previous namespace.
// func (ns *Namespace) Exit() error {
// 	err := ns.prevNs.set()
// 	if err != nil {
// 		return err
// 	}

// 	ns.prevNs.Close()
// 	ns.prevNs = nil

// 	runtime.UnlockOSThread()

// 	// Recycle the netlink socket for the new network namespace.
// 	netlink.ResetSocket()

// 	return nil
// }
