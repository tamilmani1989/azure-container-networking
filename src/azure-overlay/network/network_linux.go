// Copyright 2017 Microsoft. All rights reserved.

// +build linux

package network

import (
	"fmt"
	"net"
	"strings"

	"github.com/Azure/azure-overlay/netlink"
)

const (
	// Prefix for bridge names.
	bridgePrefix = "azure"

	bridgeName = "azure-overlay"
	
	genericData = "com.docker.network.generic"
)

func (nm *networkManager) createOvsInterface(nwInfo *NetworkInfo) *externalInterface {

	subnet := nwInfo.Subnets[0].Prefix.String()
	gateway := nwInfo.Subnets[0].Gateway

	ipAddr := strings.Replace(subnet, "0/", "1/", 1)
	opt, _ := nwInfo.Options[genericData].(map[string]interface{})

	clusterInfo := opt["nodeList"].(string)

	//nodeList := opt["clusterInfo"].(string[])
	nodeList := strings.Split(clusterInfo, ",")

	fmt.Println("nodes", nodeList)

	var clusterNodeInfoList = make([]string, len(nodeList))

	i := 0
	for _, node := range nodeList {
		clusterNodeInfoList[i] = opt[node].(string)
		i = i + 1
	}

	fmt.Println("Nodeinfolist", clusterNodeInfoList)

	fmt.Println("ovs ip", ipAddr)
	system("ovs-vsctl", "add-br", bridgeName)
	system("ovs-vsctl", "set", "bridge", bridgeName, "stp_enable=true")
	system("ifconfig", bridgeName, ipAddr, "up")

	for index, node := range clusterNodeInfoList {
		vxlanIf := fmt.Sprintf("%s%d", "vxlan", index)
		nodeInfo := strings.Split(node, ",")
		optRemoteIp := fmt.Sprintf("%s:%s=%s", "options", "remote_ip", nodeInfo[0])
		system("ovs-vsctl", "add-port", bridgeName, vxlanIf, "--", "set", "interface", vxlanIf, "type=vxlan", optRemoteIp)
		system("ip", "route", "add", nodeInfo[1], "dev", bridgeName)
	}

	hostIf, err := net.InterfaceByName(bridgeName)
	if err != nil {
		system("ovs-vsctl", "del-br", bridgeName)
		return nil
	}

	extIf := externalInterface{
		Name:       bridgeName,
		Networks:   make(map[string]*network),
		MacAddress: hostIf.HardwareAddr,
		//IPv4Gateway: net.IPv4zero,
		IPv4Gateway: gateway,
		IPv6Gateway: net.IPv6unspecified,
	}
	extIf.Subnets = append(extIf.Subnets, subnet)

	nm.ExternalInterfaces[bridgeName] = &extIf
	return &extIf
}

// Linux implementation of route.
type route netlink.Route

// NewNetworkImpl creates a new container network.
func (nm *networkManager) newNetworkImpl(nwInfo *NetworkInfo) (*network, *externalInterface, error) {
	var extIf *externalInterface
	// Connect the external interface.
	switch nwInfo.Mode {
	case opModeOverlay:
		extIf = nm.createOvsInterface(nwInfo)
		fmt.Println("extif", extIf)
		if extIf == nil {
			return nil, nil, errOvsNotCreated
		}
	default:
		return nil, nil, errNetworkModeInvalid
	}

	// Create the network object.
	nw := &network{
		Id:        nwInfo.Id,
		Mode:      nwInfo.Mode,
		Endpoints: make(map[string]*endpoint),
		extIf:     extIf,
	}

	return nw, extIf, nil
}

// DeleteNetworkImpl deletes an existing container network.
func (nm *networkManager) deleteNetworkImpl(nw *network) error {
	// Delete the ovs bridge
	if len(nw.extIf.Networks) == 1 {
		system("ovs-vsctl", "del-br", bridgeName)
	}

	return nil
}
