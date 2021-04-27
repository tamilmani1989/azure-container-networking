// +build enablePubsubProxy

package pubsubproxy

import (
	"dnc/datamodel"
	"time"
)

// implementation of madariDataContracts interface. This struct
// does handling of vnet/subnet data that is published by RNM
type rnmDataContracts struct {
}

func (r *rnmDataContracts) rehydrateSubscriptions()  {
	logDebugf("Rehydrating vnet subscriptions")
	for {
		networksMap, err := networkRepo.GetAllNetworkIDs()
		if err == nil {
			for networkKey := range networksMap {
				networkID := datamodel.GetNetworkIdFromKey(networkKey)
				RegisterVnet(networkID)
			}
			// Exit after rehydrating to all vnet subscriptions
			return
		} else {
			logErrorf("Failed to rehydrate vnets. Error: %v", err)
			time.Sleep(time.Second * 1)
		}
	}
}