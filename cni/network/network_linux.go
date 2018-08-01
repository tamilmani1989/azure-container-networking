package network

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/Azure/azure-container-networking/cni"
	"github.com/Azure/azure-container-networking/cns"
	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/network"
	cniTypes "github.com/containernetworking/cni/pkg/types"
	cniTypesCurr "github.com/containernetworking/cni/pkg/types/current"
)

const (
	snatInterface = "eth1"
	binPath       = "/opt/cni/bin"
)

// handleConsecutiveAdd is a dummy function for Linux platform.
func handleConsecutiveAdd(containerId, endpointId string, nwInfo *network.NetworkInfo, nwCfg *cni.NetworkConfig) (*cniTypesCurr.Result, error) {
	return nil, nil
}

func addDefaultRoute(gwIPString string, epInfo *network.EndpointInfo, result *cniTypesCurr.Result) {
	_, defaultIPNet, _ := net.ParseCIDR("0.0.0.0/0")
	dstIP := net.IPNet{IP: net.ParseIP("0.0.0.0"), Mask: defaultIPNet.Mask}
	gwIP := net.ParseIP(gwIPString)
	epInfo.Routes = append(epInfo.Routes, network.RouteInfo{Dst: dstIP, Gw: gwIP, DevName: snatInterface})
	result.Routes = append(result.Routes, &cniTypes.Route{Dst: dstIP, GW: gwIP})
}

func setNetworkOptions(cnsNwConfig *cns.GetNetworkContainerResponse, nwInfo *network.NetworkInfo) {
	if cnsNwConfig != nil && cnsNwConfig.MultiTenancyInfo.ID != 0 {
		log.Printf("Setting Network Options")
		vlanMap := make(map[string]interface{})
		vlanMap[network.VlanIDKey] = strconv.Itoa(cnsNwConfig.MultiTenancyInfo.ID)
		vlanMap[network.SnatBridgeIPKey] = cnsNwConfig.LocalIPConfiguration.GatewayIPAddress + "/" + strconv.Itoa(int(cnsNwConfig.LocalIPConfiguration.IPSubnet.PrefixLength))
		nwInfo.Options[dockerNetworkOption] = vlanMap
	}
}

func setEndpointOptions(cnsNwConfig *cns.GetNetworkContainerResponse, epInfo *network.EndpointInfo, vethName string) {
	if cnsNwConfig != nil && cnsNwConfig.MultiTenancyInfo.ID != 0 {
		log.Printf("Setting Endpoint Options")
		epInfo.Data[network.VlanIDKey] = cnsNwConfig.MultiTenancyInfo.ID
		epInfo.Data[network.LocalIPKey] = cnsNwConfig.LocalIPConfiguration.IPSubnet.IPAddress + "/" + strconv.Itoa(int(cnsNwConfig.LocalIPConfiguration.IPSubnet.PrefixLength))
		epInfo.Data[network.SnatBridgeIPKey] = cnsNwConfig.LocalIPConfiguration.GatewayIPAddress + "/" + strconv.Itoa(int(cnsNwConfig.LocalIPConfiguration.IPSubnet.PrefixLength))
	}

	epInfo.Data[network.OptVethName] = vethName
}

func addSnatInterface(nwCfg *cni.NetworkConfig, result *cniTypesCurr.Result) {
	if nwCfg != nil && nwCfg.MultiTenancy {
		snatIface := &cniTypesCurr.Interface{
			Name: snatInterface,
		}

		result.Interfaces = append(result.Interfaces, snatIface)
	}
}

func startMonitorIfNotRunning(nwCfg *cni.NetworkConfig) error {
	if nwCfg != nil {
		netmon := nwCfg.NetworkMonitor
		if netmon.Name == "" {
			return fmt.Errorf("NetworkMonitor is not set in conflist")
		}

		if netmon.Disable == true {
			log.Printf("Network Monitor is not enabled")
			return nil
		}

		cmd := fmt.Sprintf("pgrep -x %v", netmon.Name)
		out, err := platform.ExecuteShellCommand(cmd)
		if err != nil {
			log.Printf("Error running the cmd %v failed with %v", cmd, err)
			return err
		}

		if out != "" {
			pidSplit := strings.Split(out, "\n")
			if len(pidSplit) > 0 {
				log.Printf("Azure Network monitor is already running with pid %v", pidSplit[0])
				return nil
			} else {
				log.Printf("Out is not empty but split string is out : %v", out)
			}
		}

		cmd := fmt.Sprintf("/opt/cni/bin/%v -m %v", netmon.Name, netmon.MonitorAllChain)

		if netmon.Interval != 0 {
			cmd = fmt.Sprintf("%v -it %v", cmd, netmon.Interval)
		}

		out, err := platform.ExecuteShellCommand(cmd)
		if err != nil {
			log.Printf("Starting Network Monitor failed with error %v", err)
			return err
		}

		log.Printf("Azure Network Monitor started")
	}

	return nil
}
