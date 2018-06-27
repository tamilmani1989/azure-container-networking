// Copyright 2017 Microsoft. All rights reserved.
// MIT License

// +build linux

package network

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net"

	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/netlink"
)

const (
	// Common prefix for all types of host network interface names.
	commonInterfacePrefix = "az"

	// Prefix for host virtual network interface names.
	hostVEthInterfacePrefix = commonInterfacePrefix + "v"

	// Prefix for container network interface names.
	containerInterfacePrefix = "eth"
)

func generateVethName(key string) string {
	h := sha1.New()
	h.Write([]byte(key))
	return hex.EncodeToString(h.Sum(nil))[:11]
}

// newEndpointImpl creates a new endpoint in the network.
func (nw *network) newEndpointImpl(epInfo *EndpointInfo) (*endpoint, error) {
	var containerIf *net.Interface
	var ns *Namespace
	var ep *endpoint
	var err error
	var hostIfName string
	var contIfName string
	var epClient EndpointClient
	var vlanid int = 0
	var localIP string 

	if epInfo.Data != nil {
		if _, ok := epInfo.Data["vlanid"]; ok {
			vlanid = epInfo.Data["vlanid"].(int)
		}
		
		if _, ok := epInfo.Data["LocalIP"]; ok {
			localIP = epInfo.Data["LocalIP"].(string)
		}
	}

	if nw.Endpoints[epInfo.Id] != nil {
		log.Printf("[net] Endpoint alreday exists.")
		err = errEndpointExists
		return nil, err
	}

	if _, ok := epInfo.Data[OptVethName]; ok {
		log.Printf("Generate veth name based on the key provided")
		key := epInfo.Data[OptVethName].(string)
		vethname := generateVethName(key)
		hostIfName = fmt.Sprintf("%s%s", hostVEthInterfacePrefix, vethname)
		contIfName = fmt.Sprintf("%s%s2", hostVEthInterfacePrefix, vethname)
	} else {
		// Create a veth pair.
		log.Printf("Generate veth name based on endpoint id")
		hostIfName = fmt.Sprintf("%s%s", hostVEthInterfacePrefix, epInfo.Id[:7])
		contIfName = fmt.Sprintf("%s%s-2", hostVEthInterfacePrefix, epInfo.Id[:7])
	}

	if vlanid != 0 {
		epClient = NewOVSEndpointClient(nw.extIf.BridgeName, nw.extIf.Name, hostIfName, nw.extIf.MacAddress.String(), contIfName, localIP, vlanid, epInfo.EnableSnatOnHost)
	} else {
		epClient = NewLinuxBridgeEndpointClient(nw.extIf.BridgeName, nw.extIf.Name, hostIfName, contIfName, nw.extIf.MacAddress, nw.Mode)
	}

	if err := epClient.AddEndpoints(epInfo); err != nil {
		return nil, err
	}

	// On failure, delete the veth pair.
	defer func() {
		if err != nil {
			netlink.DeleteLink(contIfName)
		}
	}()

	containerIf, err = net.InterfaceByName(contIfName)
	if err != nil {
		return nil, err
	}

	// Setup rules for IP addresses on the container interface.
	epClient.AddEndpointRules(epInfo)

	// If a network namespace for the container interface is specified...
	if epInfo.NetNsPath != "" {
		// Open the network namespace.
		log.Printf("[net] Opening netns %v.", epInfo.NetNsPath)
		ns, err = OpenNamespace(epInfo.NetNsPath)
		if err != nil {
			return nil, err
		}
		defer ns.Close()

		if err := epClient.MoveEndpointsToContainerNS(epInfo, ns.GetFd()); err != nil {
			return nil, err
		}

		// Enter the container network namespace.
		log.Printf("[net] Entering netns %v.", epInfo.NetNsPath)
		if err := ns.Enter(); err != nil {
			return nil, err
		}

		// Return to host network namespace.
		defer func() {
			log.Printf("[net] Exiting netns %v.", epInfo.NetNsPath)
			err = ns.Exit()
			if err != nil {
				log.Printf("[net] Failed to exit netns, err:%v.", err)
			}
		}()
	}

	// If a name for the container interface is specified...
	if epInfo.IfName != "" {
		if err := epClient.SetupContainerInterfaces(epInfo); err != nil {
			return nil, err
		}
	}

	if err := epClient.ConfigureContainerInterfacesAndRoutes(epInfo); err != nil {
		return nil, err
	}

	// Create the endpoint object.
	ep = &endpoint{
		Id:          epInfo.Id,
		IfName:      contIfName,
		HostIfName:  hostIfName,
		MacAddress:  containerIf.HardwareAddr,
		IPAddresses: epInfo.IPAddresses,
		Gateways:    []net.IP{nw.extIf.IPv4Gateway},
		DNS:         epInfo.DNS,
		VlanID:      vlanid,
		EnableSnatOnHost: epInfo.EnableSnatOnHost,
	}

	for _, route := range epInfo.Routes {
		ep.Routes = append(ep.Routes, route)
	}

	return ep, nil
}

// deleteEndpointImpl deletes an existing endpoint from the network.
func (nw *network) deleteEndpointImpl(ep *endpoint) error {
	var epClient EndpointClient

	// Delete the veth pair by deleting one of the peer interfaces.
	// Deleting the host interface is more convenient since it does not require
	// entering the container netns and hence works both for CNI and CNM.
	if ep.VlanID != 0 {
		epClient = NewOVSEndpointClient(nw.extIf.BridgeName, nw.extIf.Name, ep.HostIfName, nw.extIf.MacAddress.String(), "", "", ep.VlanID, ep.EnableSnatOnHost)
	} else {
		epClient = NewLinuxBridgeEndpointClient(nw.extIf.BridgeName, nw.extIf.Name, ep.HostIfName, "", nw.extIf.MacAddress, nw.Mode)
	}

	epClient.DeleteEndpointRules(ep)
	epClient.DeleteEndpoints(ep)

	return nil
}

// getInfoImpl returns information about the endpoint.
func (ep *endpoint) getInfoImpl(epInfo *EndpointInfo) {
}
