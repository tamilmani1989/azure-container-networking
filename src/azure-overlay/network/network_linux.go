// Copyright 2017 Microsoft. All rights reserved.

// +build linux

package network

import (
	"fmt"
	"log"
	"net"
	"strings"

	"azure-overlay/common"
	"azure-overlay/netlink"
)

const (
	// Prefix for bridge names.
	bridgePrefix = "azure"

	bridgeName = "azure-overlay"

	genericData = "com.docker.network.generic"

	AzureOvsCommonBridge = "azure-ovs"
)

func (nm *networkManager) createOvsInterface(nwInfo *NetworkInfo) *externalInterface {

	if brName, ok := nm.Bridges[nwInfo.Id]; ok {
		log.Printf("Bridge %v already exists for network id %v. Nothing to do.", brName, nwInfo)
		return nil
	}

	brName := fmt.Sprintf("%s%d", bridgeName, nm.BridgeCount+1)
	subnet := nwInfo.Subnets[0].Prefix.String()
	gateway := nwInfo.Subnets[0].Gateway
	// convert CIDR notation into Gateway address as needed by plugin.
	ipAddr := strings.Replace(subnet, "0/", "1/", 1)
	opt, _ := nwInfo.Options[genericData].(map[string]interface{})
	clusterInfo := opt["nodeList"].(string)
	vxlanID := opt["vxlanID"].(string)
	nodeList := strings.Split(clusterInfo, ",")
	log.Printf("nodes %v", nodeList)

	netnsName := "netns" + brName
	err := common.System("ip", "netns", "add", netnsName)
	if err != nil {
		log.Printf("Error while creating network namespace %v", err.Error())
		return nil
	}

	var clusterNodeInfoList = make([]string, len(nodeList))
	i := 0

	for _, node := range nodeList {
		clusterNodeInfoList[i] = opt[node].(string)
		i = i + 1
	}

	log.Printf("Nodeinfolist %v", clusterNodeInfoList)
	log.Printf("ovs ip %v name %v", ipAddr, brName)

	err = common.System("ovs-vsctl", "add-br", brName)
	if err != nil {
		log.Printf("Error while creating overalay bridge for network %v", err.Error())
		return nil
	}

	hostIf, err := net.InterfaceByName(brName)
	if err != nil {
		common.System("ovs-vsctl", "del-br", brName)
		return nil
	}

	err = common.System("ovs-vsctl", "set", "bridge", brName, "stp_enable=true")
	if err != nil {
		log.Printf("Error while enabling stp on ovs bridge %v", err.Error())
		return nil
	}

	for _, node := range clusterNodeInfoList {
		log.Printf("VxlanCount %d", nm.VxlanCount)
		vxlanIf := fmt.Sprintf("%s%d", "vxlan", nm.VxlanCount)
		nodeInfo := strings.Split(node, ",")
		optRemoteIp := fmt.Sprintf("%s:%s=%s", "options", "remote_ip", nodeInfo[0])
		optVxLANID := fmt.Sprintf("%s:%s=%s", "options", "key", vxlanID)
		// options:key=<VNI>
		err = common.System("ovs-vsctl", "add-port", brName, vxlanIf, "--", "set", "interface", vxlanIf, "type=vxlan", optRemoteIp, optVxLANID)
		if err != nil {
			log.Printf("Error while adding vxlan on ovs bridge %v", err.Error())
			return nil
		}
		nm.VxlanCount++
	}

	extIf := externalInterface{
		Name:        brName,
		Networks:    make(map[string]*network),
		MacAddress:  hostIf.HardwareAddr,
		IPv4Gateway: gateway,
		IPv6Gateway: net.IPv6unspecified,
	}

	vethSuffix := strings.Split(brName, "-")

	log.Printf("Create and add veth pair to connect to common ovs, vethsuffix0 %v vethsuffix1 %v ", vethSuffix[0], vethSuffix[1])
	err = common.System("ip", "link", "add", "v0"+vethSuffix[1], "type", "veth", "peer", "name", "v1"+vethSuffix[1])
	if err != nil {
		log.Printf("Error while creating veth pair to connect to common OVS bridge %v", err.Error())
		return nil
	}

	err = common.System("ovs-vsctl", "add-port", AzureOvsCommonBridge, "v1"+vethSuffix[1])
	if err != nil {
		log.Printf("Error while adding veth pair to common OVS %v", err.Error())
		return nil
	}

	err = common.System("ovs-vsctl", "add-port", brName, "v0"+vethSuffix[1])
	if err != nil {
		log.Printf("Error while adding veth to overlay network bridge %v", err.Error())
		return nil
	}

	log.Printf("Assigning %v to new ns ", brName)
	err = common.System("ip", "link", "set", brName, "netns", netnsName)
	if err != nil {
		log.Printf("Error while moving ovs bridge to new network namespace %v", err.Error())
		return nil
	}

	err = common.System("ip", "netns", "exec", netnsName, "ifconfig", brName, ipAddr, "up")
	if err != nil {
		log.Printf("Error assigning ip address to ovs bridge %v", err.Error())
		return nil
	}

	log.Printf("Add default gw in new namespace %v ", gateway.String())
	err = common.System("ip", "netns", "exec", netnsName, "route", "add", "default", "gw", gateway.String(), "dev", brName)
	if err != nil {
		log.Printf("Error adding default gw for new network namespace %v", err.Error())
		return nil
	}

	extIf.Subnets = append(extIf.Subnets, subnet)
	nm.ExternalInterfaces[brName] = &extIf
	nm.Bridges[nwInfo.Id] = brName
	//TODO: Handle overflow and unique bridge names
	nm.BridgeCount = nm.BridgeCount + 1
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
		err := common.System("ovs-vsctl", "del-br", nm.Bridges[nw.Id])
		if err != nil {
			log.Printf("Error deleting overlay bridge %v", err.Error())
			return err
		}

		log.Printf("Delete network namespace")

		err = common.System("ip", "netns", "delete", "netns"+nm.Bridges[nw.Id])
		if err != nil {
			log.Printf("Error deleting network namespace %v", err.Error())
			return err
		}

		vethSuffix := strings.Split(nm.Bridges[nw.Id], "-")

		err = common.System("ip", "link", "del", "dev", "v1"+vethSuffix[1])
		if err != nil {
			log.Printf("Error removing veth pair which connects to common ovs %v", err.Error())
			return err
		}

		delete(nm.Bridges, nw.Id) // does not return anything
	}

	return nil
}
