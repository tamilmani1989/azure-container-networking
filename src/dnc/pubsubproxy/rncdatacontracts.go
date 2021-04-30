// +build enablePubsubProxy

package pubsubproxy

import (
	"fmt"
	"strings"
	"sync"
)

// implementation of madariDataContracts interface. This struct
// does handling of vnet/subnet data that is published by RNM
type rncDataContracts struct {
	cachedVnetSubnetMap sync.Map
}

const (
	// PubSub paths
	rncVnetPartitionKeyFormat   = "/virtualnetworks/%s"
	rncSubnetRelativePathFormat = "/subnets/%s/config"
)

func (s *rncDataContracts) rehydrateSubscriptions() {
	// TODO: Add subnet subscription here
}

func (s *rncDataContracts) getCachedSubnetConfig(
	vnetId string,
	subnetId string) (cachedSubnetConfig, error) {

	key := fmt.Sprintf("%s_%s", vnetId, subnetId)
	config, found := s.cachedVnetSubnetMap.Load(key)
	if !found {
		return cachedSubnetConfig{}, fmt.Errorf("Configuration not found for VnetId: %s, Subnet:%s", vnetId, subnetId)
	}

	return config.(cachedSubnetConfig), nil
}

func (s *rncDataContracts) deleteCachedSubnetConfig(
	vnetId, subnetId string) error {
	key := fmt.Sprintf("%s_%s", vnetId, subnetId)
	s.cachedVnetSubnetMap.Delete(key)
	return nil
}

func (s *rncDataContracts) updateCachedSubnetConfig(
	vnetId string,
	subnetId string,
	subnetConfig *SubnetConfiguration) error {

	cachedSubnetConfig := cachedSubnetConfig{
		subnetName:    subnetConfig.SubnetName,
		addressPrefix: subnetConfig.AddressPrefix,
		expectedToken: subnetConfig.ExpectedToken,
		owningService: subnetConfig.OwningService,
	}

	key := fmt.Sprintf("%s_%s", vnetId, subnetId)
	s.cachedVnetSubnetMap.Store(key, cachedSubnetConfig)
	return nil
}

func (s *rncDataContracts) subscribeCachedSubnetConfig(
	vnetId, subnetId string) error {
	subnetConfig, err := s.getCachedSubnetConfig(vnetId, subnetId)
	if err == nil {
		// Mark the existing vnet as subscribed
		subnetConfig.isSubscribed = true
		s.cachedVnetSubnetMap.Store(vnetId, subnetConfig)
		return nil
	}

	// If subnet config doesn't exist, create an entry as a subscribed vnet
	key := fmt.Sprintf("%s_%s", vnetId, subnetId)
	s.cachedVnetSubnetMap.Store(key, cachedSubnetConfig{
		isSubscribed: true,
	})

	return nil
}

func (s *rncDataContracts) unsubscribeCachedSubnetConfigs() error {
	s.cachedVnetSubnetMap.Range(func(key interface{}, config interface{}) bool {
		mapKey := key.(string)
		if strings.Contains(mapKey, "_") {
			parts := strings.Split(mapKey, "_")
			subnetConfig, err := s.getCachedSubnetConfig(parts[0], parts[1])
			if err == nil {
				subnetConfig.isSubscribed = false
				s.cachedVnetSubnetMap.Store(mapKey, subnetConfig)
			}
		} else {
			// This is no-op for now. In future if we store other data
			// in this map, this can be implemented
			logDebugf("unsubscribeCachedSubnetConfigs does not support non-subnet paths")
		}
		return true
	})

	return nil
}

func (s *rncDataContracts) getSubnetPartitionKey(vnetId, subnetId string) string {
	return fmt.Sprintf(rncVnetPartitionKeyFormat, vnetId)
}

func (s *rncDataContracts) getSubnetRelativePath(vnetId, subnetId string) string {
	return fmt.Sprintf(rncSubnetRelativePathFormat, subnetId)
}