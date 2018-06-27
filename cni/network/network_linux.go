package network

import (
	"net"
	"strconv"
	"github.com/Azure/azure-container-networking/cni"
	"github.com/Azure/azure-container-networking/network"
	cniTypesCurr "github.com/containernetworking/cni/pkg/types/current"
	cniTypes "github.com/containernetworking/cni/pkg/types"
)

const (
	internetBridgeIP 	= "169.254.0.1"
)
// handleConsecutiveAdd is a dummy function for Linux platform.
func handleConsecutiveAdd(containerId, endpointId string, nwInfo *network.NetworkInfo, nwCfg *cni.NetworkConfig) (*cniTypesCurr.Result, error) {
	return nil, nil
}

func addDefaultRoute(epInfo *network.EndpointInfo, result *cniTypesCurr.Result) {
	_, defaultIPNet, _ := net.ParseCIDR("0.0.0.0/0")
	dstIP := net.IPNet{IP: net.ParseIP("0.0.0.0"), Mask: defaultIPNet.Mask}	
	gwIP := net.ParseIP(internetBridgeIP)
	epInfo.Routes = append(epInfo.Routes, network.RouteInfo{Dst: dstIP, Gw: gwIP, DevName: snatInterface})
	result.Routes = append(result.Routes, &cniTypes.Route{Dst: dstIP, GW: gwIP})
}

func setNetworkOptions(vlanid int, nwInfo *network.NetworkInfo) {
	if vlanid != 0 {
		vlanMap := make(map[string]interface{})
		vlanMap[network.VlanIDKey] = strconv.Itoa(vlanid)
		vlanMap[network.InternetBridgeIPKey] = internetBridgeIP
		nwInfo.Options[dockerNetworkOption] = vlanMap
	}
}

func setEndpointOptions(vlanid int, localIP string, epInfo *network.EndpointInfo) {
	epInfo.Data[network.VlanIDKey] = vlanid
	epInfo.Data[network.LocalIPKey] = localIP
}