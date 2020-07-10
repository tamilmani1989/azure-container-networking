package api

import (
	"fmt"
)

// NetworksResponse - Struct to hold the vnet information.
type NetworksResponse struct {
	NetworkInfos []NetworkInfo
}

type NetworkInfo struct {
	GUID string
}

func (n *NetworksResponse) String() string {
	networkInfos := "NetworkInfos:["
	if n.NetworkInfos != nil {
		for _, networkInfo := range n.NetworkInfos {
			networkInfos += fmt.Sprintf("{GUID: %s}", networkInfo.GUID)
		}
	}
	networkInfos += "]"

	return fmt.Sprintf(
		"NetworksResponse:[VnetIDs:%s]", networkInfos)
}
