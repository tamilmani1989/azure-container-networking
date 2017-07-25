// Copyright 2017 Microsoft. All rights reserved.
// MIT License

// +build windows

package restserver

import (
	"github.com/Azure/azure-container-networking/log"
)

const (
	defaultNetworkPluginName = "l2tunnel"
	overlayPluginName= "l2tunnel"
	hostLocalReservationID = "HostLocal"
)

func (service *httpRestService) getOSSpecificHostLocalIP(networkType string) (found bool, hostLocalIP string, errmsg string) {
	found = false
	switch networkType {
	case "Underlay":
		if service.imdsClient != nil {
			piface, err := service.imdsClient.GetPrimaryInterfaceInfoFromMemory()
			if err == nil {
				hostLocalIP = piface.PrimaryIP
				found = true
			} else {
				log.Printf("[Azure-CNS] Received error from GetPrimaryInterfaceInfoFromMemory. err: %v", err.Error())
			}
		}

	case "Overlay":
		errmsg = "[Azure-CNS] Overlay is not yet supported."
	}

	return found, hostLocalIP, errmsg

}
