// Copyright 2017 Microsoft. All rights reserved.
// MIT License

// +build linux

package restserver

const (
	defaultNetworkPluginName = "azure-vnet"
	overlayPluginName        = "azure-overlay"
)

func (service *httpRestService) getOSSpecificHostLocalIP(networkType string) (found bool, hostLocalIP string, errmsg string) {

	return false, "", ""
}
