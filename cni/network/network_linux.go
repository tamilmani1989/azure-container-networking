package network

import (
	"github.com/Azure/azure-container-networking/cns"
	"net"
	"strconv"
	"github.com/Azure/azure-container-networking/cni"
	"github.com/Azure/azure-container-networking/network"
	cniTypesCurr "github.com/containernetworking/cni/pkg/types/current"
	cniTypes "github.com/containernetworking/cni/pkg/types"
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
	if cnsNwConfig.MultiTenancyInfo.ID != 0 {
		vlanMap := make(map[string]interface{})
		vlanMap[network.VlanIDKey] = strconv.Itoa(cnsNwConfig.MultiTenancyInfo.ID)
		vlanMap[network.InternetBridgeIPKey] = cnsNwConfig.LocalIPConfiguration.GatewayIPAddress + "/" + strconv.Itoa(int(cnsNwConfig.LocalIPConfiguration.IPSubnet.PrefixLength))
		nwInfo.Options[dockerNetworkOption] = vlanMap
	}
}

func setEndpointOptions(cnsNwConfig *cns.GetNetworkContainerResponse, epInfo *network.EndpointInfo) {
	if cnsNwConfig.MultiTenancyInfo.ID != 0 {
		epInfo.Data[network.VlanIDKey] = cnsNwConfig.MultiTenancyInfo.ID
		epInfo.Data[network.LocalIPKey] = cnsNwConfig.LocalIPConfiguration.IPSubnet.IPAddress + "/" + strconv.Itoa(int(cnsNwConfig.LocalIPConfiguration.IPSubnet.PrefixLength))
		epInfo.Data[network.InternetBridgeIPKey] = cnsNwConfig.LocalIPConfiguration.GatewayIPAddress + "/" + strconv.Itoa(int(cnsNwConfig.LocalIPConfiguration.IPSubnet.PrefixLength))

	}
}